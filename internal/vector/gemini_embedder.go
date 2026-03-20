package vector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const geminiEmbeddingModel = "text-embedding-004"
const batchSize = 32

// geminiEmbedder wraps the Gemini embedding model.
type geminiEmbedder struct {
	client *genai.Client
	model  string
}

func newGeminiEmbedder(ctx context.Context, apiKey string) (*geminiEmbedder, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}
	return &geminiEmbedder{client: client, model: geminiEmbeddingModel}, nil
}

func (e *geminiEmbedder) Close() {
	e.client.Close()
}

func (e *geminiEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
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

func (e *geminiEmbedder) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (e *geminiEmbedder) embedChunk(ctx context.Context, texts []string) ([][]float32, error) {
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
			// Don't retry auth/key errors — they won't recover with retries
			if isAuthError(err) {
				return nil, fmt.Errorf("gemini API key invalid or unauthorized: %w", err)
			}
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

// isAuthError returns true for API key / authentication failures that should not be retried.
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "API_KEY_INVALID") ||
		strings.Contains(msg, "UNAUTHENTICATED") ||
		strings.Contains(msg, "401") ||
		strings.Contains(msg, "403")
}
