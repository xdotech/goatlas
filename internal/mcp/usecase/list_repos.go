package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/mcp/registry"
)

// ListReposUseCase lists all indexed repositories.
type ListReposUseCase struct {
	registry *registry.RepoRegistry
}

// NewListReposUseCase creates a new ListReposUseCase.
func NewListReposUseCase(r *registry.RepoRegistry) *ListReposUseCase {
	return &ListReposUseCase{registry: r}
}

// Execute returns a formatted list of all indexed repositories.
func (uc *ListReposUseCase) Execute(ctx context.Context) (string, error) {
	repos, err := uc.registry.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list repos: %w", err)
	}

	if len(repos) == 0 {
		return "No repositories indexed yet. Run `goatlas index <repo-path>` first.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d indexed repository(ies):\n\n", len(repos)))

	for _, r := range repos {
		lastIndexed := "never"
		if r.LastIndexedAt != nil {
			lastIndexed = r.LastIndexedAt.Format("2006-01-02 15:04:05")
		}
		commit := r.LastCommit
		if len(commit) > 8 {
			commit = commit[:8]
		}
		if commit == "" {
			commit = "(none)"
		}
		sb.WriteString(fmt.Sprintf("- **%s**\n  Path: `%s`\n  Last indexed: %s\n  Last commit: %s\n\n", r.Name, r.Path, lastIndexed, commit))
	}
	return sb.String(), nil
}
