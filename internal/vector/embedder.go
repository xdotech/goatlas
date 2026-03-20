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
	Provider  string // "gemini" (default) | "ollama" | "openai"
	GeminiKey string
	OllamaURL string // default: http://localhost:11434
	OllamaModel string // default: nomic-embed-text
	// OpenAI-compatible API settings
	OpenAIBaseURL      string // e.g. "http://10.1.1.246:8001/v1"
	OpenAIEmbedBaseURL string // separate base URL for embeddings (falls back to OpenAIBaseURL)
	OpenAIAPIKey       string
	OpenAIModel        string // e.g. "text-embedding-ada-002"
}

// NewEmbedder returns the Embedder implementation for the configured provider.
func NewEmbedder(ctx context.Context, cfg EmbedConfig) (Embedder, error) {
	switch cfg.Provider {
	case "ollama":
		return newOllamaEmbedder(cfg)
	case "openai":
		return newOpenAIEmbedder(cfg)
	default:
		return newGeminiEmbedder(ctx, cfg.GeminiKey)
	}
}
