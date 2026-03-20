package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xdotech/goatlas/internal/config"
	"github.com/xdotech/goatlas/internal/db"
	"github.com/xdotech/goatlas/internal/indexer"
	mcpusecase "github.com/xdotech/goatlas/internal/mcp/usecase"
	"github.com/xdotech/goatlas/internal/mcp/registry"
	"github.com/spf13/cobra"
)

// hookCmd is the parent for "goatlas hook pre" and "goatlas hook post".
var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Claude Code hook handlers (PreToolUse / PostToolUse)",
	Long:  `These subcommands are called by Claude Code hooks. They read tool input/output from stdin and write enrichment to stdout.`,
}

// hookPreOutput is the JSON structure Claude Code expects for additionalContext injection.
type hookPreOutput struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// hookPreCmd handles PreToolUse: when Claude calls Grep/Glob, enrich with semantic context.
var hookPreCmd = &cobra.Command{
	Use:   "pre",
	Short: "PreToolUse hook: enrich Grep/Glob results with semantic context",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil // fail silently, don't block Claude
		}

		var payload struct {
			ToolName  string         `json:"tool_name"`
			ToolInput map[string]any `json:"tool_input"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil
		}

		toolName := strings.ToLower(payload.ToolName)
		if toolName != "grep" && toolName != "glob" {
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

		// Use a short timeout so we never block Claude
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		context_ := searchSymbols(ctx, pattern)
		if context_ == "" {
			return nil
		}

		out := hookPreOutput{}
		out.HookSpecificOutput.HookEventName = "PreToolUse"
		out.HookSpecificOutput.AdditionalContext = context_

		return json.NewEncoder(os.Stdout).Encode(out)
	},
}

// searchSymbols queries GoAtlas DB for symbols matching pattern and returns formatted context.
func searchSymbols(ctx context.Context, pattern string) string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		return ""
	}
	defer pool.Close()

	indexerSvc := indexer.NewService(pool)
	uc := mcpusecase.NewFindSymbolUseCase(indexerSvc.SymbolRepo, cfg.RepoPath)

	result, err := uc.Execute(ctx, pattern, "")
	if err != nil || result == "" {
		return ""
	}

	return fmt.Sprintf("GoAtlas semantic context for %q:\n%s", pattern, result)
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
		if toolName != "write" && toolName != "edit" && toolName != "multiedit" &&
			toolName != "write_to_file" && toolName != "replace_file_content" && toolName != "multi_replace_file_content" {
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

// hookSessionCmd handles SessionStart: inject indexed repo context at conversation start.
var hookSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "SessionStart hook: inject indexed repository context",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		cfg, err := config.Load()
		if err != nil {
			return nil
		}

		pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return nil
		}
		defer pool.Close()

		reg := registry.NewRepoRegistry(pool)
		repos, err := reg.List(ctx)
		if err != nil || len(repos) == 0 {
			return nil
		}

		var sb strings.Builder
		sb.WriteString("GoAtlas indexed repositories:\n")
		for _, r := range repos {
			lastIndexed := "never"
			if r.LastIndexedAt != nil {
				lastIndexed = r.LastIndexedAt.Format("2006-01-02 15:04")
			}
			sb.WriteString(fmt.Sprintf("  - %s (path: %s, last indexed: %s)\n", r.Name, r.Path, lastIndexed))
		}

		out := hookPreOutput{}
		out.HookSpecificOutput.HookEventName = "SessionStart"
		out.HookSpecificOutput.AdditionalContext = sb.String()

		return json.NewEncoder(os.Stdout).Encode(out)
	},
}

func init() {
	hookCmd.AddCommand(hookPreCmd)
	hookCmd.AddCommand(hookPostCmd)
	hookCmd.AddCommand(hookSessionCmd)
}
