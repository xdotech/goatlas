package usecase

import (
	"context"
	"fmt"

	"github.com/goatlas/goatlas/internal/vector"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GenerateEmbeddingsUseCase generates vector embeddings via MCP.
type GenerateEmbeddingsUseCase struct {
	pool      *pgxpool.Pool
	qdrantURL string
	apiKey    string
}

// NewGenerateEmbeddingsUseCase creates a new GenerateEmbeddingsUseCase.
func NewGenerateEmbeddingsUseCase(pool *pgxpool.Pool, qdrantURL, apiKey string) *GenerateEmbeddingsUseCase {
	return &GenerateEmbeddingsUseCase{pool: pool, qdrantURL: qdrantURL, apiKey: apiKey}
}

// Execute generates embeddings for all indexed symbols.
func (uc *GenerateEmbeddingsUseCase) Execute(ctx context.Context, force bool) (string, error) {
	if uc.apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY not configured — cannot generate embeddings")
	}

	// Select vector store: Qdrant if configured, otherwise pgvector (default).
	var store vector.VectorStore
	if uc.qdrantURL != "" {
		qc, err := vector.NewQdrantClient(ctx, uc.qdrantURL)
		if err != nil {
			return "", fmt.Errorf("connect qdrant: %w", err)
		}
		defer qc.Close()
		store = qc
	} else {
		store = vector.NewPgVectorStore(uc.pool)
	}

	embedder, err := vector.NewEmbedder(ctx, uc.apiKey)
	if err != nil {
		return "", fmt.Errorf("create embedder: %w", err)
	}
	defer embedder.Close()

	indexer := vector.NewVectorIndexer(uc.pool, store, embedder)
	result, err := indexer.IndexRepository(ctx, force)
	if err != nil {
		return "", err
	}

	backend := "pgvector"
	if uc.qdrantURL != "" {
		backend = "qdrant"
	}

	return fmt.Sprintf(
		"Embedding complete (backend: %s)\n  Embedded: %d\n  Skipped:  %d",
		backend, result.EmbeddedCount, result.SkippedCount,
	), nil
}
