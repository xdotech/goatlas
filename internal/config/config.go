package config

import (
	"os"

	"github.com/spf13/viper"
)

// Config holds all application configuration loaded from env vars or .env file.
type Config struct {
	RepoPath     string
	DatabaseDSN  string
	QdrantURL    string
	Neo4jURL     string
	Neo4jUser    string
	Neo4jPass    string
	GeminiAPIKey string
	HTTPAddr     string
	RRFK         int // RRF constant k (default 60)

	// LLM provider: "gemini" (default) | "ollama" | "openai"
	LLMProvider string
	// Embedding provider: "gemini" (default) | "ollama" | "openai"
	EmbedProvider string
	// Ollama settings (used when provider is "ollama")
	OllamaURL        string
	OllamaModel      string // chat model, e.g. "llama3.2"
	OllamaEmbedModel string // embedding model, e.g. "nomic-embed-text"
	// OpenAI-compatible settings (used when provider is "openai")
	OpenAIBaseURL    string // e.g. "http://10.1.1.246:8001/v1"
	OpenAIAPIKey     string // API key (use "ignored" for servers that don't require auth)
	OpenAIModel      string // chat model, e.g. "qwen3.5-35b"
	OpenAIEmbedModel string // embedding model, e.g. "text-embedding-ada-002"
	OpenAIDisableThinking bool // disable reasoning/thinking mode (e.g. Qwen3)
}

// Load reads configuration from environment variables and optional .env file.
func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	viper.SetDefault("DATABASE_DSN", "postgres://goatlas:goatlas@localhost:5432/goatlas")
	// QDRANT_URL has no default — empty means use pgvector
	viper.SetDefault("NEO4J_URL", "bolt://localhost:7687")
	viper.SetDefault("NEO4J_USER", "neo4j")
	viper.SetDefault("NEO4J_PASS", "goatlas_neo4j")
	viper.SetDefault("HTTP_ADDR", ":8080")
	viper.SetDefault("GOATLAS_RRF_K", 60)
	viper.SetDefault("LLM_PROVIDER", "gemini")
	viper.SetDefault("EMBED_PROVIDER", "gemini")
	viper.SetDefault("OLLAMA_URL", "http://localhost:11434")
	viper.SetDefault("OLLAMA_MODEL", "llama3.2")
	viper.SetDefault("OLLAMA_EMBED_MODEL", "nomic-embed-text")
	viper.SetDefault("OPENAI_BASE_URL", "http://localhost:8001/v1")
	viper.SetDefault("OPENAI_API_KEY", "")
	viper.SetDefault("OPENAI_MODEL", "gpt-3.5-turbo")
	viper.SetDefault("OPENAI_EMBED_MODEL", "text-embedding-ada-002")
	viper.SetDefault("OPENAI_DISABLE_THINKING", false)

	_ = viper.ReadInConfig() // ignore missing .env file

	repoPath := viper.GetString("REPO_PATH")
	if repoPath == "" {
		repoPath, _ = os.Getwd()
	}

	return &Config{
		RepoPath:         repoPath,
		DatabaseDSN:      viper.GetString("DATABASE_DSN"),
		QdrantURL:        viper.GetString("QDRANT_URL"),
		Neo4jURL:         viper.GetString("NEO4J_URL"),
		Neo4jUser:        viper.GetString("NEO4J_USER"),
		Neo4jPass:        viper.GetString("NEO4J_PASS"),
		GeminiAPIKey:     viper.GetString("GEMINI_API_KEY"),
		HTTPAddr:         viper.GetString("HTTP_ADDR"),
		RRFK:             viper.GetInt("GOATLAS_RRF_K"),
		LLMProvider:      viper.GetString("LLM_PROVIDER"),
		EmbedProvider:    viper.GetString("EMBED_PROVIDER"),
		OllamaURL:        viper.GetString("OLLAMA_URL"),
		OllamaModel:      viper.GetString("OLLAMA_MODEL"),
		OllamaEmbedModel: viper.GetString("OLLAMA_EMBED_MODEL"),
		OpenAIBaseURL:    viper.GetString("OPENAI_BASE_URL"),
		OpenAIAPIKey:     viper.GetString("OPENAI_API_KEY"),
		OpenAIModel:      viper.GetString("OPENAI_MODEL"),
		OpenAIEmbedModel: viper.GetString("OPENAI_EMBED_MODEL"),
		OpenAIDisableThinking: viper.GetBool("OPENAI_DISABLE_THINKING"),
	}, nil
}
