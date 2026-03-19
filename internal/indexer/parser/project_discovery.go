package parser

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ProjectDeps holds discovered dependency identifiers per ecosystem.
type ProjectDeps struct {
	GoModules      []string          // e.g. ["google.golang.org/grpc", "github.com/segmentio/kafka-go"]
	NPMPkgs        []string          // e.g. ["kafkajs", "@grpc/grpc-js", "axios"]
	PyPkgs         []string          // e.g. ["grpcio", "kafka-python", "httpx"]
	MavenPkgs      []string          // e.g. ["io.grpc:grpc-stub", "org.springframework.kafka:spring-kafka"]
	TSConfigAliases map[string]string // alias prefix → target prefix, e.g. {"@/": "src/"}
}

// DiscoverDependencies reads root-level manifest files from repoPath.
// Missing or malformed files are silently skipped.
func DiscoverDependencies(repoPath string) (*ProjectDeps, error) {
	deps := &ProjectDeps{
		TSConfigAliases: make(map[string]string),
	}

	if mods, err := parseGoMod(filepath.Join(repoPath, "go.mod")); err == nil {
		deps.GoModules = mods
	}
	if pkgs, err := parsePackageJSON(filepath.Join(repoPath, "package.json")); err == nil {
		deps.NPMPkgs = pkgs
	}
	if pkgs, err := parseRequirementsTxt(filepath.Join(repoPath, "requirements.txt")); err == nil {
		deps.PyPkgs = pkgs
	}
	if pkgs, err := parsePomXML(filepath.Join(repoPath, "pom.xml")); err == nil {
		deps.MavenPkgs = pkgs
	}
	if aliases, err := parseTsConfig(repoPath); err == nil {
		deps.TSConfigAliases = aliases
	}

	return deps, nil
}

// parseGoMod extracts module paths from go.mod require blocks.
// Handles both block and single-line requires. Ignores indirect deps.
func parseGoMod(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var mods []string
	inBlock := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Strip inline comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)

		if line == "require (" {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}

		if inBlock {
			// "google.golang.org/grpc v1.64.0"
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				mods = append(mods, fields[0])
			}
		} else if strings.HasPrefix(line, "require ") {
			// single-line: "require google.golang.org/grpc v1.64.0"
			rest := strings.TrimPrefix(line, "require ")
			rest = strings.TrimSpace(rest)
			fields := strings.Fields(rest)
			if len(fields) >= 2 {
				mods = append(mods, fields[0])
			}
		}
	}
	return mods, scanner.Err()
}

// parsePackageJSON extracts package names from dependencies + devDependencies.
func parsePackageJSON(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var pkgs []string
	for name := range pkg.Dependencies {
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			pkgs = append(pkgs, name)
		}
	}
	for name := range pkg.DevDependencies {
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			pkgs = append(pkgs, name)
		}
	}
	return pkgs, nil
}

// parseRequirementsTxt extracts package names from requirements.txt.
// Strips version specifiers and skips comments, -r includes, and git+ refs.
var reVersionSep = regexp.MustCompile(`[=<>!~;]`)

func parseRequirementsTxt(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "-r") || strings.HasPrefix(line, "-e") ||
			strings.HasPrefix(line, "git+") {
			continue
		}
		// Strip inline comments
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		// Strip version specifier
		name := reVersionSep.Split(line, 2)[0]
		name = strings.TrimSpace(strings.ToLower(name))
		if name != "" {
			pkgs = append(pkgs, name)
		}
	}
	return pkgs, scanner.Err()
}

// parsePomXML extracts groupId:artifactId pairs from pom.xml dependency blocks.
func parsePomXML(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []string
	var groupID, artifactID string
	inDep := false

	reTag := regexp.MustCompile(`<(\w+)>([^<]+)</\w+>`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "<dependency>") {
			inDep = true
			groupID = ""
			artifactID = ""
			continue
		}
		if strings.Contains(line, "</dependency>") {
			if inDep && groupID != "" && artifactID != "" {
				pkgs = append(pkgs, groupID+":"+artifactID)
			}
			inDep = false
			continue
		}
		if !inDep {
			continue
		}

		m := reTag.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch m[1] {
		case "groupId":
			groupID = strings.TrimSpace(m[2])
		case "artifactId":
			artifactID = strings.TrimSpace(m[2])
		}
	}
	return pkgs, scanner.Err()
}

// parseTsConfig extracts compilerOptions.paths from tsconfig.json (or fallbacks).
// Returns alias prefix → target prefix with glob wildcards stripped.
// e.g. {"@/*": ["src/*"]} → {"@/": "src/"}
func parseTsConfig(repoPath string) (map[string]string, error) {
	candidates := []string{"tsconfig.json", "tsconfig.app.json", "tsconfig.base.json"}

	for _, name := range candidates {
		path := filepath.Join(repoPath, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Strip JS comments before JSON parsing
		cleaned := stripJSComments(string(data))

		var raw map[string]json.RawMessage
		if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
			continue
		}

		compOpts, ok := raw["compilerOptions"]
		if !ok {
			continue
		}

		var opts map[string]json.RawMessage
		if err := json.Unmarshal(compOpts, &opts); err != nil {
			continue
		}

		pathsRaw, ok := opts["paths"]
		if !ok {
			continue
		}

		var paths map[string][]string
		if err := json.Unmarshal(pathsRaw, &paths); err != nil {
			continue
		}

		result := make(map[string]string, len(paths))
		for alias, targets := range paths {
			if len(targets) == 0 {
				continue
			}
			// "@/*" → "@/", "src/*" → "src/"  (strip trailing * only)
			aliasPrefix := strings.TrimSuffix(alias, "*")
			targetPrefix := strings.TrimSuffix(targets[0], "*")
			result[aliasPrefix] = targetPrefix
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	return map[string]string{}, nil
}

// stripJSComments removes // and /* */ comments from a JSON-like string.
// Correctly skips content inside string literals.
func stripJSComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		// Handle string literals — don't strip comments inside strings
		if s[i] == '"' {
			b.WriteByte(s[i])
			i++
			for i < len(s) {
				b.WriteByte(s[i])
				if s[i] == '\\' && i+1 < len(s) {
					// Escaped char — write both and skip
					i++
					b.WriteByte(s[i])
				} else if s[i] == '"' {
					// End of string
					i++
					break
				}
				i++
			}
			continue
		}
		// Line comment
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '/' {
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
