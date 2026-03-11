package usecase

import (
	"context"
	"encoding/json"
	"fmt"

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

	// result is map[string]any, format it as JSON for MCP output
	out, _ := json.MarshalIndent(result, "", "  ")
	return string(out), nil
}
