package usecase

import (
	"context"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// SearchSymbolsUseCase performs full-text symbol search against the index.
type SearchSymbolsUseCase struct {
	symbolRepo domain.SymbolRepository
}

// NewSearchSymbolsUseCase creates a new SearchSymbolsUseCase.
func NewSearchSymbolsUseCase(sr domain.SymbolRepository) *SearchSymbolsUseCase {
	return &SearchSymbolsUseCase{symbolRepo: sr}
}

// Execute searches for symbols matching query, limited to limit results.
// Passing kind filters results to a specific symbol kind (func, struct, etc.).
func (uc *SearchSymbolsUseCase) Execute(ctx context.Context, query string, limit int, kind string) ([]domain.Symbol, error) {
	return uc.symbolRepo.Search(ctx, query, limit, kind)
}
