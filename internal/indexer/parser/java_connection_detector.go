package parser

import (
	"fmt"
	"os"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// DetectJavaConnections detects cross-service connections in a Java source file.
func DetectJavaConnections(filePath string, cfg []JavaCallPattern) ([]domain.ServiceConnection, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}

	p := tree_sitter.NewParser()
	defer p.Close()

	lang := tree_sitter.NewLanguage(tree_sitter_java.Language())
	if err := p.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("set language: %w", err)
	}

	tree := p.Parse(source, nil)
	defer tree.Close()

	root := tree.RootNode()
	imports := buildJavaImportSet(root, source)

	var conns []domain.ServiceConnection
	walkJavaNodes(root, source, func(node *tree_sitter.Node) {
		for _, pattern := range cfg {
			var conn *domain.ServiceConnection
			if pattern.Annotation != "" {
				conn = matchJavaAnnotation(node, source, imports, pattern)
			} else if pattern.MethodCall != "" {
				conn = matchJavaMethodCall(node, source, imports, pattern)
			}
			if conn != nil {
				conns = append(conns, *conn)
				break
			}
		}
	})

	return conns, nil
}

// buildJavaImportSet returns a set of all imported paths.
func buildJavaImportSet(root *tree_sitter.Node, source []byte) map[string]struct{} {
	result := make(map[string]struct{})
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		if child.Kind() == "import_declaration" {
			content := nodeContent(child, source)
			content = strings.TrimPrefix(content, "import ")
			content = strings.TrimPrefix(content, "static ")
			content = strings.TrimSuffix(content, ";")
			content = strings.TrimSpace(content)
			result[content] = struct{}{}
		}
	}
	return result
}

// matchJavaAnnotation checks if a node is a matching annotation.
func matchJavaAnnotation(node *tree_sitter.Node, source []byte, imports map[string]struct{}, p JavaCallPattern) *domain.ServiceConnection {
	if !strings.Contains(node.Kind(), "annotation") {
		return nil
	}

	content := nodeContent(node, source)
	if !strings.Contains(content, p.Annotation) {
		return nil
	}

	if !hasImportContaining(imports, p.ImportContains) {
		return nil
	}

	target := extractAnnotationAttribute(node, source, p.TargetAttribute)
	pos := node.StartPosition()
	return &domain.ServiceConnection{
		ConnType: p.ConnType,
		Target:   target,
		Line:     int(pos.Row) + 1,
	}
}

// matchJavaMethodCall checks if a method_invocation node matches a pattern.
func matchJavaMethodCall(node *tree_sitter.Node, source []byte, imports map[string]struct{}, p JavaCallPattern) *domain.ServiceConnection {
	if node.Kind() != "method_invocation" {
		return nil
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil
	}

	if nodeContent(nameNode, source) != p.MethodCall {
		return nil
	}

	if !hasImportContaining(imports, p.ImportContains) {
		return nil
	}

	target := extractJavaArgString(node, source, p.TargetArgIndex)
	pos := node.StartPosition()
	return &domain.ServiceConnection{
		ConnType: p.ConnType,
		Target:   target,
		Line:     int(pos.Row) + 1,
	}
}

// extractAnnotationAttribute extracts a named attribute string value from an annotation.
// e.g. @KafkaListener(topics = "orders") with attrName="topics" → "orders"
func extractAnnotationAttribute(node *tree_sitter.Node, source []byte, attrName string) string {
	content := nodeContent(node, source)

	if attrName == "" {
		// Return the first string literal
		return extractAnnotationStringValue(content)
	}

	// Look for attrName = "value" or attrName = {"value"}
	searchKey := attrName + " = "
	if idx := strings.Index(content, searchKey); idx >= 0 {
		rest := content[idx+len(searchKey):]
		// Strip array notation
		rest = strings.TrimPrefix(rest, "{")
		if q := strings.IndexByte(rest, '"'); q >= 0 {
			rest = rest[q+1:]
			if end := strings.IndexByte(rest, '"'); end >= 0 {
				return rest[:end]
			}
		}
	}
	return ""
}

// extractJavaArgString returns the string value of positional argument at index.
func extractJavaArgString(node *tree_sitter.Node, source []byte, index int) string {
	argsNode := node.ChildByFieldName("arguments")
	if argsNode == nil {
		return ""
	}

	argIdx := 0
	for i := uint(0); i < argsNode.NamedChildCount(); i++ {
		child := argsNode.NamedChild(i)
		if child.Kind() == "string_literal" {
			if argIdx == index {
				content := nodeContent(child, source)
				content = strings.Trim(content, `"`)
				return content
			}
			argIdx++
		} else if child.Kind() != "," {
			argIdx++
		}
	}
	return ""
}

// hasImportContaining checks if any import path contains the given substring.
func hasImportContaining(imports map[string]struct{}, contains string) bool {
	if contains == "" {
		return true
	}
	for imp := range imports {
		if strings.Contains(imp, contains) {
			return true
		}
	}
	return false
}

// walkJavaNodes walks all nodes in the tree calling fn for each.
func walkJavaNodes(node *tree_sitter.Node, source []byte, fn func(*tree_sitter.Node)) {
	fn(node)
	for i := uint(0); i < node.NamedChildCount(); i++ {
		walkJavaNodes(node.NamedChild(i), source, fn)
	}
}
