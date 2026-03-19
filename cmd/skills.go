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

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage auto-generated SKILL.md files",
}

var skillsGenerateCmd = &cobra.Command{
	Use:   "generate <repo-path>",
	Short: "Generate SKILL.md files for each community cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := filepath.Abs(args[0])
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

		// Resolve repo ID
		var repoID int64
		if err := pool.QueryRow(ctx, `SELECT id FROM repositories WHERE path = $1 LIMIT 1`, repoPath).Scan(&repoID); err != nil {
			return fmt.Errorf("repo not indexed — run `goatlas index %s` first", repoPath)
		}

		// Create agent
		a, err := agent.NewAgent(ctx, agent.DefaultConfig(), cfg.GeminiAPIKey, nil, "")
		if err != nil {
			return fmt.Errorf("create agent: %w", err)
		}
		defer a.Close()

		gen := agent.NewDocGenerator(a, pool)
		results, err := gen.GenerateSkillsForRepo(ctx, repoID, repoPath)
		if err != nil {
			return err
		}

		for _, r := range results {
			// Backup existing SKILL.md
			if _, err := os.Stat(r.FilePath); err == nil {
				backupPath := r.FilePath + ".bak"
				_ = os.Rename(r.FilePath, backupPath)
				fmt.Printf("📋 Backed up %s → %s\n", r.FilePath, backupPath)
			}

			// Ensure directory exists
			if err := os.MkdirAll(filepath.Dir(r.FilePath), 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating dir for %s: %v\n", r.FilePath, err)
				continue
			}

			if err := os.WriteFile(r.FilePath, []byte(r.Content), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", r.FilePath, err)
				continue
			}
			fmt.Printf("✅ %s → %s\n", r.CommunityName, r.FilePath)
		}

		fmt.Printf("\n📝 Generated %d SKILL.md file(s)\n", len(results))
		return nil
	},
}

func init() {
	skillsCmd.AddCommand(skillsGenerateCmd)
}
