package usecase

import (
	"context"
	"fmt"

	"github.com/goatlas/goatlas/internal/graph"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BuildGraphUseCase builds the Neo4j knowledge graph via MCP.
type BuildGraphUseCase struct {
	graphClient *graph.Client
	pool        *pgxpool.Pool
}

// NewBuildGraphUseCase creates a new BuildGraphUseCase.
func NewBuildGraphUseCase(graphClient *graph.Client, pool *pgxpool.Pool) *BuildGraphUseCase {
	return &BuildGraphUseCase{graphClient: graphClient, pool: pool}
}

// Execute builds the knowledge graph from indexed data.
func (uc *BuildGraphUseCase) Execute(ctx context.Context) (string, error) {
	if uc.graphClient == nil {
		return "", fmt.Errorf("Neo4j not configured — cannot build graph")
	}

	builder := graph.NewBuilder(uc.pool, uc.graphClient)
	result, err := builder.Build(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Knowledge graph built successfully\n  Package nodes: %d\n  File nodes:    %d\n  Function nodes:%d\n  Type nodes:    %d\n  Import edges:  %d",
		result.PackageNodes,
		result.FileNodes,
		result.FunctionNodes,
		result.TypeNodes,
		result.ImportEdges,
	), nil
}
