package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/graph"
)

// GetAPIHandlersUseCase queries Neo4j for functions matching an endpoint pattern.
type GetAPIHandlersUseCase struct {
	querier *graph.Querier
}

// NewGetAPIHandlersUseCase constructs a GetAPIHandlersUseCase.
// querier may be nil when Neo4j is not configured.
func NewGetAPIHandlersUseCase(q *graph.Querier) *GetAPIHandlersUseCase {
	return &GetAPIHandlersUseCase{querier: q}
}

// Execute returns a formatted list of handler functions matching the pattern.
func (uc *GetAPIHandlersUseCase) Execute(ctx context.Context, endpointPattern string) (string, error) {
	if uc.querier == nil {
		return "Graph database not available. Run 'goatlas build-graph' with Neo4j configured.", nil
	}
	results, err := uc.querier.GetAPIHandlers(ctx, endpointPattern)
	if err != nil {
		return "Graph query failed: " + err.Error(), nil
	}
	if len(results) == 0 {
		return fmt.Sprintf("No handlers found for pattern: %s", endpointPattern), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("API handlers for %q:\n\n", endpointPattern))
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("  %s %s -> %s\n", r.Method, r.Path, r.HandlerName))
	}
	return sb.String(), nil
}
