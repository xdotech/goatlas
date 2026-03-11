package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "goatlas",
	Short: "AI Code Intelligence & Spec Verification System",
	Long:  `GoAtlas helps LLMs understand large codebases via MCP tools.`,
}

// Execute runs the root cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(coverageCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(embedCmd)
	rootCmd.AddCommand(graphCmd)
}
