package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// GetFileSymbolsUseCase retrieves all symbols defined in a specific file.
type GetFileSymbolsUseCase struct {
	fileRepo   domain.FileRepository
	symbolRepo domain.SymbolRepository
}

// NewGetFileSymbolsUseCase constructs a GetFileSymbolsUseCase.
func NewGetFileSymbolsUseCase(fr domain.FileRepository, sr domain.SymbolRepository) *GetFileSymbolsUseCase {
	return &GetFileSymbolsUseCase{fileRepo: fr, symbolRepo: sr}
}

// Execute returns all symbols in the given file path.
func (uc *GetFileSymbolsUseCase) Execute(ctx context.Context, path string) (string, error) {
	file, err := uc.fileRepo.GetByPath(ctx, "", path)
	if err != nil {
		return "", err
	}
	if file == nil {
		return fmt.Sprintf("File %q not found in index", path), nil
	}
	symbols, err := uc.symbolRepo.GetByFile(ctx, file.ID)
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 {
		return fmt.Sprintf("No symbols found in %s", path), nil
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Symbols in %s:\n\n", path))
	for _, s := range symbols {
		sb.WriteString(fmt.Sprintf("  [%s] %s (line %d)\n", s.Kind, s.Name, s.Line))
		if s.Signature != "" {
			sb.WriteString(fmt.Sprintf("    %s\n", s.Signature))
		}
	}
	return sb.String(), nil
}
