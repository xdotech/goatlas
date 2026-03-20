package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// goatlasEnvKeys are the env vars GoAtlas needs, synced to ~/.claude/settings.json on install.
var goatlasEnvKeys = []string{
	"DATABASE_DSN",
	"NEO4J_URL",
	"NEO4J_USER",
	"NEO4J_PASS",
	"GEMINI_API_KEY",
	"LLM_PROVIDER",
	"EMBED_PROVIDER",
	"OLLAMA_URL",
	"OLLAMA_MODEL",
	"OLLAMA_EMBED_MODEL",
}

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

		// Resolve absolute path of this binary so hooks work regardless of PATH.
		binaryPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable path: %w", err)
		}

		// Build hook entries
		preHook := map[string]any{
			"matcher": "Grep|Glob",
			"hooks": []map[string]any{
				{"type": "command", "command": binaryPath + " hook pre"},
			},
		}
		postHook := map[string]any{
			"matcher": "Write|Edit|MultiEdit",
			"hooks": []map[string]any{
				{"type": "command", "command": binaryPath + " hook post"},
			},
		}

		// Merge into existing hooks
		hooks, _ := settings["hooks"].(map[string]any)
		if hooks == nil {
			hooks = make(map[string]any)
		}

		// Set PreToolUse — replace any existing GoAtlas entries
		hooks["PreToolUse"] = mergeHookEntries(hooks["PreToolUse"], preHook, "hook pre")
		hooks["PostToolUse"] = mergeHookEntries(hooks["PostToolUse"], postHook, "hook post")

		settings["hooks"] = hooks

		// Write back project settings
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
		if err := os.WriteFile(settingsFile, data, 0o644); err != nil {
			return fmt.Errorf("write settings: %w", err)
		}

		fmt.Printf("✅ GoAtlas Claude Code hooks installed in %s\n", settingsFile)
		fmt.Printf("   PreToolUse:  Grep|Glob           → %s hook pre\n", binaryPath)
		fmt.Printf("   PostToolUse: Write|Edit|MultiEdit → %s hook post\n", binaryPath)

		// Sync env vars to ~/.claude/settings.json so hooks work in any project
		if synced, err := syncEnvToClaudeSettings(repoPath); err != nil {
			fmt.Printf("⚠️  Could not sync env to ~/.claude/settings.json: %v\n", err)
		} else if len(synced) > 0 {
			home, _ := os.UserHomeDir()
			globalSettings := filepath.Join(home, ".claude", "settings.json")
			fmt.Printf("✅ GoAtlas env vars synced: %s\n", strings.Join(synced, ", "))
			fmt.Printf("   To change values later, edit the \"env\" section in: %s\n", globalSettings)
		}

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
		hooks["PreToolUse"] = removeGoatlasEntries(hooks["PreToolUse"], "goatlas hook")
		hooks["PostToolUse"] = removeGoatlasEntries(hooks["PostToolUse"], "goatlas hook")
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
// cmdSubstr is a substring matched against hook commands to identify existing GoAtlas entries.
func mergeHookEntries(existing any, newEntry map[string]any, cmdSubstr string) []any {
	var entries []any

	// Keep non-GoAtlas entries
	if arr, ok := existing.([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if !containsGoatlasHook(m, cmdSubstr) {
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
func removeGoatlasEntries(existing any, cmdSubstr string) []any {
	var entries []any
	if arr, ok := existing.([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				if !containsGoatlasHook(m, cmdSubstr) {
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

// containsGoatlasHook checks if a hook entry contains a command with the given substring.
func containsGoatlasHook(entry map[string]any, cmdSubstr string) bool {
	if hooks, ok := entry["hooks"].([]any); ok {
		for _, h := range hooks {
			if hm, ok := h.(map[string]any); ok {
				if cmd, ok := hm["command"].(string); ok {
					if strings.Contains(cmd, cmdSubstr) {
						return true
					}
				}
			}
		}
	}
	return false
}

// promptConfig defines a required config value to prompt for interactively.
type promptConfig struct {
	key         string
	label       string
	defaultVal  string
	required    bool
}

// goatlasPrompts are the values we prompt for when no .env or env vars are present.
var goatlasPrompts = []promptConfig{
	{key: "DATABASE_DSN", label: "PostgreSQL DSN", defaultVal: "postgres://goatlas:goatlas@localhost:5432/goatlas", required: true},
	{key: "GEMINI_API_KEY", label: "Gemini API key (leave blank to use Ollama)", required: false},
	{key: "NEO4J_URL", label: "Neo4j URL", defaultVal: "bolt://localhost:7687", required: false},
	{key: "NEO4J_USER", label: "Neo4j user", defaultVal: "neo4j", required: false},
	{key: "NEO4J_PASS", label: "Neo4j password", defaultVal: "goatlas_neo4j", required: false},
}

// placeholderValues are values that should be treated as unconfigured.
var placeholderValues = map[string]bool{
	"your_gemini_api_key_here": true,
	"/path/to/your/repo":       true,
}

// syncEnvToClaudeSettings reads env vars from the repo's .env file (falling back to process env,
// then interactive prompts) and writes them into ~/.claude/settings.json so GoAtlas hooks work
// in any project. Returns the list of keys that were synced.
func syncEnvToClaudeSettings(repoPath string) ([]string, error) {
	// 1. Collect values: .env file takes priority, then process environment
	envVals := make(map[string]string)
	dotEnvPath := filepath.Join(repoPath, ".env")
	if data, err := os.ReadFile(dotEnvPath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			envVals[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	for _, key := range goatlasEnvKeys {
		if _, ok := envVals[key]; !ok {
			if v := os.Getenv(key); v != "" {
				envVals[key] = v
			}
		}
	}

	// 2. Check if required values are missing or placeholder — prompt interactively if so
	needsPrompt := false
	for _, p := range goatlasPrompts {
		v := envVals[p.key]
		if p.required && (v == "" || placeholderValues[v]) {
			needsPrompt = true
			break
		}
	}

	if needsPrompt {
		fmt.Println("\n🔧 GoAtlas configuration — press Enter to accept defaults:")
		reader := bufio.NewReader(os.Stdin)
		for _, p := range goatlasPrompts {
			current := envVals[p.key]
			if placeholderValues[current] {
				current = ""
			}

			// Build prompt label
			display := p.label
			if current != "" {
				display = fmt.Sprintf("%s [%s]", p.label, current)
			} else if p.defaultVal != "" {
				display = fmt.Sprintf("%s [%s]", p.label, p.defaultVal)
			}
			fmt.Printf("   %s: ", display)

			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch {
			case input != "":
				envVals[p.key] = input
			case current != "":
				// keep existing
			case p.defaultVal != "":
				envVals[p.key] = p.defaultVal
			}
		}
	}

	// 3. Load ~/.claude/settings.json and merge
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	globalSettings := filepath.Join(home, ".claude", "settings.json")

	global := make(map[string]any)
	if data, err := os.ReadFile(globalSettings); err == nil {
		_ = json.Unmarshal(data, &global)
	}

	existing, _ := global["env"].(map[string]any)
	if existing == nil {
		existing = make(map[string]any)
	}

	var synced []string
	for _, key := range goatlasEnvKeys {
		val := envVals[key]
		if val == "" || placeholderValues[val] {
			continue
		}
		existing[key] = val
		synced = append(synced, key)
	}

	if len(synced) == 0 {
		return nil, nil
	}

	global["env"] = existing
	out, err := json.MarshalIndent(global, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal global settings: %w", err)
	}
	if err := os.WriteFile(globalSettings, out, 0o644); err != nil {
		return nil, fmt.Errorf("write global settings: %w", err)
	}
	return synced, nil
}

func init() {
	hooksCmd.AddCommand(hooksInstallCmd)
	hooksCmd.AddCommand(hooksUninstallCmd)
}
