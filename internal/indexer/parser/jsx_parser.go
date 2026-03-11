package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// JSX/TS symbol kinds
const (
	KindComponent = "component"
	KindHook      = "hook"
	KindInterface = "interface"
	KindTypeAlias = "type_alias"
)

var (
	// Function/arrow components — capital first letter, not a hook
	reFuncComponent  = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?function\s+([A-Z][a-zA-Z0-9_]*)`)
	reArrowComponent = regexp.MustCompile(`^(?:export\s+)?(?:const|let)\s+([A-Z][a-zA-Z0-9_]*)\s*(?::\s*(?:React\.)?(?:FC|FunctionComponent|ComponentType|NamedExoticComponent|VFC|ReactNode|JSX\.Element)[^=]*)?\s*=\s*(?:\(|async\s*\(|React\.memo|React\.forwardRef)`)
	reClassComponent = regexp.MustCompile(`^(?:export\s+)?(?:default\s+)?class\s+([A-Z][a-zA-Z0-9_]*)\s+extends\s+(?:React\.)?(?:Component|PureComponent)`)

	// Custom hooks — useXxx pattern
	reFuncHook  = regexp.MustCompile(`^(?:export\s+)?function\s+(use[A-Z][a-zA-Z0-9_]*)`)
	reArrowHook = regexp.MustCompile(`^(?:export\s+)?(?:const|let)\s+(use[A-Z][a-zA-Z0-9_]*)\s*(?::[^=]*)?\s*=\s*(?:\(|async\s*\()`)

	// TypeScript interfaces and type aliases
	reInterface = regexp.MustCompile(`^(?:export\s+)?interface\s+([A-Z][a-zA-Z0-9_]*)`)
	reTypeAlias = regexp.MustCompile(`^(?:export\s+)?type\s+([A-Z][a-zA-Z0-9_]*)\s*(?:<[^>]*>)?\s*=`)

	// Import path extraction — matches the 'from ...' part on any line
	reFromPath = regexp.MustCompile(`from\s+['"]([^'"]+)['"]`)
	// Direct import (no 'from'): import 'side-effect-module'
	reDirectImport = regexp.MustCompile(`^import\s+['"]([^'"]+)['"]`)

	// JSDoc / single-line comment just above a declaration
	reComment = regexp.MustCompile(`^\s*(?:///?|/\*\*?|\*)\s*(.+)`)
)

// ParseJSXFile parses a .tsx/.ts/.jsx/.js file and extracts symbols and imports.
// Symbol kinds produced: "component", "hook", "interface", "type_alias".
func ParseJSXFile(filePath string) (*ParseResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer f.Close()

	result := &ParseResult{}
	pkgName := moduleNameFromPath(filePath)

	var pendingDoc strings.Builder
	lineNum := 0

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Accumulate doc comments
		if m := reComment.FindStringSubmatch(line); m != nil && isCommentLine(line) {
			if pendingDoc.Len() > 0 {
				pendingDoc.WriteByte(' ')
			}
			pendingDoc.WriteString(strings.TrimSpace(m[1]))
			continue
		}

		doc := pendingDoc.String()
		pendingDoc.Reset()

		// Skip blank lines and non-declaration lines after resetting doc
		if line == "" {
			continue
		}

		// Imports
		if strings.HasPrefix(line, "import ") {
			if paths := extractImportPaths(line); len(paths) > 0 {
				for _, p := range paths {
					result.Imports = append(result.Imports, domain.Import{ImportPath: p})
				}
			}
			continue
		}

		// Custom hooks (check before components since hooks start with 'use')
		if sym, ok := matchSymbol(line, reFuncHook, KindHook, pkgName, lineNum, doc); ok {
			result.Symbols = append(result.Symbols, sym)
			continue
		}
		if sym, ok := matchSymbol(line, reArrowHook, KindHook, pkgName, lineNum, doc); ok {
			result.Symbols = append(result.Symbols, sym)
			continue
		}

		// React components
		if sym, ok := matchSymbol(line, reFuncComponent, KindComponent, pkgName, lineNum, doc); ok {
			result.Symbols = append(result.Symbols, sym)
			continue
		}
		if sym, ok := matchSymbol(line, reArrowComponent, KindComponent, pkgName, lineNum, doc); ok {
			result.Symbols = append(result.Symbols, sym)
			continue
		}
		if sym, ok := matchSymbol(line, reClassComponent, KindComponent, pkgName, lineNum, doc); ok {
			sym.Signature = "class component"
			result.Symbols = append(result.Symbols, sym)
			continue
		}

		// TypeScript interfaces and type aliases
		if sym, ok := matchSymbol(line, reInterface, KindInterface, pkgName, lineNum, doc); ok {
			result.Symbols = append(result.Symbols, sym)
			continue
		}
		if sym, ok := matchSymbol(line, reTypeAlias, KindTypeAlias, pkgName, lineNum, doc); ok {
			result.Symbols = append(result.Symbols, sym)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", filePath, err)
	}

	result.Module = pkgName
	return result, nil
}

// matchSymbol tries to match a regex against line, returning a Symbol on success.
func matchSymbol(line string, re *regexp.Regexp, kind, pkg string, lineNum int, doc string) (domain.Symbol, bool) {
	m := re.FindStringSubmatch(line)
	if m == nil {
		return domain.Symbol{}, false
	}
	name := m[1]
	return domain.Symbol{
		Kind:          kind,
		Name:          name,
		QualifiedName: pkg + "." + name,
		Signature:     summarizeDeclaration(line),
		Line:          lineNum,
		DocComment:    doc,
	}, true
}

// extractImportPaths returns all module paths from an import line.
func extractImportPaths(line string) []string {
	if m := reFromPath.FindStringSubmatch(line); m != nil {
		return []string{m[1]}
	}
	if m := reDirectImport.FindStringSubmatch(line); m != nil {
		return []string{m[1]}
	}
	return nil
}

// isCommentLine returns true if the line is a JS/TS comment.
func isCommentLine(line string) bool {
	return strings.HasPrefix(line, "//") ||
		strings.HasPrefix(line, "/*") ||
		strings.HasPrefix(line, "*") ||
		strings.HasPrefix(line, "/**")
}

// summarizeDeclaration trims a declaration line to a readable signature.
func summarizeDeclaration(line string) string {
	// Truncate at opening brace or arrow body
	for _, sep := range []string{"{", "=>", "= React.memo", "= React.forwardRef"} {
		if i := strings.Index(line, sep); i > 0 {
			line = strings.TrimSpace(line[:i])
			break
		}
	}
	if len(line) > 120 {
		line = line[:120] + "..."
	}
	return line
}

// moduleNameFromPath derives a pseudo-module name from the file's directory.
// e.g. src/screens/HomeScreen.tsx → screens
func moduleNameFromPath(filePath string) string {
	parts := strings.Split(filePath, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	if len(parts) == 1 {
		name := parts[0]
		if idx := strings.LastIndex(name, "."); idx > 0 {
			return name[:idx]
		}
		return name
	}
	return "unknown"
}
