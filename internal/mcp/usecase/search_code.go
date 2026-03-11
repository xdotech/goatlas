package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/goatlas/goatlas/internal/vector"
)

// SearchCodeUseCase searches indexed symbols by keyword or semantic query.
type SearchCodeUseCase struct {
	symbolRepo domain.SymbolRepository
	searcher   *vector.Searcher // nil if vector search is not configured
}

// NewSearchCodeUseCase constructs a SearchCodeUseCase.
// Pass nil for searcher to use keyword-only search.
func NewSearchCodeUseCase(sr domain.SymbolRepository, searcher *vector.Searcher) *SearchCodeUseCase {
	return &SearchCodeUseCase{symbolRepo: sr, searcher: searcher}
}

// Execute performs symbol search and returns a formatted result string.
// mode can be "keyword" (default), "semantic", or "hybrid".
func (uc *SearchCodeUseCase) Execute(ctx context.Context, query string, limit int, kind, mode string) (string, error) {
	if limit <= 0 {
		limit = 20
	}

	if uc.searcher != nil && (mode == "semantic" || mode == "hybrid") {
		return uc.searcher.Search(ctx, query, limit, "")
	}

	// Default: PostgreSQL full-text search.
	symbols, err := uc.symbolRepo.Search(ctx, query, limit, kind)
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 {
		return "No symbols found matching: " + query, nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d symbol(s) for query: %q\n\n", len(symbols), query)
	for _, s := range symbols {
		fmt.Fprintf(&sb, "  [%s] %s\n", s.Kind, s.QualifiedName)
		if s.Signature != "" {
			fmt.Fprintf(&sb, "    sig: %s\n", s.Signature)
		}
		fmt.Fprintf(&sb, "    line: %d\n", s.Line)
		if s.DocComment != "" {
			doc := s.DocComment
			if len(doc) > 100 {
				doc = doc[:100] + "..."
			}
			fmt.Fprintf(&sb, "    doc: %s\n", doc)
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
