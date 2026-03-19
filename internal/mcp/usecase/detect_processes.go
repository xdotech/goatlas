package usecase

import (
	"context"
	"fmt"

	"github.com/xdotech/goatlas/internal/graph"
	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DetectProcessesUseCase triggers process and community detection.
type DetectProcessesUseCase struct {
	pool          *pgxpool.Pool
	graphClient   *graph.Client
	processRepo   domain.ProcessRepository
	communityRepo domain.CommunityRepository
}

// NewDetectProcessesUseCase creates a new DetectProcessesUseCase.
func NewDetectProcessesUseCase(
	pool *pgxpool.Pool,
	graphClient *graph.Client,
	processRepo domain.ProcessRepository,
	communityRepo domain.CommunityRepository,
) *DetectProcessesUseCase {
	return &DetectProcessesUseCase{
		pool:          pool,
		graphClient:   graphClient,
		processRepo:   processRepo,
		communityRepo: communityRepo,
	}
}

// Execute detects processes and communities for the first indexed repo.
func (uc *DetectProcessesUseCase) Execute(ctx context.Context) (string, error) {
	repoID, err := getFirstRepoID(ctx, uc.pool)
	if err != nil {
		return "", fmt.Errorf("no indexed repository found: %w", err)
	}

	// Detect processes
	pd := graph.NewProcessDetector(uc.pool, uc.graphClient, uc.processRepo)
	processCount, err := pd.DetectAll(ctx, repoID)
	if err != nil {
		return "", fmt.Errorf("process detection failed: %w", err)
	}

	// Detect communities
	cd := graph.NewCommunityDetector(uc.pool, uc.communityRepo)
	communityCount, err := cd.DetectAll(ctx, repoID)
	if err != nil {
		return "", fmt.Errorf("community detection failed: %w", err)
	}

	return fmt.Sprintf(
		"Detection complete:\n  Processes detected:   %d\n  Communities detected: %d\n\nUse list_processes and list_communities to explore results.",
		processCount, communityCount,
	), nil
}

