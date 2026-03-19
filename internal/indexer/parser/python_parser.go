package parser

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// Python symbol kinds
const (
	KindClass  = "class"
	KindFunc   = "func"
	KindMethod = "method"
	KindConst  = "const"
	KindVar    = "var"
)

// Python API framework patterns for endpoint detection
var (
	reFlaskRoute   = regexp.MustCompile(`@(?:app|blueprint|bp)\.route\(\s*["']([^"']+)["']`)
	reFastAPIRoute = regexp.MustCompile(`@(?:app|router)\.(?:get|post|put|delete|patch|options|head)\(\s*["']([^"']+)["']`)
	reHTTPMethod   = regexp.MustCompile(`methods\s*=\s*\[([^\]]+)\]`)
	reFastAPIVerb  = regexp.MustCompile(`@(?:app|router)\.(get|post|put|delete|patch|options|head)\(`)
)

// ParsePythonFile parses a Python source file using tree-sitter and extracts symbols, imports, and API endpoints.
func ParsePythonFile(filePath string) (*ParseResult, error) {
	sourceCode, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
	}

	parser := tree_sitter.NewParser()
	defer parser.Close()

	lang := tree_sitter.NewLanguage(tree_sitter_python.Language())
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("set language: %w", err)
	}

	tree := parser.Parse(sourceCode, nil)
	defer tree.Close()

	root := tree.RootNode()
	result := &ParseResult{}
	pkgName := moduleNameFromPath(filePath)

	// Walk top-level children
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		extractPythonNode(child, sourceCode, pkgName, "", result)
	}

	result.Module = pkgName
	return result, nil
}

// extractPythonNode recursively extracts symbols from a tree-sitter AST node.
func extractPythonNode(node *tree_sitter.Node, source []byte, pkg, parentClass string, result *ParseResult) {
	switch node.Kind() {
	case "class_definition":
		extractPythonClass(node, source, pkg, result)

	case "function_definition":
		extractPythonFunction(node, source, pkg, parentClass, "", result)

	case "decorated_definition":
		extractPythonDecorated(node, source, pkg, parentClass, result)

	case "import_statement":
		extractPythonImport(node, source, result)

	case "import_from_statement":
		extractPythonFromImport(node, source, result)

	case "expression_statement":
		if parentClass == "" {
			extractPythonAssignment(node, source, pkg, result)
		}
	}
}

// extractPythonClass extracts a class definition and its methods.
func extractPythonClass(node *tree_sitter.Node, source []byte, pkg string, result *ParseResult) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	className := nodeContent(nameNode, source)
	startPoint := node.StartPosition()

	// Build signature
	sig := "class " + className
	superNode := node.ChildByFieldName("superclasses")
	if superNode != nil {
		sig += nodeContent(superNode, source)
	}

	doc := extractPythonDocstring(node, source)

	result.Symbols = append(result.Symbols, domain.Symbol{
		Kind:          KindClass,
		Name:          className,
		QualifiedName: pkg + "." + className,
		Signature:     sig,
		Line:          int(startPoint.Row) + 1,
		Col:           int(startPoint.Column) + 1,
		DocComment:    doc,
	})

	// Walk class body for methods
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		child := body.NamedChild(i)
		switch child.Kind() {
		case "function_definition":
			extractPythonFunction(child, source, pkg, className, "", result)
		case "decorated_definition":
			extractPythonDecorated(child, source, pkg, className, result)
		case "class_definition":
			extractPythonClass(child, source, pkg, result)
		}
	}
}

// extractPythonFunction extracts a function/method definition.
func extractPythonFunction(node *tree_sitter.Node, source []byte, pkg, parentClass, decoratorText string, result *ParseResult) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	funcName := nodeContent(nameNode, source)
	startPoint := node.StartPosition()

	kind := KindFunc
	receiver := ""
	if parentClass != "" {
		kind = KindMethod
		receiver = parentClass
	}

	// Build signature
	sig := "def " + funcName
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		sig += nodeContent(paramsNode, source)
	}
	retNode := node.ChildByFieldName("return_type")
	if retNode != nil {
		sig += " -> " + nodeContent(retNode, source)
	}

	doc := extractPythonDocstring(node, source)

	qualifiedName := pkg + "." + funcName
	if parentClass != "" {
		qualifiedName = pkg + ".(" + parentClass + ")." + funcName
	}

	result.Symbols = append(result.Symbols, domain.Symbol{
		Kind:          kind,
		Name:          funcName,
		QualifiedName: qualifiedName,
		Signature:     sig,
		Receiver:      receiver,
		Line:          int(startPoint.Row) + 1,
		Col:           int(startPoint.Column) + 1,
		DocComment:    doc,
	})

	// Check if decorator indicates an API endpoint
	if decoratorText != "" {
		endpoints := extractPythonEndpoints(decoratorText, funcName, int(startPoint.Row)+1)
		result.Endpoints = append(result.Endpoints, endpoints...)
	}
}

// extractPythonDecorated extracts a decorated definition.
func extractPythonDecorated(node *tree_sitter.Node, source []byte, pkg, parentClass string, result *ParseResult) {
	var decoratorTexts []string
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "decorator" {
			decoratorTexts = append(decoratorTexts, nodeContent(child, source))
		}
	}
	decoratorText := strings.Join(decoratorTexts, "\n")

	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch child.Kind() {
		case "function_definition":
			extractPythonFunction(child, source, pkg, parentClass, decoratorText, result)
		case "class_definition":
			extractPythonClass(child, source, pkg, result)
		}
	}
}

