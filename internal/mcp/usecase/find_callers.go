package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// FindCallersUseCase finds symbols that reference a given function name.
type FindCallersUseCase struct {
	symbolRepo domain.SymbolRepository
}

// NewFindCallersUseCase constructs a FindCallersUseCase.
func NewFindCallersUseCase(sr domain.SymbolRepository) *FindCallersUseCase {
	return &FindCallersUseCase{symbolRepo: sr}
}

// Execute searches for references/callers of the given function name.
func (uc *FindCallersUseCase) Execute(ctx context.Context, functionName string) (string, error) {
	symbols, err := uc.symbolRepo.Search(ctx, functionName, 30, "")
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 {
		return fmt.Sprintf("No callers found for %q", functionName), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Potential callers/references for %q:\n\n", functionName))
	for _, s := range symbols {
		sb.WriteString(fmt.Sprintf("  [%s] %s (line %d)\n", s.Kind, s.QualifiedName, s.Line))
	}
	return sb.String(), nil
}
