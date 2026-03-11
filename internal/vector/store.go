package vector

import "context"

// VectorStore is the abstraction for vector storage backends.
// Implementations: PgVectorStore (default), QdrantClient (optional).
type VectorStore interface {
	// UpsertPoints inserts or updates embeddings.
	UpsertPoints(ctx context.Context, points []CodePoint) error

	// Search performs nearest-neighbor search and returns matching results.
	Search(ctx context.Context, vector []float32, limit int, filter map[string]string) ([]SearchResult, error)

	// Close releases resources held by the store.
	Close() error
}
