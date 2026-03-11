package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/goatlas/goatlas/internal/indexer"
)

// IndexRepoUseCase indexes a repository via MCP.
type IndexRepoUseCase struct {
	svc *indexer.Service
}

// NewIndexRepoUseCase creates a new IndexRepoUseCase.
func NewIndexRepoUseCase(svc *indexer.Service) *IndexRepoUseCase {
	return &IndexRepoUseCase{svc: svc}
}

// Execute indexes the given repository path.
func (uc *IndexRepoUseCase) Execute(ctx context.Context, repoPath string, force bool) (string, error) {
	if repoPath == "" {
		return "", fmt.Errorf("repo path is required")
	}

	result, err := uc.svc.IndexRepo.Execute(ctx, repoPath, force)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"Indexing complete in %s\n  Files indexed:   %d\n  Files skipped:   %d\n  Symbols found:   %d\n  Endpoints found: %d",
		result.Duration.Round(time.Millisecond),
		result.FilesIndexed,
		result.FilesSkipped,
		result.SymbolsFound,
		result.EndpointsFound,
	), nil
}