// extractPythonDocstring extracts the docstring from a function or class node.
func extractPythonDocstring(node *tree_sitter.Node, source []byte) string {
	body := node.ChildByFieldName("body")
	if body == nil || body.NamedChildCount() == 0 {
		return ""
	}

	firstStmt := body.NamedChild(0)
	if firstStmt == nil || firstStmt.Kind() != "expression_statement" {
		return ""
	}

	if firstStmt.NamedChildCount() == 0 {
		return ""
	}

	expr := firstStmt.NamedChild(0)
	if expr == nil || expr.Kind() != "string" {
		return ""
	}

	raw := nodeContent(expr, source)
	for _, quote := range []string{`"""`, `'''`} {
		if strings.HasPrefix(raw, quote) && strings.HasSuffix(raw, quote) && len(raw) >= 6 {
			raw = raw[3 : len(raw)-3]
			break
		}
	}
	for _, quote := range []string{`"`, `'`} {
		if strings.HasPrefix(raw, quote) && strings.HasSuffix(raw, quote) && len(raw) >= 2 {
			raw = raw[1 : len(raw)-1]
			break
		}
	}

	return strings.TrimSpace(raw)
}

// extractPythonImport extracts `import foo` / `import foo.bar` statements.
func extractPythonImport(node *tree_sitter.Node, source []byte, result *ParseResult) {
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		switch child.Kind() {
		case "dotted_name":
			result.Imports = append(result.Imports, domain.Import{
				ImportPath: nodeContent(child, source),
			})
		case "aliased_import":
			nameNode := child.ChildByFieldName("name")
			aliasNode := child.ChildByFieldName("alias")
			if nameNode != nil {
				imp := domain.Import{ImportPath: nodeContent(nameNode, source)}
				if aliasNode != nil {
					imp.Alias = nodeContent(aliasNode, source)
				}
				result.Imports = append(result.Imports, imp)
			}
		}
	}
}

// extractPythonFromImport extracts `from foo import bar` statements.
func extractPythonFromImport(node *tree_sitter.Node, source []byte, result *ParseResult) {
	moduleName := node.ChildByFieldName("module_name")
	modulePath := ""
	if moduleName != nil {
		modulePath = nodeContent(moduleName, source)
	}

	// Handle relative imports
	content := nodeContent(node, source)
	if strings.Contains(content, "from .") {
		idx := strings.Index(content, "from ")
		if idx >= 0 {
			rest := content[idx+5:]
			rest = strings.TrimSpace(rest)
			importIdx := strings.Index(rest, " import")
			if importIdx > 0 {
				modulePath = strings.TrimSpace(rest[:importIdx])
			}
		}
	}

	if modulePath == "" {
		modulePath = "."
	}

	result.Imports = append(result.Imports, domain.Import{
		ImportPath: modulePath,
	})
}

// extractPythonAssignment extracts module-level assignments as const (UPPER_CASE) or var.
func extractPythonAssignment(node *tree_sitter.Node, source []byte, pkg string, result *ParseResult) {
	if node.NamedChildCount() == 0 {
		return
	}

	child := node.NamedChild(0)
	if child == nil || child.Kind() != "assignment" {
		return
	}

	leftNode := child.ChildByFieldName("left")
	if leftNode == nil || leftNode.Kind() != "identifier" {
		return
	}

	name := nodeContent(leftNode, source)
	startPoint := node.StartPosition()

	// Skip dunder names
	if strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") {
		return
	}

	kind := KindVar
	if isUpperSnakeCase(name) {
		kind = KindConst
	}

	result.Symbols = append(result.Symbols, domain.Symbol{
		Kind:          kind,
		Name:          name,
		QualifiedName: pkg + "." + name,
		Line:          int(startPoint.Row) + 1,
		Col:           int(startPoint.Column) + 1,
	})
}

// extractPythonEndpoints detects API endpoints from decorator text.
func extractPythonEndpoints(decoratorText, handlerName string, line int) []domain.APIEndpoint {
	var endpoints []domain.APIEndpoint

	// FastAPI: @router.get("/path")
	if m := reFastAPIRoute.FindStringSubmatch(decoratorText); m != nil {
		method := "GET"
		if v := reFastAPIVerb.FindStringSubmatch(decoratorText); v != nil {
			method = strings.ToUpper(v[1])
		}
		endpoints = append(endpoints, domain.APIEndpoint{
			Method:      method,
			Path:        m[1],
			HandlerName: handlerName,
			Framework:   "fastapi",
			Line:        line,
		})
		return endpoints
	}

	// Flask: @app.route("/path", methods=["GET", "POST"])
	if m := reFlaskRoute.FindStringSubmatch(decoratorText); m != nil {
		methods := []string{"GET"}
		if mm := reHTTPMethod.FindStringSubmatch(decoratorText); mm != nil {
			methods = parseMethodList(mm[1])
		}
		for _, method := range methods {
			endpoints = append(endpoints, domain.APIEndpoint{
				Method:      method,
				Path:        m[1],
				HandlerName: handlerName,
				Framework:   "flask",
				Line:        line,
			})
		}
		return endpoints
	}

	return endpoints
}

// parseMethodList parses "GET", "POST" from a methods=[...] string.
func parseMethodList(raw string) []string {
	var methods []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			methods = append(methods, strings.ToUpper(part))
		}
	}
	return methods
}

// isUpperSnakeCase checks if a name is UPPER_SNAKE_CASE.
func isUpperSnakeCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, r := range name {
		if !unicode.IsUpper(r) && r != '_' && !unicode.IsDigit(r) {
			return false
		}
	}
	for _, r := range name {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// nodeContent extracts the text content of a tree-sitter node.
func nodeContent(node *tree_sitter.Node, source []byte) string {
	start := node.StartByte()
	end := node.EndByte()
	if start >= uint(len(source)) || end > uint(len(source)) {
		return ""
	}
	return string(source[start:end])
}
