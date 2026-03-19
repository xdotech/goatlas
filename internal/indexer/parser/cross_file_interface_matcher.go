package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CrossFileInterfaceMatcher performs cross-file interface implementation
// detection by loading all interfaces and method sets from PostgreSQL,
// then matching across files. This is a second pass after per-file extraction.
type CrossFileInterfaceMatcher struct {
	pool   *pgxpool.Pool
	iiRepo domain.InterfaceImplRepository
}

// NewCrossFileInterfaceMatcher creates a CrossFileInterfaceMatcher.
func NewCrossFileInterfaceMatcher(pool *pgxpool.Pool, iiRepo domain.InterfaceImplRepository) *CrossFileInterfaceMatcher {
	return &CrossFileInterfaceMatcher{pool: pool, iiRepo: iiRepo}
}

// interfaceDef holds an interface's methods from the symbols table.
type interfaceDef struct {
	name      string // qualified name, e.g. "domain.SymbolRepository"
	fileID    int64
	methods   []string
	pkgPrefix string
}

// structMethodSet holds methods defined on a struct receiver.
type structMethodSet struct {
	receiver string // e.g. "postgres.(SymbolRepo)"
	fileID   int64
	methods  map[string]bool
}

// MatchAll detects cross-file interface implementations for a repository.
// Returns the number of new interface_impl records created.
func (m *CrossFileInterfaceMatcher) MatchAll(ctx context.Context, repoID int64) (int, error) {
	// 1. Load all interface definitions from symbols table
	interfaces, err := m.loadInterfaces(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("load interfaces: %w", err)
	}
	if len(interfaces) == 0 {
		return 0, nil
	}

	// 2. Load all struct method sets
	methodSets, err := m.loadStructMethods(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("load struct methods: %w", err)
	}

	// 3. Match: for each interface, find structs implementing ALL methods
	//    but in DIFFERENT files (same-file already handled by per-file extractor)
	var newImpls []domain.InterfaceImpl
	for _, iface := range interfaces {
		for _, ms := range methodSets {
			// Skip same-file matches (already detected)
			if ms.fileID == iface.fileID {
				continue
			}

			// Check if struct has ALL interface methods
			matchCount := 0
			for _, method := range iface.methods {
				if ms.methods[method] {
					matchCount++
				}
			}

			if matchCount == len(iface.methods) {
				// All methods matched cross-file → record with 0.75 confidence
				for _, method := range iface.methods {
					newImpls = append(newImpls, domain.InterfaceImpl{
						FileID:        ms.fileID,
						InterfaceName: iface.name,
						StructName:    ms.receiver,
						MethodName:    method,
						Confidence:    0.75, // cross-file, all methods matched
					})
				}
			}
		}

		// Cap to prevent explosion
		if len(newImpls) > 5000 {
			break
		}
	}

	if len(newImpls) == 0 {
		return 0, nil
	}

	// 4. Persist new cross-file matches
	if err := m.iiRepo.BulkInsert(ctx, newImpls); err != nil {
		return 0, fmt.Errorf("insert cross-file impls: %w", err)
	}

	return len(newImpls), nil
}

// loadInterfaces loads interface definitions with their method lists.
func (m *CrossFileInterfaceMatcher) loadInterfaces(ctx context.Context, repoID int64) ([]interfaceDef, error) {
	rows, err := m.pool.Query(ctx, `
		SELECT s.qualified_name, s.file_id
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		JOIN repositories r ON f.repo_id = r.id
		WHERE r.id = $1 AND s.kind = 'interface'
		ORDER BY s.qualified_name
		LIMIT 500
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type ifaceKey struct {
		name   string
		fileID int64
	}
	var ifaceNames []ifaceKey
	for rows.Next() {
		var ik ifaceKey
		if err := rows.Scan(&ik.name, &ik.fileID); err != nil {
			return nil, err
		}
		ifaceNames = append(ifaceNames, ik)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// For each interface, find its methods from the same-file symbols
	var interfaces []interfaceDef
	for _, ik := range ifaceNames {
		methods, err := m.loadInterfaceMethods(ctx, ik.name, ik.fileID)
		if err != nil || len(methods) == 0 {
			continue
		}
		pkgPrefix := ""
		if idx := strings.LastIndex(ik.name, "."); idx > 0 {
			pkgPrefix = ik.name[:idx]
		}
		interfaces = append(interfaces, interfaceDef{
			name:      ik.name,
			fileID:    ik.fileID,
			methods:   methods,
			pkgPrefix: pkgPrefix,
		})
	}
	return interfaces, nil
}

// loadInterfaceMethods finds methods belonging to an interface by looking at
// existing same-file interface_impls records (which captured the interface's method names).
func (m *CrossFileInterfaceMatcher) loadInterfaceMethods(ctx context.Context, ifaceName string, fileID int64) ([]string, error) {
	rows, err := m.pool.Query(ctx, `
		SELECT DISTINCT method_name
		FROM interface_impls
		WHERE interface_name = $1 AND file_id = $2
		ORDER BY method_name
	`, ifaceName, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var methods []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		methods = append(methods, name)
	}
	return methods, rows.Err()
}

// loadStructMethods loads all struct method sets from the symbols table.
func (m *CrossFileInterfaceMatcher) loadStructMethods(ctx context.Context, repoID int64) ([]structMethodSet, error) {
	rows, err := m.pool.Query(ctx, `
		SELECT s.qualified_name, s.file_id, s.name
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		JOIN repositories r ON f.repo_id = r.id
		WHERE r.id = $1 AND s.kind = 'method'
		ORDER BY s.qualified_name
		LIMIT 10000
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group methods by receiver
	receiverMethods := make(map[string]*structMethodSet)
	for rows.Next() {
		var qualName string
		var fileID int64
		var methodName string
		if err := rows.Scan(&qualName, &fileID, &methodName); err != nil {
			return nil, err
		}

		// Extract receiver from qualified name: "pkg.(Receiver).Method" → "pkg.(Receiver)"
		receiver := extractReceiver(qualName)
		if receiver == "" {
			continue
		}

		key := fmt.Sprintf("%s@%d", receiver, fileID)
		ms, ok := receiverMethods[key]
		if !ok {
			ms = &structMethodSet{
				receiver: receiver,
				fileID:   fileID,
				methods:  make(map[string]bool),
			}
			receiverMethods[key] = ms
		}
		ms.methods[methodName] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []structMethodSet
	for _, ms := range receiverMethods {
		result = append(result, *ms)
	}
	return result, nil
}

// extractReceiver extracts the receiver part from a qualified name.
// "pkg.(Receiver).Method" → "pkg.(Receiver)"
func extractReceiver(qualifiedName string) string {
	// Look for pattern: something.(Something).something
	parenStart := strings.Index(qualifiedName, ".(")
	if parenStart < 0 {
		return ""
	}
	parenEnd := strings.Index(qualifiedName[parenStart:], ").")
	if parenEnd < 0 {
		return ""
	}
	return qualifiedName[:parenStart+parenEnd+1]
}
