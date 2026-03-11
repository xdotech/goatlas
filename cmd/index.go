package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/db"
	"github.com/goatlas/goatlas/internal/indexer"
	"github.com/spf13/cobra"
)

var indexForce bool

var indexCmd = &cobra.Command{
	Use:   "index <repo-path>",
	Short: "Index a Go repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := args[0]
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

		svc := indexer.NewService(pool)
		result, err := svc.IndexRepo.Execute(ctx, repoPath, indexForce)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return err
		}

		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	indexCmd.Flags().BoolVarP(&indexForce, "force", "f", false, "Force re-index all files")
}
