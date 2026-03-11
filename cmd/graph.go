package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/db"
	"github.com/goatlas/goatlas/internal/graph"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "build-graph",
	Short: "Build knowledge graph in Neo4j from indexed repository",
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

		graphClient, err := graph.NewClient(ctx, cfg.Neo4jURL, cfg.Neo4jUser, cfg.Neo4jPass)
		if err != nil {
			return fmt.Errorf("connect neo4j: %w", err)
		}
		defer graphClient.Close(ctx)

		builder := graph.NewBuilder(pool, graphClient)
		result, err := builder.Build(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		fmt.Printf("Knowledge graph built successfully\n")
		fmt.Printf("  Package nodes: %d\n", result.PackageNodes)
		fmt.Printf("  File nodes:    %d\n", result.FileNodes)
		fmt.Printf("  Function nodes:%d\n", result.FunctionNodes)
		fmt.Printf("  Type nodes:    %d\n", result.TypeNodes)
		fmt.Printf("  Import edges:  %d\n", result.ImportEdges)
		return nil
	},
}
