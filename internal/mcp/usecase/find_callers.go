package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/xdotech/goatlas/internal/graph"
	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// FindCallersUseCase finds symbols that reference a given function name.
// It also resolves interface methods to their concrete implementations.
type FindCallersUseCase struct {
	symbolRepo domain.SymbolRepository
	iiRepo     domain.InterfaceImplRepository
	querier    *graph.Querier // optional: used when Neo4j graph is available
}

// NewFindCallersUseCase constructs a FindCallersUseCase.
func NewFindCallersUseCase(sr domain.SymbolRepository, iiRepo domain.InterfaceImplRepository, querier *graph.Querier) *FindCallersUseCase {
	return &FindCallersUseCase{symbolRepo: sr, iiRepo: iiRepo, querier: querier}
}

// Execute searches for references/callers of the given function name.
// When a Neo4j graph querier is available, it uses graph-based CALLS traversal.
// depth controls how deep to traverse (default: 5). repo restricts to one repo (optional).
func (uc *FindCallersUseCase) Execute(ctx context.Context, functionName string, depth int, repo string) (string, error) {
	if depth <= 0 {
		depth = 5
	}

	// Prefer graph-based callers when Neo4j is available
	if uc.querier != nil {
		callers, err := uc.querier.FindCallers(ctx, functionName, depth, 0.0, repo)
		if err == nil {
			return uc.querier.FormatCallers(functionName, callers), nil
		}
		// Log and fall back to postgres on transient graph error
		log.Printf("WARN find_callers: graph query failed (%v), falling back to postgres symbol search", err)
	}

	// Postgres fallback: resolve interface methods → concrete implementations
	searchNames := []string{functionName}

	if uc.iiRepo != nil {
		interfaceName, methodName := splitInterfaceMethod(functionName)
		if methodName != "" {
			impls, err := uc.iiRepo.FindImplementations(ctx, interfaceName, methodName)
			if err == nil && len(impls) > 0 {
				for _, impl := range impls {
					searchNames = append(searchNames, impl.StructName+"."+impl.MethodName)
				}
			}
		}
	}

	var allSymbols []domain.Symbol
	seen := make(map[string]bool)

	for _, name := range searchNames {
		symbols, err := uc.symbolRepo.Search(ctx, name, 30, "")
		if err != nil {
			return "", err
		}
		for _, s := range symbols {
			key := fmt.Sprintf("%s:%d", s.QualifiedName, s.Line)
			if !seen[key] {
				seen[key] = true
				allSymbols = append(allSymbols, s)
			}
		}
	}

	if len(allSymbols) == 0 {
		return fmt.Sprintf("No callers found for %q\n(Tip: run build_graph to enable graph-based caller search)", functionName), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name matches for %q (no graph — run build_graph for true callers)", functionName))
	if len(searchNames) > 1 {
		sb.WriteString(fmt.Sprintf(" (+ %d concrete implementations)", len(searchNames)-1))
	}
	sb.WriteString(":\n\n")
	for _, s := range allSymbols {
		sb.WriteString(fmt.Sprintf("  [%s] %s (line %d)\n", s.Kind, s.QualifiedName, s.Line))
	}
	return sb.String(), nil
}

// splitInterfaceMethod splits "InterfaceName.MethodName" or just "MethodName".
func splitInterfaceMethod(name string) (interfaceName, methodName string) {
	// Handle qualified names like "pkg.InterfaceName.Method"
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "", name
}
