package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/xdotech/goatlas/internal/config"
	"github.com/xdotech/goatlas/internal/db"
	"github.com/xdotech/goatlas/internal/graph"
	"github.com/xdotech/goatlas/internal/indexer"
	mcpserver "github.com/xdotech/goatlas/internal/mcp"
	"github.com/xdotech/goatlas/internal/vector"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server (stdio transport)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		fmt.Fprintln(os.Stderr, "🚀 GoAtlas MCP Server starting...")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		fmt.Fprintln(os.Stderr, "✅ Config loaded")

		pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer pool.Close()
		fmt.Fprintln(os.Stderr, "✅ PostgreSQL connected")

		// Optionally connect to Neo4j — degrade gracefully if unavailable.
		var graphClient *graph.Client
		if cfg.Neo4jURL != "" {
			gc, err := graph.NewClient(ctx, cfg.Neo4jURL, cfg.Neo4jUser, cfg.Neo4jPass)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Neo4j unavailable: %v\n", err)
			} else {
				graphClient = gc
				defer gc.Close(ctx)
				fmt.Fprintln(os.Stderr, "✅ Neo4j connected")
			}
		} else {
			fmt.Fprintln(os.Stderr, "⚠️  Neo4j not configured (graph tools disabled)")
		}

		// Select vector store: Qdrant if configured, otherwise pgvector (default).
		var vectorSearcher *vector.Searcher
		embedEnabled := cfg.EmbedProvider == "ollama" || cfg.EmbedProvider == "openai" || cfg.GeminiAPIKey != ""
		if embedEnabled {
			var store vector.VectorStore
			if cfg.QdrantURL != "" {
				qc, err := vector.NewQdrantClient(ctx, cfg.QdrantURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  Qdrant unavailable: %v\n", err)
				} else {
					defer qc.Close()
					store = qc
					fmt.Fprintln(os.Stderr, "✅ Vector store: Qdrant")
				}
			}
			if store == nil {
				store = vector.NewPgVectorStore(pool)
				fmt.Fprintln(os.Stderr, "✅ Vector store: pgvector (PostgreSQL)")
			}

			embedCfg := vector.EmbedConfig{
				Provider:      cfg.EmbedProvider,
				GeminiKey:     cfg.GeminiAPIKey,
				OllamaURL:     cfg.OllamaURL,
				OllamaModel:   cfg.OllamaEmbedModel,
				OpenAIBaseURL: cfg.OpenAIBaseURL,
				OpenAIAPIKey:  cfg.OpenAIAPIKey,
				OpenAIModel:   cfg.OpenAIEmbedModel,
			}
			embedder, err := vector.NewEmbedder(ctx, embedCfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Embedder unavailable: %v\n", err)
			} else {
				defer embedder.Close()
				vectorSearcher = vector.NewSearcher(store, embedder)
			}
		} else {
			fmt.Fprintln(os.Stderr, "⚠️  No embed provider configured (semantic search disabled)")
		}

		indexerSvc := indexer.NewService(pool)
		srv := mcpserver.NewServer(mcpserver.ServerConfig{
			Config:      cfg,
			RepoRoot:    cfg.RepoPath,
			IndexerSvc:  indexerSvc,
			Pool:        pool,
			Searcher:    vectorSearcher,
			GraphClient: graphClient,
		})

		fmt.Fprintln(os.Stderr, "✅ MCP Server ready — listening on stdio")
		return srv.RunStdio()
	},
}
