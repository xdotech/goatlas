package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	idxusecase "github.com/goatlas/goatlas/internal/indexer/usecase"
)

// CheckStalenessUseCase checks whether a repository needs re-indexing.
type CheckStalenessUseCase struct {
	repoRepo domain.RepositoryRepository
}

// NewCheckStalenessUseCase constructs a CheckStalenessUseCase.
func NewCheckStalenessUseCase(rr domain.RepositoryRepository) *CheckStalenessUseCase {
	return &CheckStalenessUseCase{repoRepo: rr}
}

// Execute checks staleness for the named repo (or first repo if name is empty).
func (uc *CheckStalenessUseCase) Execute(ctx context.Context, repoName string) (string, error) {
	var repo *domain.Repository
	var err error

	if repoName == "" {
		repos, e := uc.repoRepo.List(ctx)
		if e != nil || len(repos) == 0 {
			return "No repositories indexed yet.", e
		}
		r := repos[0]
		repo = &r
	} else {
		repo, err = uc.repoRepo.GetByName(ctx, repoName)
		if err != nil || repo == nil {
			return fmt.Sprintf("Repository %q not found.", repoName), err
		}
	}

	currentHead, _ := idxusecase.GetHeadCommit(repo.Path)
	stale := currentHead != "" && repo.LastCommit != currentHead
	indexedAt := "never"
	if repo.LastIndexedAt != nil {
		indexedAt = repo.LastIndexedAt.Format("2006-01-02 15:04:05 UTC")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Repository: %s\n", repo.Name)
	fmt.Fprintf(&sb, "Path: %s\n", repo.Path)
	fmt.Fprintf(&sb, "Indexed commit: %s\n", orNone(repo.LastCommit))
	fmt.Fprintf(&sb, "Current HEAD:   %s\n", orNone(currentHead))
	fmt.Fprintf(&sb, "Last indexed:   %s\n", indexedAt)
	if currentHead == "" {
		fmt.Fprintf(&sb, "Status: unknown (not a git repository or no commits)\n")
	} else if stale {
		fmt.Fprintf(&sb, "Status: STALE — run 'goatlas index --incremental %s' to update\n", repo.Path)
	} else {
		fmt.Fprintf(&sb, "Status: up to date\n")
	}
	return sb.String(), nil
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}
