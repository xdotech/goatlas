package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// hooksCmd is the parent for "goatlas hooks install|uninstall".
var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Install or uninstall Claude Code hooks for a repository",
	Long:  `Manage Claude Code hook configurations in .claude/settings.json for automatic GoAtlas integration.`,
}

var hooksInstallCmd = &cobra.Command{
	Use:   "install <repo-path>",
	Short: "Install Claude Code hooks (PreToolUse + PostToolUse) into a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		claudeDir := filepath.Join(repoPath, ".claude")
		settingsFile := filepath.Join(claudeDir, "settings.json")

		// Create .claude/ directory if needed
		if err := os.MkdirAll(claudeDir, 0o755); err != nil {
			return fmt.Errorf("create .claude directory: %w", err)
		}

		// Load existing settings or start fresh
		settings := make(map[string]any)
		if data, err := os.ReadFile(settingsFile); err == nil {
			if err := json.Unmarshal(data, &settings); err != nil {
				return fmt.Errorf("parse existing settings.json: %w", err)
			}
		}

		// Build hook entries
		preHook := map[string]any{
			"matcher": "Grep|Glob",
			"hooks": []map[string]any{
				{"type": "command", "command": "goatlas hook pre"},
			},
		}
		postHook := map[string]any{
			"matcher": "Write|Edit",
			"hooks": []map[string]any{
				{"type": "command", "command": "goatlas hook post"},
			},
		}

		// Merge into existing hooks
		hooks, _ := settings["hooks"].(map[string]any)
		if hooks == nil {
			hooks = make(map[string]any)
		}

		// Set PreToolUse — replace any existing GoAtlas entries
		hooks["PreToolUse"] = mergeHookEntries(hooks["PreToolUse"], preHook, "goatlas hook pre")
		hooks["PostToolUse"] = mergeHookEntries(hooks["PostToolUse"], postHook, "goatlas hook post")

		settings["hooks"] = hooks

		// Write back
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
		if err := os.WriteFile(settingsFile, data, 0o644); err != nil {
			return fmt.Errorf("write settings: %w", err)
		}

		fmt.Printf("✅ GoAtlas Claude Code hooks installed in %s\n", settingsFile)
		fmt.Println("   PreToolUse:  Grep|Glob → goatlas hook pre")
		fmt.Println("   PostToolUse: Write|Edit → goatlas hook post")
		return nil
	},
}

var hooksUninstallCmd = &cobra.Command{
	Use:   "uninstall <repo-path>",
	Short: "Remove GoAtlas Claude Code hooks from a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("resolve path: %w", err)
		}

		settingsFile := filepath.Join(repoPath, ".claude", "settings.json")
		data, err := os.ReadFile(settingsFile)
		if err != nil {
			fmt.Println("No .claude/settings.json found — nothing to uninstall.")
			return nil
		}

		settings := make(map[string]any)
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse settings: %w", err)
		}

		hooks, _ := settings["hooks"].(map[string]any)
		if hooks == nil {
			fmt.Println("No hooks found in settings — nothing to uninstall.")
			return nil
		}

		// Remove GoAtlas entries from PreToolUse and PostToolUse
		hooks["PreToolUse"] = removeGoatlasEntries(hooks["PreToolUse"])
		hooks["PostToolUse"] = removeGoatlasEntries(hooks["PostToolUse"])
		settings["hooks"] = hooks

		out, _ := json.MarshalIndent(settings, "", "  ")
		if err := os.WriteFile(settingsFile, out, 0o644); err != nil {
			return fmt.Errorf("write settings: %w", err)
		}

		fmt.Printf("✅ GoAtlas hooks removed from %s\n", settingsFile)
		return nil
	},
}

// mergeHookEntries adds a GoAtlas hook entry to existing entries, replacing any previous GoAtlas entry.
func mergeHookEntries(existing any, newEntry map[string]any, goatlasCmd string) []any {
	var entries []any

	// Keep non-GoAtlas entries
	if arr, ok := existing.([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if !containsGoatlasHook(m, goatlasCmd) {
					entries = append(entries, m)
				}
			}
		}
	}

	// Add new GoAtlas entry
	entries = append(entries, newEntry)
	return entries
}

// removeGoatlasEntries removes all entries containing GoAtlas hook commands.
func removeGoatlasEntries(existing any) []any {
	var entries []any
	if arr, ok := existing.([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if !containsGoatlasHook(m, "goatlas hook") {
					entries = append(entries, m)
				}
			}
		}
	}
	if entries == nil {
		entries = []any{}
	}
	return entries
}

// containsGoatlasHook checks if a hook entry contains a GoAtlas command.
func containsGoatlasHook(entry map[string]any, cmdPrefix string) bool {
	if hooks, ok := entry["hooks"].([]any); ok {
		for _, h := range hooks {
			if hm, ok := h.(map[string]any); ok {
				if cmd, ok := hm["command"].(string); ok {
					if len(cmd) >= len(cmdPrefix) && cmd[:len(cmdPrefix)] == cmdPrefix {
						return true
					}
				}
			}
		}
	}
	return false
}

func init() {
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)
}
