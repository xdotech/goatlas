package cmd

import (
	"context"
	"fmt"

	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/db"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		pool, err := db.NewPool(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer pool.Close()
		if err := db.Migrate(context.Background(), pool); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		fmt.Println("Migrations applied successfully")
		return nil
	},
}
