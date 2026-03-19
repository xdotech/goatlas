package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/xdotech/goatlas/internal/config"
	"github.com/xdotech/goatlas/internal/db"
	"github.com/xdotech/goatlas/internal/graph"
	"github.com/xdotech/goatlas/internal/indexer/repository/postgres"
	"github.com/spf13/cobra"
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect execution processes and code communities",
	Long:  `Runs process detection (forward BFS from entry points) and community detection (Louvain) on indexed data.`,
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

		// Neo4j is optional — process detection works with PG-only fallback
		var graphClient *graph.Client
		if cfg.Neo4jURL != "" {
			gc, err := graph.NewClient(ctx, cfg.Neo4jURL, cfg.Neo4jUser, cfg.Neo4jPass)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Neo4j unavailable, using PG-only mode: %v\n", err)
			} else {
				graphClient = gc
				defer graphClient.Close(ctx)
			}
		}

		processRepo := postgres.NewProcessRepo(pool)
		communityRepo := postgres.NewCommunityRepo(pool)

		// Get first repo ID
		var repoID int64
		if err := pool.QueryRow(ctx, `SELECT id FROM repositories ORDER BY id LIMIT 1`).Scan(&repoID); err != nil {
			return fmt.Errorf("no indexed repository found — run 'goatlas index' first: %w", err)
		}

		// Detect processes
		fmt.Println("Detecting execution processes...")
		pd := graph.NewProcessDetector(pool, graphClient, processRepo)
		processCount, err := pd.DetectAll(ctx, repoID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Process detection error: %v\n", err)
		} else {
			fmt.Printf("  Processes detected: %d\n", processCount)
		}

		// Detect communities
		fmt.Println("Detecting code communities (Louvain)...")
		cd := graph.NewCommunityDetector(pool, communityRepo)
		communityCount, err := cd.DetectAll(ctx, repoID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Community detection error: %v\n", err)
		} else {
			fmt.Printf("  Communities detected: %d\n", communityCount)
		}

		fmt.Println("\nDone. Use MCP tools list_processes and list_communities to explore results.")
		return nil
	},
}
