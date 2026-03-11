package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/goatlas/goatlas/internal/vector"
)

const defaultSnippetLines = 12

// SearchCodeUseCase searches indexed symbols by keyword or semantic query.
type SearchCodeUseCase struct {
	symbolRepo domain.SymbolRepository
	searcher   *vector.Searcher // nil if vector search is not configured
	repoRoot   string
}

// NewSearchCodeUseCase constructs a SearchCodeUseCase.
// Pass nil for searcher to use keyword-only search.
func NewSearchCodeUseCase(sr domain.SymbolRepository, searcher *vector.Searcher, repoRoot string) *SearchCodeUseCase {
	return &SearchCodeUseCase{symbolRepo: sr, searcher: searcher, repoRoot: repoRoot}
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

	// Default: PostgreSQL full-text search with file paths for snippets.
	symbols, err := uc.symbolRepo.SearchWithFile(ctx, query, limit, kind)
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
		fmt.Fprintf(&sb, "    file: %s:%d\n", s.FilePath, s.Line)
		if s.Signature != "" {
			fmt.Fprintf(&sb, "    sig: %s\n", s.Signature)
		}
		if s.DocComment != "" {
			doc := s.DocComment
			if len(doc) > 100 {
				doc = doc[:100] + "..."
			}
			fmt.Fprintf(&sb, "    doc: %s\n", doc)
		}
		// Add source snippet
		if snippet := ExtractSnippet(uc.repoRoot, s.FilePath, s.Line, defaultSnippetLines); snippet != "" {
			sb.WriteString("    ───────\n")
			sb.WriteString(snippet)
			sb.WriteString("    ───────\n")
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
