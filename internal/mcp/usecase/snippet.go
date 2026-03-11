package usecase

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ExtractSnippet reads source lines around the given line number from a file.
// It returns up to maxLines of code starting from (startLine - contextBefore).
// For structs/interfaces, it tries to capture the full definition.
// For functions, it shows the first N lines.
func ExtractSnippet(repoRoot, relPath string, line, maxLines int) string {
	if repoRoot == "" || relPath == "" || line <= 0 {
		return ""
	}
	absPath := filepath.Join(repoRoot, relPath)
	f, err := os.Open(absPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Read all lines from file
	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if scanner.Err() != nil || len(allLines) == 0 {
		return ""
	}

	// Convert to 0-indexed
	startIdx := line - 1
	if startIdx >= len(allLines) {
		return ""
	}

	// Determine snippet range: start 1 line before symbol, capture up to maxLines
	from := startIdx - 1
	if from < 0 {
		from = 0
	}

	// Try to find end of definition (closing brace at indentation level 0)
	to := from + maxLines
	if to > len(allLines) {
		to = len(allLines)
	}

	// For function/struct/interface bodies: try to find the matching closing brace
	// but cap at maxLines to avoid huge snippets
	braceCount := 0
	foundOpen := false
	for i := startIdx; i < len(allLines) && i < startIdx+maxLines; i++ {
		trimmed := strings.TrimSpace(allLines[i])
		for _, ch := range trimmed {
			if ch == '{' {
				braceCount++
				foundOpen = true
			} else if ch == '}' {
				braceCount--
			}
		}
		if foundOpen && braceCount <= 0 {
			to = i + 1
			break
		}
	}

	// Enforce max lines
	if to-from > maxLines {
		to = from + maxLines
	}

	lines := allLines[from:to]

	// If we truncated, add ellipsis
	truncated := to < len(allLines) && (foundOpen && braceCount > 0)

	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString("  " + l + "\n")
	}
	if truncated {
		sb.WriteString("  ...\n")
	}

	return sb.String()
}
