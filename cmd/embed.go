package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/xdotech/goatlas/internal/config"
	"github.com/xdotech/goatlas/internal/db"
	"github.com/xdotech/goatlas/internal/vector"
	"github.com/spf13/cobra"
)

var embedForce bool

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Embed indexed symbols into the vector database (pgvector or Qdrant)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.EmbedProvider != "ollama" && cfg.EmbedProvider != "openai" && cfg.GeminiAPIKey == "" {
			return fmt.Errorf("GEMINI_API_KEY not set (or set EMBED_PROVIDER=ollama|openai)")
		}

		pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer pool.Close()

		// Select vector store: Qdrant if configured, otherwise pgvector (default).
		var store vector.VectorStore
		if cfg.QdrantURL != "" {
			fmt.Fprintln(os.Stderr, "📦 Using Qdrant vector store")
			qc, err := vector.NewQdrantClient(ctx, cfg.QdrantURL)
			if err != nil {
				return fmt.Errorf("connect qdrant: %w", err)
			}
			defer qc.Close()
			store = qc
		} else {
			fmt.Fprintln(os.Stderr, "📦 Using pgvector (PostgreSQL)")
			store = vector.NewPgVectorStore(pool)
		}

		embedCfg := vector.EmbedConfig{
			Provider:           cfg.EmbedProvider,
			GeminiKey:          cfg.GeminiAPIKey,
			OllamaURL:          cfg.OllamaURL,
			OllamaModel:        cfg.OllamaEmbedModel,
			OpenAIBaseURL:      cfg.OpenAIBaseURL,
			OpenAIEmbedBaseURL: cfg.OpenAIEmbedBaseURL,
			OpenAIAPIKey:       cfg.OpenAIAPIKey,
			OpenAIModel:        cfg.OpenAIEmbedModel,
		}
		embedder, err := vector.NewEmbedder(ctx, embedCfg)
		if err != nil {
			return fmt.Errorf("create embedder: %w", err)
		}
		defer embedder.Close()

		indexer := vector.NewVectorIndexer(pool, store, embedder)
		result, err := indexer.IndexRepository(ctx, embedForce)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		fmt.Println("Embedding complete")
		fmt.Printf("  Embedded: %d\n", result.EmbeddedCount)
		fmt.Printf("  Skipped:  %d\n", result.SkippedCount)
		return nil
	},
}

func init() {
	embedCmd.Flags().BoolVarP(&embedForce, "force", "f", false, "Re-embed all symbols (including already-embedded)")
}
