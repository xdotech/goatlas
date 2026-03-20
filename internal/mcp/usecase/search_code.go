package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/xdotech/goatlas/internal/vector"
	"golang.org/x/sync/errgroup"
)

// isGeminiAuthError returns true if the error is a Gemini API auth failure.
func isGeminiAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "API_KEY_INVALID") ||
		strings.Contains(msg, "UNAUTHENTICATED") ||
		strings.Contains(msg, "gemini API key invalid")
}

const defaultSnippetLines = 12
const defaultRRFK = 60

// SearchCodeUseCase searches indexed symbols by keyword or semantic query.
type SearchCodeUseCase struct {
	symbolRepo domain.SymbolRepository
	searcher   *vector.Searcher // nil if vector search is not configured
	repoRoot   string
	rrfK       int
}

// NewSearchCodeUseCase constructs a SearchCodeUseCase.
// Pass nil for searcher to use keyword-only search. rrfK is the RRF constant (0 = use default 60).
func NewSearchCodeUseCase(sr domain.SymbolRepository, searcher *vector.Searcher, repoRoot string, rrfK int) *SearchCodeUseCase {
	if rrfK <= 0 {
		rrfK = defaultRRFK
	}
	return &SearchCodeUseCase{symbolRepo: sr, searcher: searcher, repoRoot: repoRoot, rrfK: rrfK}
}

// Execute performs symbol search and returns a formatted result string.
// mode can be "keyword" (default), "semantic", or "hybrid".
func (uc *SearchCodeUseCase) Execute(ctx context.Context, query string, limit int, kind, mode string) (string, error) {
	if limit <= 0 {
		limit = 20
	}

	// Auto-select hybrid when searcher is available and mode is unset.
	if mode == "" && uc.searcher != nil {
		mode = "hybrid"
	}

	switch mode {
	case "semantic":
		if uc.searcher != nil {
			return uc.searcher.Search(ctx, query, limit, "")
		}
		fallthrough
	case "hybrid":
		if uc.searcher != nil {
			return uc.hybridSearch(ctx, query, limit, kind)
		}
		fallthrough
	default:
		symbols, err := uc.symbolRepo.SearchWithFile(ctx, query, limit, kind)
		if err != nil {
			return "", err
		}
		if len(symbols) == 0 {
			return "No symbols found matching: " + query, nil
		}
		return formatKeywordResults(query, symbols, uc.repoRoot), nil
	}
}

// hybridSearch merges BM25 and semantic results using Reciprocal Rank Fusion.
func (uc *SearchCodeUseCase) hybridSearch(ctx context.Context, query string, limit int, kind string) (string, error) {
	g, gctx := errgroup.WithContext(ctx)
	var bm25 []domain.SymbolWithRank
	var vecResults []vector.SearchResult

	g.Go(func() error {
		var err error
		bm25, err = uc.symbolRepo.SearchRanked(gctx, query, limit*2, kind)
		return err
	})
	g.Go(func() error {
		var err error
		vecResults, err = uc.searcher.SearchStructured(gctx, query, limit*2, "")
		return err
	})

	var partialErr error
	if err := g.Wait(); err != nil {
		if len(bm25) == 0 && len(vecResults) == 0 {
			return "Search failed: " + err.Error(), nil
		}
		partialErr = err
		// Surface auth errors clearly instead of burying them
		if isGeminiAuthError(err) {
			partialErr = fmt.Errorf("⚠️ Semantic search unavailable (GEMINI_API_KEY invalid). Showing keyword results only")
		}
	}

	type scoreEntry struct {
		symbolID int64
		score    float64
		name     string
		kind     string
		file     string
		line     int
	}
	scores := make(map[int64]*scoreEntry)

	for i, r := range bm25 {
		if _, ok := scores[r.ID]; !ok {
			scores[r.ID] = &scoreEntry{symbolID: r.ID, name: r.QualifiedName, kind: r.Kind, file: r.FilePath, line: r.Line}
		}
		scores[r.ID].score += 1.0 / float64(uc.rrfK+i+1)
	}
	for i, r := range vecResults {
		if _, ok := scores[r.SymbolID]; !ok {
			scores[r.SymbolID] = &scoreEntry{symbolID: r.SymbolID, name: r.Name, kind: r.Kind, file: r.File, line: r.Line}
		}
		scores[r.SymbolID].score += 1.0 / float64(uc.rrfK+i+1)
	}

	merged := make([]*scoreEntry, 0, len(scores))
	for _, e := range scores {
		merged = append(merged, e)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].score > merged[j].score
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}

	if len(merged) == 0 {
		return "No symbols found matching: " + query, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Hybrid search results for %q (%d found):\n", query, len(merged))
	if partialErr != nil {
		fmt.Fprintf(&sb, "(warning: one search path failed — results may be incomplete: %v)\n", partialErr)
	}
	sb.WriteString("\n")
	for _, e := range merged {
		fmt.Fprintf(&sb, "  [%.4f] [%s] %s\n", e.score, e.kind, e.name)
		if e.file != "" {
			fmt.Fprintf(&sb, "    file: %s:%d\n", e.file, e.line)
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

func formatKeywordResults(query string, symbols []domain.SymbolWithFile, repoRoot string) string {
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
		if snippet := ExtractSnippet(repoRoot, s.FilePath, s.Line, defaultSnippetLines); snippet != "" {
			sb.WriteString("    ───────\n")
			sb.WriteString(snippet)
			sb.WriteString("    ───────\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
