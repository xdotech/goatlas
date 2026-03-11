package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// FindSymbolUseCase finds a symbol by name and optional kind filter.
type FindSymbolUseCase struct {
	symbolRepo domain.SymbolRepository
}

// NewFindSymbolUseCase constructs a FindSymbolUseCase.
func NewFindSymbolUseCase(sr domain.SymbolRepository) *FindSymbolUseCase {
	return &FindSymbolUseCase{symbolRepo: sr}
}

// Execute searches for a symbol by name and returns formatted details.
func (uc *FindSymbolUseCase) Execute(ctx context.Context, name, kind string) (string, error) {
	symbols, err := uc.symbolRepo.Search(ctx, name, 10, kind)
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 {
		return fmt.Sprintf("Symbol %q not found", name), nil
	}
	var sb strings.Builder
	for _, s := range symbols {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", s.Kind, s.QualifiedName))
		if s.Signature != "" {
			sb.WriteString(fmt.Sprintf("  signature: %s\n", s.Signature))
		}
		if s.Receiver != "" {
			sb.WriteString(fmt.Sprintf("  receiver: %s\n", s.Receiver))
		}
		sb.WriteString(fmt.Sprintf("  line: %d\n", s.Line))
		if s.DocComment != "" {
			sb.WriteString(fmt.Sprintf("  doc: %s\n", s.DocComment))
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
