package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetProcessFlowUseCase retrieves the ordered steps of a named process.
type GetProcessFlowUseCase struct {
	processRepo domain.ProcessRepository
	pool        *pgxpool.Pool
}

// NewGetProcessFlowUseCase creates a new GetProcessFlowUseCase.
func NewGetProcessFlowUseCase(pr domain.ProcessRepository, pool *pgxpool.Pool) *GetProcessFlowUseCase {
	return &GetProcessFlowUseCase{processRepo: pr, pool: pool}
}

// Execute retrieves the flow for a process by name.
func (uc *GetProcessFlowUseCase) Execute(ctx context.Context, processName string) (string, error) {
	repoID, err := getFirstRepoID(ctx, uc.pool)
	if err != nil {
		return "", fmt.Errorf("no indexed repository found: %w", err)
	}

	proc, err := uc.processRepo.GetByName(ctx, repoID, processName)
	if err != nil {
		// Try partial match
		procs, listErr := uc.processRepo.List(ctx, repoID)
		if listErr != nil {
			return "", err
		}
		for _, p := range procs {
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(processName)) {
				proc = &p
				break
			}
		}
		if proc == nil {
			return fmt.Sprintf("Process %q not found. Use list_processes to see available processes.", processName), nil
		}
	}

	steps, err := uc.processRepo.GetSteps(ctx, proc.ID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Process: %s\n", proc.Name))
	sb.WriteString(fmt.Sprintf("Entry point: %s\n", proc.EntryPoint))
	if proc.FilePath != "" {
		sb.WriteString(fmt.Sprintf("File: %s\n", proc.FilePath))
	}
	sb.WriteString(fmt.Sprintf("\nExecution flow (%d steps):\n\n", len(steps)))

	for _, s := range steps {
		sb.WriteString(fmt.Sprintf("  %d. %s", s.StepOrder, s.SymbolName))
		if s.FilePath != "" {
			sb.WriteString(fmt.Sprintf(" @ %s", s.FilePath))
			if s.Line > 0 {
				sb.WriteString(fmt.Sprintf(":%d", s.Line))
			}
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

