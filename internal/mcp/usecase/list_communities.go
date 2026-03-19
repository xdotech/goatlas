package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListCommunitiesUseCase lists all detected code communities.
type ListCommunitiesUseCase struct {
	communityRepo domain.CommunityRepository
	pool          *pgxpool.Pool
}

// NewListCommunitiesUseCase creates a new ListCommunitiesUseCase.
func NewListCommunitiesUseCase(cr domain.CommunityRepository, pool *pgxpool.Pool) *ListCommunitiesUseCase {
	return &ListCommunitiesUseCase{communityRepo: cr, pool: pool}
}

// Execute lists all communities for the first indexed repo.
func (uc *ListCommunitiesUseCase) Execute(ctx context.Context) (string, error) {
	repoID, err := getFirstRepoID(ctx, uc.pool)
	if err != nil {
		return "", fmt.Errorf("no indexed repository found: %w", err)
	}

	comms, err := uc.communityRepo.List(ctx, repoID)
	if err != nil {
		return "", err
	}
	if len(comms) == 0 {
		return "No communities detected. Run detect_processes first (it also runs community detection).", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d code community/communities:\n\n", len(comms)))
	for _, c := range comms {
		sb.WriteString(fmt.Sprintf("  • %s (%d members)\n", c.Name, c.MemberCount))
	}
	return sb.String(), nil
}

