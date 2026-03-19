package vector

import (
	"context"
	"fmt"
	"strings"
)

// Searcher performs semantic search using vector embeddings.
type Searcher struct {
	store    VectorStore
	embedder Embedder
}

// NewSearcher creates a Searcher.
func NewSearcher(store VectorStore, embedder Embedder) *Searcher {
	return &Searcher{store: store, embedder: embedder}
}

// Search embeds the query and returns formatted nearest-neighbour results.
// serviceFilter restricts results to a specific service/module; empty means all.
func (s *Searcher) Search(ctx context.Context, query string, limit int, serviceFilter string) (string, error) {
	vec, err := s.embedder.EmbedOne(ctx, query)
	if err != nil {
		return "", fmt.Errorf("embed query: %w", err)
	}

	filter := map[string]string{}
	if serviceFilter != "" {
		filter["service"] = serviceFilter
	}

	results, err := s.store.Search(ctx, vec, limit, filter)
	if err != nil {
		return "", fmt.Errorf("vector search: %w", err)
	}

	if len(results) == 0 {
		return "No semantically similar symbols found for: " + query, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Semantic search results for %q:\n\n", query)
	for _, r := range results {
		fmt.Fprintf(&sb, "  [%.3f] [%s] %s\n", r.Score, r.Kind, r.Name)
		if r.File != "" {
			fmt.Fprintf(&sb, "    file: %s\n", r.File)
		}
	}
	return sb.String(), nil
}

// SearchStructured embeds the query and returns structured nearest-neighbour results.
// This is used by hybrid search for RRF merging.
func (s *Searcher) SearchStructured(ctx context.Context, query string, limit int, serviceFilter string) ([]SearchResult, error) {
	vec, err := s.embedder.EmbedOne(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	filter := map[string]string{}
	if serviceFilter != "" {
		filter["service"] = serviceFilter
	}
	return s.store.Search(ctx, vec, limit, filter)
}
