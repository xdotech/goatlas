package usecase

import (
	"context"

	"github.com/goatlas/goatlas/internal/graph"
)

// GetServiceDepsUseCase queries Neo4j for a service's imported packages.
type GetServiceDepsUseCase struct {
	querier *graph.Querier
}

// NewGetServiceDepsUseCase constructs a GetServiceDepsUseCase.
// querier may be nil when Neo4j is not configured.
func NewGetServiceDepsUseCase(q *graph.Querier) *GetServiceDepsUseCase {
	return &GetServiceDepsUseCase{querier: q}
}

// Execute returns a formatted list of packages imported by the given service.
func (uc *GetServiceDepsUseCase) Execute(ctx context.Context, service string) (string, error) {
	if uc.querier == nil {
		return "Graph database not available. Run 'goatlas build-graph' with Neo4j configured.", nil
	}
	deps, err := uc.querier.GetServiceDependencies(ctx, service)
	if err != nil {
		return "Graph query failed: " + err.Error(), nil
	}
	return uc.querier.FormatServiceDependencies(service, deps), nil
}
