package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/db"
	"github.com/goatlas/goatlas/internal/vector"
	"github.com/spf13/cobra"
)

var embedForce bool

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Embed indexed symbols into the Qdrant vector database",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.GeminiAPIKey == "" {
			return fmt.Errorf("GEMINI_API_KEY not set")
		}

		pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer pool.Close()

		qdrantClient, err := vector.NewQdrantClient(ctx, cfg.QdrantURL)
		if err != nil {
			return fmt.Errorf("connect qdrant: %w", err)
		}
		defer qdrantClient.Close()

		embedder, err := vector.NewEmbedder(ctx, cfg.GeminiAPIKey)
		if err != nil {
			return fmt.Errorf("create embedder: %w", err)
		}
		defer embedder.Close()

		indexer := vector.NewVectorIndexer(pool, qdrantClient, embedder)
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
