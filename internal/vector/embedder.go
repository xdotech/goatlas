package vector

import "context"

// Embedder is the interface for embedding text into vectors.
type Embedder interface {
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	EmbedOne(ctx context.Context, text string) ([]float32, error)
	Close()
}

// EmbedConfig selects and configures the embedding provider.
type EmbedConfig struct {
	Provider  string // "gemini" (default) | "ollama"
	GeminiKey string
	OllamaURL string // default: http://localhost:11434
	OllamaModel string // default: nomic-embed-text
}

// NewEmbedder returns the Embedder implementation for the configured provider.
func NewEmbedder(ctx context.Context, cfg EmbedConfig) (Embedder, error) {
	switch cfg.Provider {
	case "ollama":
		return newOllamaEmbedder(cfg)
	default:
		return newGeminiEmbedder(ctx, cfg.GeminiKey)
	}
}
