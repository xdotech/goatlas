package usecase

import (
	"context"
	"fmt"
	"sort"
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

		// Show up to 3 representative members (shortest qualified names = most readable)
		members, err := uc.communityRepo.GetMembers(ctx, c.ID)
		if err == nil && len(members) > 0 {
			// Sort by name length ascending (shortest = simplest)
			sorted := make([]string, len(members))
			for i, m := range members {
				sorted[i] = m.SymbolName
			}
			sort.Slice(sorted, func(i, j int) bool {
				return len(sorted[i]) < len(sorted[j])
			})
			limit := 3
			if len(sorted) < limit {
				limit = len(sorted)
			}
			preview := make([]string, limit)
			for i := 0; i < limit; i++ {
				// Show just the last part of the qualified name for readability
				parts := strings.Split(sorted[i], ".")
				preview[i] = parts[len(parts)-1]
			}
			sb.WriteString(fmt.Sprintf("    → %s\n", strings.Join(preview, ", ")))
		}
	}
	return sb.String(), nil
}

