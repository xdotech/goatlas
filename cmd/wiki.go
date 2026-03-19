package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xdotech/goatlas/internal/agent"
	"github.com/xdotech/goatlas/internal/config"
	"github.com/xdotech/goatlas/internal/db"
	"github.com/spf13/cobra"
)

var wikiCmd = &cobra.Command{
	Use:   "wiki <output-dir>",
	Short: "Generate a Markdown wiki from the knowledge graph",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}
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

		// Resolve default repo
		var repoID int64
		if err := pool.QueryRow(ctx, `SELECT id FROM repositories ORDER BY last_indexed_at DESC NULLS LAST LIMIT 1`).Scan(&repoID); err != nil {
			return fmt.Errorf("no repositories indexed — run `goatlas index <path>` first")
		}

		// Create output directories
		for _, dir := range []string{outputDir, filepath.Join(outputDir, "services"), filepath.Join(outputDir, "communities")} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create directory %s: %w", dir, err)
			}
		}

		// Create agent
		a, err := agent.NewAgent(ctx, agent.DefaultConfig(), cfg.GeminiAPIKey, nil, "")
		if err != nil {
			return fmt.Errorf("create agent: %w", err)
		}
		defer a.Close()

		gen := agent.NewDocGenerator(a, pool)
		files, err := gen.GenerateWiki(ctx, repoID, outputDir)
		if err != nil {
			return err
		}

		// Write pages
		pages := gen.GetWikiPages()
		for path, content := range pages {
			fullPath := filepath.Join(outputDir, path)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating dir for %s: %v\n", fullPath, err)
				continue
			}
			if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", fullPath, err)
				continue
			}
			fmt.Printf("✅ %s\n", fullPath)
		}

		fmt.Printf("\n📚 Generated %d wiki page(s) in %s\n", len(files), outputDir)
		return nil
	},
}
