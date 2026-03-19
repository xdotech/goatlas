package graph

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProcessDetector detects entry points and traces execution flows forward.
type ProcessDetector struct {
	client    *Client
	pool      *pgxpool.Pool
	processRepo domain.ProcessRepository
}

// NewProcessDetector constructs a ProcessDetector.
func NewProcessDetector(pool *pgxpool.Pool, client *Client, processRepo domain.ProcessRepository) *ProcessDetector {
	return &ProcessDetector{pool: pool, client: client, processRepo: processRepo}
}

// EntryPoint represents a detected entry point in the codebase.
type EntryPoint struct {
	QualifiedName string
	Name          string
	FilePath      string
	Line          int
	Kind          string // "http_handler", "kafka_consumer", "main", "cli"
}

// DetectAll detects all processes for a repository and persists them.
func (d *ProcessDetector) DetectAll(ctx context.Context, repoID int64) (int, error) {
	// Clear previous results
	if err := d.processRepo.DeleteByRepoID(ctx, repoID); err != nil {
		return 0, fmt.Errorf("clear old processes: %w", err)
	}

	entries, err := d.DetectEntryPoints(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("detect entry points: %w", err)
	}

	count := 0
	for _, ep := range entries {
		steps, err := d.TraceForward(ctx, ep.QualifiedName, 15)
		if err != nil {
			continue // skip broken chains
		}
		if len(steps) == 0 {
			continue
		}

		processName := d.deriveProcessName(ep)
		proc := &domain.Process{
			RepoID:     repoID,
			Name:       processName,
			EntryPoint: ep.QualifiedName,
			FilePath:   ep.FilePath,
		}
		procID, err := d.processRepo.Insert(ctx, proc)
		if err != nil {
			continue
		}

		domainSteps := make([]domain.ProcessStep, len(steps))
		for i, s := range steps {
			domainSteps[i] = domain.ProcessStep{
				ProcessID:  procID,
				StepOrder:  i + 1,
				SymbolName: s.QualifiedName,
				FilePath:   s.FilePath,
				Line:       s.Line,
			}
		}
		if err := d.processRepo.InsertSteps(ctx, domainSteps); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// DetectEntryPoints finds all entry points in the repository.
func (d *ProcessDetector) DetectEntryPoints(ctx context.Context, repoID int64) ([]EntryPoint, error) {
	var entries []EntryPoint

	// 1. HTTP handlers from api_endpoints table
	httpEntries, err := d.detectHTTPHandlers(ctx, repoID)
	if err == nil {
		entries = append(entries, httpEntries...)
	}

	// 2. Kafka consumers from service_connections table
	kafkaEntries, err := d.detectKafkaConsumers(ctx, repoID)
	if err == nil {
		entries = append(entries, kafkaEntries...)
	}

	// 3. main() functions
	mainEntries, err := d.detectMainFunctions(ctx, repoID)
	if err == nil {
		entries = append(entries, mainEntries...)
	}

	return entries, nil
}

func (d *ProcessDetector) detectHTTPHandlers(ctx context.Context, repoID int64) ([]EntryPoint, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT ae.handler_name, f.path, ae.line
		FROM api_endpoints ae
		JOIN files f ON ae.file_id = f.id
		WHERE f.repo_id = $1
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	var entries []EntryPoint
	for rows.Next() {
		var handlerName, filePath string
		var line int
		if err := rows.Scan(&handlerName, &filePath, &line); err != nil {
			return nil, err
		}

		// Try to find the qualified name for this handler
		qname := d.resolveQualifiedName(ctx, handlerName, filePath)
		if qname == "" {
			qname = handlerName
		}
		if _, ok := seen[qname]; ok {
			continue
		}
		seen[qname] = struct{}{}

		entries = append(entries, EntryPoint{
			QualifiedName: qname,
			Name:          handlerName,
			FilePath:      filePath,
			Line:          line,
			Kind:          "http_handler",
		})
	}
	return entries, rows.Err()
}

func (d *ProcessDetector) detectKafkaConsumers(ctx context.Context, repoID int64) ([]EntryPoint, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT sc.target, f.path, sc.line
		FROM service_connections sc
		LEFT JOIN files f ON sc.file_id = f.id
		WHERE sc.repo_id = $1 AND sc.conn_type = 'kafka_consume'
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EntryPoint
	for rows.Next() {
		var target string
		var filePath *string
		var line int
		if err := rows.Scan(&target, &filePath, &line); err != nil {
			return nil, err
		}
		fp := ""
		if filePath != nil {
			fp = *filePath
		}
		entries = append(entries, EntryPoint{
			QualifiedName: "kafka:" + target,
			Name:          target,
			FilePath:      fp,
			Line:          line,
			Kind:          "kafka_consumer",
		})
	}
	return entries, rows.Err()
}

func (d *ProcessDetector) detectMainFunctions(ctx context.Context, repoID int64) ([]EntryPoint, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT s.qualified_name, f.path, s.line
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE f.repo_id = $1 AND s.name = 'main' AND s.kind = 'func'
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EntryPoint
	for rows.Next() {
		var qname, filePath string
		var line int
		if err := rows.Scan(&qname, &filePath, &line); err != nil {
			return nil, err
		}
		entries = append(entries, EntryPoint{
			QualifiedName: qname,
			Name:          "main",
			FilePath:      filePath,
			Line:          line,
			Kind:          "main",
		})
	}
	return entries, rows.Err()
}

// resolveQualifiedName attempts to find the qualified name for a handler.
func (d *ProcessDetector) resolveQualifiedName(ctx context.Context, handlerName, filePath string) string {
	var qname string
	_ = d.pool.QueryRow(ctx, `
		SELECT s.qualified_name FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE s.name = $1 AND f.path = $2
		LIMIT 1
	`, handlerName, filePath).Scan(&qname)
	return qname
}

// TraceForward performs a BFS from the entry point through CALLS edges.
func (d *ProcessDetector) TraceForward(ctx context.Context, entryQName string, maxDepth int) ([]ProcessStepResult, error) {
	if d.client == nil {
		return d.traceForwardPG(ctx, entryQName, maxDepth)
	}
	return d.traceForwardNeo4j(ctx, entryQName, maxDepth)
}

// ProcessStepResult holds one step from a forward trace.
type ProcessStepResult struct {
	QualifiedName string
	Name          string
	FilePath      string
	Line          int
	Depth         int
}

func (d *ProcessDetector) traceForwardNeo4j(ctx context.Context, entryQName string, maxDepth int) ([]ProcessStepResult, error) {
	records, err := d.client.QueryNodes(ctx, fmt.Sprintf(`
		MATCH path = (entry:Function)-[:CALLS*1..%d]->(callee:Function)
		WHERE entry.qualified = $entry OR entry.name = $entry
		WITH callee, min(length(path)) AS depth
		RETURN callee.qualified AS qualified, callee.name AS name, depth
		ORDER BY depth, qualified
		LIMIT 100
	`, maxDepth), map[string]any{"entry": entryQName})
	if err != nil {
		return nil, err
	}

	// Start with the entry point itself
	steps := []ProcessStepResult{{QualifiedName: entryQName, Depth: 0}}
	for _, r := range records {
		step := ProcessStepResult{}
		if v, ok := r["qualified"].(string); ok {
			step.QualifiedName = v
		}
		if v, ok := r["name"].(string); ok {
			step.Name = v
		}
		if v, ok := r["depth"].(int64); ok {
			step.Depth = int(v)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

// traceForwardPG is a fallback using Postgres function_calls table when Neo4j is unavailable.
func (d *ProcessDetector) traceForwardPG(ctx context.Context, entryQName string, maxDepth int) ([]ProcessStepResult, error) {
	visited := map[string]int{} // qname -> depth
	queue := []struct {
		qname string
		depth int
	}{{entryQName, 0}}
	visited[entryQName] = 0

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.depth >= maxDepth {
			continue
		}

		rows, err := d.pool.Query(ctx, `
			SELECT DISTINCT callee_name FROM function_calls
			WHERE caller_qualified_name = $1
		`, curr.qname)
		if err != nil {
			continue
		}

		var callees []string
		for rows.Next() {
			var callee string
			if err := rows.Scan(&callee); err != nil {
				continue
			}
			callees = append(callees, callee)
		}
		rows.Close()

		for _, callee := range callees {
			// Try to resolve qualified name
			var resolved string
			_ = d.pool.QueryRow(ctx, `
				SELECT qualified_name FROM symbols WHERE name = $1 LIMIT 1
			`, callee).Scan(&resolved)
			if resolved == "" {
				resolved = callee
			}
			if _, ok := visited[resolved]; !ok {
				visited[resolved] = curr.depth + 1
				queue = append(queue, struct {
					qname string
					depth int
				}{resolved, curr.depth + 1})
			}
		}
	}

	steps := make([]ProcessStepResult, 0, len(visited))
	for qname, depth := range visited {
		steps = append(steps, ProcessStepResult{QualifiedName: qname, Depth: depth})
	}
	// Sort by depth
	for i := 0; i < len(steps)-1; i++ {
		for j := i + 1; j < len(steps); j++ {
			if steps[j].Depth < steps[i].Depth {
				steps[i], steps[j] = steps[j], steps[i]
			}
		}
	}
	return steps, nil
}

// deriveProcessName generates a human-readable process name from an entry point.
func (d *ProcessDetector) deriveProcessName(ep EntryPoint) string {
	switch ep.Kind {
	case "http_handler":
		// Use handler name, cleaned up
		name := ep.Name
		name = strings.TrimSuffix(name, "Handler")
		name = strings.TrimSuffix(name, "Handle")
		if name == "" {
			name = ep.Name
		}
		return fmt.Sprintf("HTTP: %s", name)
	case "kafka_consumer":
		return fmt.Sprintf("Kafka: %s", ep.Name)
	case "main":
		dir := filepath.Dir(ep.FilePath)
		base := filepath.Base(dir)
		return fmt.Sprintf("Main: %s", base)
	default:
		return ep.Name
	}
}
