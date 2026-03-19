package parser

import (
	"fmt"
	"os"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// DetectPythonConnections uses tree-sitter AST to detect cross-service connections.
func DetectPythonConnections(filePath string, cfg []PyCallPattern) ([]domain.ServiceConnection, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	p := tree_sitter.NewParser()
	defer p.Close()

	lang := tree_sitter.NewLanguage(tree_sitter_python.Language())
	if err := p.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("set language: %w", err)
	}

	tree := p.Parse(source, nil)
	defer tree.Close()

	root := tree.RootNode()
	imports := buildPythonImportMap(root, source)

	var conns []domain.ServiceConnection
	walkPythonNodes(root, source, func(node *tree_sitter.Node) {
		if node.Kind() != "call" {
			return
		}
		for _, pattern := range cfg {
			if conn := matchPyCallExpr(node, source, imports, pattern); conn != nil {
				conns = append(conns, *conn)
				break
			}
		}
	})

	return conns, nil
}

// buildPythonImportMap scans AST for imports, returns local name → module path.
func buildPythonImportMap(root *tree_sitter.Node, source []byte) map[string]string {
	result := make(map[string]string)
	for i := uint(0); i < root.NamedChildCount(); i++ {
		node := root.NamedChild(i)
		switch node.Kind() {
		case "import_statement":
			// import foo, import foo.bar, import foo as bar
			for j := uint(0); j < node.NamedChildCount(); j++ {
				child := node.NamedChild(j)
				switch child.Kind() {
				case "dotted_name":
					name := nodeContent(child, source)
					parts := strings.Split(name, ".")
					result[parts[0]] = name
				case "aliased_import":
					nameNode := child.ChildByFieldName("name")
					aliasNode := child.ChildByFieldName("alias")
					if nameNode != nil {
						modPath := nodeContent(nameNode, source)
						localName := modPath
						if aliasNode != nil {
							localName = nodeContent(aliasNode, source)
						} else {
							parts := strings.Split(modPath, ".")
							localName = parts[0]
						}
						result[localName] = modPath
					}
				}
			}

		case "import_from_statement":
			// from foo import Bar, Baz
			modNode := node.ChildByFieldName("module_name")
			if modNode == nil {
				continue
			}
			modPath := nodeContent(modNode, source)
			for j := uint(0); j < node.NamedChildCount(); j++ {
				child := node.NamedChild(j)
				switch child.Kind() {
				case "dotted_name":
					if child == modNode {
						continue
					}
					name := nodeContent(child, source)
					result[name] = modPath
				case "aliased_import":
					nameNode := child.ChildByFieldName("name")
					aliasNode := child.ChildByFieldName("alias")
					if nameNode != nil {
						localName := nodeContent(nameNode, source)
						if aliasNode != nil {
							localName = nodeContent(aliasNode, source)
						}
						result[localName] = modPath
					}
				}
			}
		}
	}
	return result
}

// matchPyCallExpr checks if a call node matches a PyCallPattern.
func matchPyCallExpr(node *tree_sitter.Node, source []byte, imports map[string]string, pattern PyCallPattern) *domain.ServiceConnection {
	// Get callee text
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return nil
	}
	calleeText := nodeContent(funcNode, source)

	// Check if pattern.CallPattern is present in callee
	// We match against the call expression text including opening paren
	fullCallText := calleeText + "("
	if !strings.Contains(fullCallText, strings.TrimSuffix(pattern.CallPattern, "(")+"(") {
		return nil
	}

	// Check import match: any imported name whose module contains pattern.ModuleContains
	importMatched := false
	for localName, modPath := range imports {
		if strings.Contains(modPath, pattern.ModuleContains) {
			// Check callee starts with localName or is localName
			if strings.HasPrefix(calleeText, localName) {
				importMatched = true
				break
			}
		}
	}
	if !importMatched {
		return nil
	}

	// Get argument list
	argListNode := node.ChildByFieldName("arguments")
	target := extractPyStringArg(argListNode, source, pattern.TargetArgIndex, pattern.TargetKeyword)

	pos := node.StartPosition()
	return &domain.ServiceConnection{
		ConnType: pattern.ConnType,
		Target:   target,
		Line:     int(pos.Row) + 1,
	}
}

// extractPyStringArg returns string value of positional arg at index or named keyword arg.
func extractPyStringArg(argList *tree_sitter.Node, source []byte, index int, keyword string) string {
	if argList == nil {
		return ""
	}

	posIdx := 0
	for i := uint(0); i < argList.NamedChildCount(); i++ {
		child := argList.NamedChild(i)
		switch child.Kind() {
		case "keyword_argument":
			if keyword != "" {
				nameNode := child.ChildByFieldName("name")
				valNode := child.ChildByFieldName("value")
				if nameNode != nil && valNode != nil {
					if nodeContent(nameNode, source) == keyword {
						return extractStringLiteral(valNode, source)
					}
				}
			}
		case "string":
			if posIdx == index {
				return extractStringLiteral(child, source)
			}
			posIdx++
		default:
			if child.Kind() != "comment" {
				posIdx++
			}
		}
	}
	return ""
}

// extractStringLiteral returns the unquoted content of a string node.
func extractStringLiteral(node *tree_sitter.Node, source []byte) string {
	raw := nodeContent(node, source)
	for _, quote := range []string{`"""`, `'''`, `"`, `'`} {
		if strings.HasPrefix(raw, quote) && strings.HasSuffix(raw, quote) && len(raw) >= 2*len(quote) {
			return raw[len(quote) : len(raw)-len(quote)]
		}
	}
	return raw
}

// walkPythonNodes walks all nodes in the tree calling fn for each.
func walkPythonNodes(node *tree_sitter.Node, source []byte, fn func(*tree_sitter.Node)) {
	fn(node)
	for i := uint(0); i < node.NamedChildCount(); i++ {
		walkPythonNodes(node.NamedChild(i), source, fn)
	}
}
