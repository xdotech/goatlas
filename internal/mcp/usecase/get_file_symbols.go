package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GetFileSymbolsUseCase retrieves all symbols defined in a specific file.
type GetFileSymbolsUseCase struct {
	pool       *pgxpool.Pool
	symbolRepo domain.SymbolRepository
}

// NewGetFileSymbolsUseCase constructs a GetFileSymbolsUseCase.
func NewGetFileSymbolsUseCase(pool *pgxpool.Pool, sr domain.SymbolRepository) *GetFileSymbolsUseCase {
	return &GetFileSymbolsUseCase{pool: pool, symbolRepo: sr}
}

// Execute returns all symbols in the given file path.
func (uc *GetFileSymbolsUseCase) Execute(ctx context.Context, path string) (string, error) {
	// Look up the file by path (across all repos)
	var fileID int64
	err := uc.pool.QueryRow(ctx, `SELECT id FROM files WHERE path = $1 LIMIT 1`, path).Scan(&fileID)
	if err != nil {
		return fmt.Sprintf("File %q not found in index", path), nil
	}

	symbols, err := uc.symbolRepo.GetByFile(ctx, fileID)
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

