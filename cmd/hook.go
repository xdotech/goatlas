package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// hookCmd is the parent for "goatlas hook pre" and "goatlas hook post".
var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Claude Code hook handlers (PreToolUse / PostToolUse)",
	Long:  `These subcommands are called by Claude Code hooks. They read tool input/output from stdin and write enrichment to stdout.`,
}

// hookPreCmd handles PreToolUse: when Claude calls Grep/Glob, enrich with semantic context.
var hookPreCmd = &cobra.Command{
	Use:   "pre",
	Short: "PreToolUse hook: enrich Grep/Glob results with semantic context",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}

		// Claude Code sends JSON with tool_name and tool_input
		var payload struct {
			ToolName  string         `json:"tool_name"`
			ToolInput map[string]any `json:"tool_input"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			// Not valid JSON, skip silently
			return nil
		}

		toolName := strings.ToLower(payload.ToolName)
		if toolName != "grep" && toolName != "glob" {
			// Not a tool we enrich
			return nil
		}

		// Extract search pattern
		var pattern string
		if p, ok := payload.ToolInput["pattern"].(string); ok {
			pattern = p
		} else if p, ok := payload.ToolInput["query"].(string); ok {
			pattern = p
		}
		if pattern == "" {
			return nil
		}

		// Call goatlas search_code via MCP or direct — for simplicity,
		// we output a hint that the AI can use
		fmt.Fprintf(os.Stdout, "\n--- GoAtlas Semantic Context ---\n")
		fmt.Fprintf(os.Stdout, "Search pattern: %s\n", pattern)
		fmt.Fprintf(os.Stdout, "Tip: Use `search_code` MCP tool with query=%q mode=semantic for deeper results.\n", pattern)
		fmt.Fprintf(os.Stdout, "--- End GoAtlas Context ---\n")

		return nil
	},
}

// hookPostCmd handles PostToolUse: when Claude writes/edits a file, trigger incremental re-index.
var hookPostCmd = &cobra.Command{
	Use:   "post",
	Short: "PostToolUse hook: trigger incremental re-index after file edits",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}

		var payload struct {
			ToolName  string         `json:"tool_name"`
			ToolInput map[string]any `json:"tool_input"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil
		}

		toolName := strings.ToLower(payload.ToolName)
		if toolName != "write" && toolName != "edit" && toolName != "write_to_file" && toolName != "replace_file_content" && toolName != "multi_replace_file_content" {
			return nil
		}

		// Extract file path
		var filePath string
		if p, ok := payload.ToolInput["file_path"].(string); ok {
			filePath = p
		} else if p, ok := payload.ToolInput["path"].(string); ok {
			filePath = p
		} else if p, ok := payload.ToolInput["TargetFile"].(string); ok {
			filePath = p
		}
		if filePath == "" {
			return nil
		}

		// Detect repo root from git
		repoRoot, err := detectRepoRoot(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goatlas hook post: cannot detect repo root: %v\n", err)
			return nil
		}

		// Trigger incremental index in background
		indexCmd := exec.Command("goatlas", "index", "--incremental", repoRoot)
		indexCmd.Stdout = nil
		indexCmd.Stderr = nil
		if err := indexCmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "goatlas hook post: failed to start index: %v\n", err)
			return nil
		}

		// Don't wait — let it run in background
		go func() { _ = indexCmd.Wait() }()

		return nil
	},
}

// detectRepoRoot finds the git root for a file path.
func detectRepoRoot(filePath string) (string, error) {
	// Use directory of the file for git detection
	dir := filePath
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		dir = filepath.Dir(filePath)
	}

	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func init() {
	hookCmd.AddCommand(hookPreCmd)
	hookCmd.AddCommand(hookPostCmd)
}
