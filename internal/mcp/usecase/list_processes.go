package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListProcessesUseCase lists all detected execution processes.
type ListProcessesUseCase struct {
	processRepo domain.ProcessRepository
	pool        *pgxpool.Pool
}

// NewListProcessesUseCase creates a new ListProcessesUseCase.
func NewListProcessesUseCase(pr domain.ProcessRepository, pool *pgxpool.Pool) *ListProcessesUseCase {
	return &ListProcessesUseCase{processRepo: pr, pool: pool}
}

// Execute lists all processes for the first indexed repo.
func (uc *ListProcessesUseCase) Execute(ctx context.Context) (string, error) {
	repoID, err := getFirstRepoID(ctx, uc.pool)
	if err != nil {
		return "", fmt.Errorf("no indexed repository found: %w", err)
	}

	procs, err := uc.processRepo.List(ctx, repoID)
	if err != nil {
		return "", err
	}
	if len(procs) == 0 {
		return "No processes detected. Run detect_processes first.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d execution process(es):\n\n", len(procs)))
	for _, p := range procs {
		sb.WriteString(fmt.Sprintf("  • %s\n", p.Name))
		sb.WriteString(fmt.Sprintf("    Entry: %s", p.EntryPoint))
		if p.FilePath != "" {
			sb.WriteString(fmt.Sprintf(" @ %s", p.FilePath))
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

