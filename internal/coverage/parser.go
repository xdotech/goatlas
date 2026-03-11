package coverage

import (
	"bufio"
	"os"
	"strings"
)

// ParseSpecFile reads a Markdown file and splits it into feature sections.
// Looks for H2/H3 headers with "Feature:" prefix or "Feature " keyword.
func ParseSpecFile(path string) ([]FeatureSection, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var sections []FeatureSection
	var current *FeatureSection
	var contentLines []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if isFeatureHeader(line) {
			if current != nil {
				current.RawText = strings.Join(contentLines, "\n")
				sections = append(sections, *current)
			}
			name := extractFeatureName(line)
			current = &FeatureSection{Name: name}
			contentLines = []string{line}
			continue
		}

		if current != nil {
			contentLines = append(contentLines, line)
		}
	}

	// Add last section
	if current != nil {
		current.RawText = strings.Join(contentLines, "\n")
		sections = append(sections, *current)
	}

	return sections, scanner.Err()
}

func isFeatureHeader(line string) bool {
	trimmed := strings.TrimLeft(line, "#")
	trimmed = strings.TrimSpace(trimmed)
	lower := strings.ToLower(trimmed)
	return (strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ")) &&
		(strings.Contains(lower, "feature:") || strings.Contains(lower, "feature "))
}

func extractFeatureName(line string) string {
	name := strings.TrimLeft(line, "# ")
	name = strings.TrimPrefix(name, "Feature:")
	name = strings.TrimPrefix(name, "feature:")
	return strings.TrimSpace(name)
}
