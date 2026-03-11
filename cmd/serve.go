package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/db"
	"github.com/goatlas/goatlas/internal/graph"
	"github.com/goatlas/goatlas/internal/indexer"
	mcpserver "github.com/goatlas/goatlas/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server (stdio transport)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer pool.Close()

		// Optionally connect to Neo4j — degrade gracefully if unavailable.
		var graphClient *graph.Client
		if cfg.Neo4jURL != "" {
			gc, err := graph.NewClient(ctx, cfg.Neo4jURL, cfg.Neo4jUser, cfg.Neo4jPass)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not connect to Neo4j: %v\n", err)
			} else {
				graphClient = gc
				defer gc.Close(ctx)
			}
		}

		indexerSvc := indexer.NewService(pool)
		srv := mcpserver.NewServer(mcpserver.ServerConfig{
			RepoRoot:    cfg.RepoPath,
			IndexerSvc:  indexerSvc,
			Pool:        pool,
			Searcher:    nil,
			GraphClient: graphClient,
		})
		return srv.RunStdio()
	},
}
