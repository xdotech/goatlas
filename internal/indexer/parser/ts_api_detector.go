package parser

import (
	"bufio"
	"os"
	"regexp"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// DetectTSAPIConnections scans a TypeScript file for API service references
// using regex patterns from the config. Returns detected connections.
func DetectTSAPIConnections(filePath string, patterns []TSAPIPattern) ([]domain.ServiceConnection, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Pre-compile regexes
	type compiledPattern struct {
		re       *regexp.Regexp
		connType string
	}
	compiled := make([]compiledPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue // skip invalid patterns
		}
		compiled = append(compiled, compiledPattern{re: re, connType: p.ConnType})
	}

	if len(compiled) == 0 {
		return nil, nil
	}

	var connections []domain.ServiceConnection
	seen := map[string]struct{}{} // dedupe same service in same file

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, cp := range compiled {
			matches := cp.re.FindStringSubmatch(line)
			if len(matches) < 2 {
				continue
			}
			serviceName := matches[1] // captured group = service name

			// Deduplicate: only one connection per service per file
			if _, exists := seen[serviceName]; exists {
				continue
			}
			seen[serviceName] = struct{}{}

			connections = append(connections, domain.ServiceConnection{
				ConnType: cp.connType,
				Target:   serviceName,
				Line:     lineNum,
			})
		}
	}

	return connections, scanner.Err()
}
