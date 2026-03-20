package parser

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// Common HTTP method call patterns in TypeScript/JavaScript.
// Matches: api.get(...), api.post(...), axios.put(...), http.delete(...), etc.
var httpMethodCallRe = regexp.MustCompile(`(?i)\b(?:api|axios|http|fetch|request)\s*\.\s*(get|post|put|patch|delete)\s*[(<]`)

// API path literal in string.
// Matches: '/api/v1/users', `/api/v1/items/${id}`, "SVC_PREFIX + '/items'"
var apiPathLiteralRe = regexp.MustCompile(`['"\x60](/[a-zA-Z0-9/_\-{}$]+)['"\x60]`)

// Template literal path.
var templatePathRe = regexp.MustCompile(`\x60([^` + "`" + `]*?/api/[^` + "`" + `]*?)\x60`)

// Template literal that starts with ${Const.VAR}/... (common in PMC projects).
// Matches: `${MasterDataApi.REASON}/${uid}`, `${ApiConstants.ITEMS}/${id}`
var templateConstPathRe = regexp.MustCompile(`\x60\$\{([A-Za-z_][A-Za-z0-9_]*\.[A-Z_][A-Z0-9_]*)\}([^` + "`" + `]*)\x60`)

// Constant reference as API path argument (no string literal).
// Matches the first argument: api.get(MasterDataApi.REASON, ...) or api.get(generatePath(MasterDataApi.ITEM_DETAIL, ...))
// Captures: "MasterDataApi.REASON", "ApiConstants.ITEMS", etc.
var constRefArgRe = regexp.MustCompile(`\.\s*(?:get|post|put|patch|delete)\s*\(\s*(?:generatePath\s*\(\s*)?([A-Za-z_][A-Za-z0-9_]*\.[A-Z_][A-Z0-9_]*)`)

// React component detection patterns.
// Matches: function ComponentName(, const ComponentName =, export default function ComponentName
var reactComponentRe = regexp.MustCompile(`(?:(?:export\s+(?:default\s+)?)?function|const)\s+([A-Z][a-zA-Z0-9]*)\s*[=(]`)

// React hook detection pattern.
// Matches: const useActivityTimeline = ..., function useActivityTimeline(, export function useFoo(
var reactHookRe = regexp.MustCompile(`(?:export\s+)?(?:const|function)\s+(use[A-Z][a-zA-Z0-9]*)\s*[=(]`)

// DetectComponentAPICalls scans a TS/JSX file and detects which React component
// makes which API calls. This is component-level detection (not file-level).
func DetectComponentAPICalls(filePath string, patterns []TSAPIPattern) ([]domain.ComponentAPICall, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Pre-compile custom patterns from config
	type compiledCustom struct {
		re          *regexp.Regexp
		serviceName string
	}
	var customPatterns []compiledCustom
	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue
		}
		customPatterns = append(customPatterns, compiledCustom{re: re, serviceName: p.ConnType})
	}

	var allCalls []domain.ComponentAPICall
	currentComponent := ""
	braceDepth := 0
	componentBraceStart := 0

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track current React component scope (PascalCase functions)
		if matches := reactComponentRe.FindStringSubmatch(line); len(matches) >= 2 {
			candidateName := matches[1]
			// Heuristic: React component names are PascalCase and typically > 2 chars
			if len(candidateName) > 2 && candidateName[0] >= 'A' && candidateName[0] <= 'Z' {
				currentComponent = candidateName
				componentBraceStart = braceDepth
			}
		}

		// Also track React hook scope (useXxx pattern — camelCase starting with "use")
		if currentComponent == "" {
			if hookMatches := reactHookRe.FindStringSubmatch(line); len(hookMatches) >= 2 {
				currentComponent = hookMatches[1]
				componentBraceStart = braceDepth
			}
		}

		// Track brace depth for scope detection
		for _, ch := range trimmed {
			if ch == '{' {
				braceDepth++
			} else if ch == '}' {
				braceDepth--
				if braceDepth <= componentBraceStart && currentComponent != "" {
					currentComponent = "" // exited component scope
				}
			}
		}

		// If no component context, use file-level fallback
		componentName := currentComponent
		if componentName == "" {
			componentName = "__file__"
		}

		// Detect HTTP method calls: api.get(), api.post(), etc.
		if httpMatches := httpMethodCallRe.FindStringSubmatch(line); len(httpMatches) >= 2 {
			method := strings.ToUpper(httpMatches[1])

			// Try to extract API path from the same line, ordered by specificity
			apiPath := ""
			targetService := ""

			// 1. Literal string path: '/api/v1/users'
			if pathMatches := apiPathLiteralRe.FindStringSubmatch(line); len(pathMatches) >= 2 {
				apiPath = pathMatches[1]
			}

			// 2. Template literal with /api/ in it: `${prefix}/api/v1/users`
			if apiPath == "" {
				if pathMatches := templatePathRe.FindStringSubmatch(line); len(pathMatches) >= 2 {
					apiPath = pathMatches[1]
				}
			}

			// 3. Template literal starting with ${Const.VAR}: `${MasterDataApi.REASON}/${uid}`
			if apiPath == "" {
				if pathMatches := templateConstPathRe.FindStringSubmatch(line); len(pathMatches) >= 2 {
					apiPath = pathMatches[1] // e.g. "MasterDataApi.REASON"
					if len(pathMatches) >= 3 && pathMatches[2] != "" {
						apiPath += pathMatches[2] // append suffix like "/${uid}"
					}
				}
			}

			// 4. Constant reference as argument: api.get(MasterDataApi.REASON, ...)
			if apiPath == "" {
				if pathMatches := constRefArgRe.FindStringSubmatch(line); len(pathMatches) >= 2 {
					apiPath = pathMatches[1] // e.g. "MasterDataApi.REASON"
				}
			}

			// Try custom patterns for service resolution
			for _, cp := range customPatterns {
				if cpMatches := cp.re.FindStringSubmatch(line); len(cpMatches) >= 2 {
					targetService = cpMatches[1]
					if apiPath == "" {
						apiPath = cpMatches[0]
					}
				}
			}

			if apiPath != "" || targetService != "" {
				// Normalize path segments so dynamic IDs become {id} for better matching
				if apiPath != "" && strings.HasPrefix(apiPath, "/") {
					apiPath = NormalizePath(apiPath)
				}
				call := domain.ComponentAPICall{
					Component:     componentName,
					HttpMethod:    method,
					APIPath:       apiPath,
					TargetService: targetService,
					Line:          lineNum,
				}
				allCalls = append(allCalls, call)
			}
		}
	}

	return allCalls, scanner.Err()
}
