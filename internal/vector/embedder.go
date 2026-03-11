package vector

import (
	"context"
	"fmt"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const embeddingModel = "text-embedding-004"
const batchSize = 32

// Embedder wraps the Gemini embedding model.
type Embedder struct {
	client *genai.Client
	model  string
}

// NewEmbedder creates an Embedder using the given Gemini API key.
func NewEmbedder(ctx context.Context, apiKey string) (*Embedder, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}
	return &Embedder{client: client, model: embeddingModel}, nil
}

// Close releases the underlying client connection.
func (e *Embedder) Close() {
	e.client.Close()
}

// EmbedBatch returns embeddings for a slice of texts, processing in chunks.
func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	var results [][]float32
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := e.embedChunk(ctx, texts[i:end])
		if err != nil {
			return nil, fmt.Errorf("embed chunk at %d: %w", i, err)
		}
		results = append(results, vecs...)
	}
	return results, nil
}

// EmbedOne returns an embedding for a single text.
func (e *Embedder) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// embedChunk embeds a slice of texts with exponential backoff on failure.
func (e *Embedder) embedChunk(ctx context.Context, texts []string) ([][]float32, error) {
	em := e.client.EmbeddingModel(e.model)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}
		batch := em.NewBatch()
		for _, t := range texts {
			batch.AddContent(genai.Text(t))
		}
		resp, err := em.BatchEmbedContents(ctx, batch)
		if err != nil {
			lastErr = err
			continue
		}
		results := make([][]float32, len(resp.Embeddings))
		for i, emb := range resp.Embeddings {
			results[i] = emb.Values
		}
		return results, nil
	}
	return nil, fmt.Errorf("embed chunk failed after 3 attempts: %w", lastErr)
}
