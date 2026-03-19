package parser

import (
	"fmt"
	"os"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_java "github.com/tree-sitter/tree-sitter-java/bindings/go"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// ParseJavaFile parses a Java source file using tree-sitter and extracts
// symbols (classes, methods), imports, and Spring MVC API endpoints.
func ParseJavaFile(filePath string) (*ParseResult, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", filePath, err)
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
	result := &ParseResult{}

	pkg := javaPackageName(root, source)
	result.Module = pkg

	// Walk top-level nodes
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		switch child.Kind() {
		case "import_declaration":
			result.Imports = append(result.Imports, extractJavaImport(child, source))
		case "class_declaration", "interface_declaration", "enum_declaration":
			syms, endpoints := extractJavaTypeDecl(child, source, pkg)
			result.Symbols = append(result.Symbols, syms...)
			result.Endpoints = append(result.Endpoints, endpoints...)
		}
	}

	return result, nil
}

// javaPackageName extracts the package name from the package declaration.
func javaPackageName(root *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		child := root.NamedChild(i)
		if child.Kind() == "package_declaration" {
			// package com.example.svc → last segment
			content := nodeContent(child, source)
			content = strings.TrimPrefix(content, "package ")
			content = strings.TrimSuffix(content, ";")
			content = strings.TrimSpace(content)
			parts := strings.Split(content, ".")
			return parts[len(parts)-1]
		}
	}
	return ""
}

// extractJavaImport extracts an import_declaration node.
func extractJavaImport(node *tree_sitter.Node, source []byte) domain.Import {
	content := nodeContent(node, source)
	content = strings.TrimPrefix(content, "import ")
	content = strings.TrimSuffix(content, ";")
	content = strings.TrimPrefix(content, "static ")
	content = strings.TrimSpace(content)
	return domain.Import{ImportPath: content}
}

// extractJavaTypeDecl extracts symbols and endpoints from a class/interface/enum.
func extractJavaTypeDecl(node *tree_sitter.Node, source []byte, pkg string) ([]domain.Symbol, []domain.APIEndpoint) {
	var symbols []domain.Symbol
	var endpoints []domain.APIEndpoint

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return nil, nil
	}
	className := nodeContent(nameNode, source)
	startPoint := node.StartPosition()

	kind := "class"
	switch node.Kind() {
	case "interface_declaration":
		kind = "interface"
	case "enum_declaration":
		kind = "enum"
	}

	symbols = append(symbols, domain.Symbol{
		Kind:          kind,
		Name:          className,
		QualifiedName: pkg + "." + className,
		Line:          int(startPoint.Row) + 1,
		Col:           int(startPoint.Column) + 1,
	})

	// Collect class-level annotations for endpoint detection
	classAnnotations := collectAnnotations(node, source)

	body := node.ChildByFieldName("body")
	if body == nil {
		return symbols, endpoints
	}

	for i := uint(0); i < body.NamedChildCount(); i++ {
		child := body.NamedChild(i)
		switch child.Kind() {
		case "method_declaration":
			sym := extractJavaMethod(child, source, pkg, className)
			symbols = append(symbols, sym)
			// Check for Spring method-level endpoint annotations
			eps := extractJavaEndpoints(child, source, sym.Name, classAnnotations)
			endpoints = append(endpoints, eps...)
		case "field_declaration":
			// Only index public static final fields
			if isPublicStaticFinal(child, source) {
				sym := extractJavaField(child, source, pkg, className)
				if sym.Name != "" {
					symbols = append(symbols, sym)
				}
			}
		}
	}

	return symbols, endpoints
}

// extractJavaMethod extracts a method_declaration node.
func extractJavaMethod(node *tree_sitter.Node, source []byte, pkg, parentClass string) domain.Symbol {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return domain.Symbol{}
	}
	name := nodeContent(nameNode, source)
	startPoint := node.StartPosition()

	return domain.Symbol{
		Kind:          "method",
		Name:          name,
		QualifiedName: pkg + ".(" + parentClass + ")." + name,
		Receiver:      parentClass,
		Line:          int(startPoint.Row) + 1,
		Col:           int(startPoint.Column) + 1,
	}
}

// extractJavaField extracts a public static final field.
func extractJavaField(node *tree_sitter.Node, source []byte, pkg, parentClass string) domain.Symbol {
	// Find the variable_declarator child
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.Kind() == "variable_declarator" {
			nameNode := child.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			name := nodeContent(nameNode, source)
			startPoint := node.StartPosition()
			return domain.Symbol{
				Kind:          "field",
				Name:          name,
				QualifiedName: pkg + "." + parentClass + "." + name,
				Line:          int(startPoint.Row) + 1,
				Col:           int(startPoint.Column) + 1,
			}
		}
	}
	return domain.Symbol{}
}

// isPublicStaticFinal checks if a field_declaration has public static final modifiers.
func isPublicStaticFinal(node *tree_sitter.Node, source []byte) bool {
	content := nodeContent(node, source)
	return strings.Contains(content, "public") &&
		strings.Contains(content, "static") &&
		strings.Contains(content, "final")
}

// collectAnnotations returns annotation texts on a node.
// Handles annotations both as direct children and inside a modifiers node.
func collectAnnotations(node *tree_sitter.Node, source []byte) []string {
	var annotations []string
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if strings.Contains(child.Kind(), "annotation") {
			annotations = append(annotations, nodeContent(child, source))
		} else if child.Kind() == "modifiers" {
			// annotations are children of the modifiers node
			for j := uint(0); j < child.NamedChildCount(); j++ {
				mod := child.NamedChild(j)
				if strings.Contains(mod.Kind(), "annotation") {
					annotations = append(annotations, nodeContent(mod, source))
				}
			}
		}
	}
	return annotations
}

// extractJavaEndpoints detects Spring MVC annotations on methods.
func extractJavaEndpoints(node *tree_sitter.Node, source []byte, handlerName string, classAnnotations []string) []domain.APIEndpoint {
	var endpoints []domain.APIEndpoint

	annotations := collectAnnotations(node, source)
	allAnnotations := append(classAnnotations, annotations...)

	springMappings := map[string]string{
		"GetMapping":    "GET",
		"PostMapping":   "POST",
		"PutMapping":    "PUT",
		"DeleteMapping": "DELETE",
		"PatchMapping":  "PATCH",
		"RequestMapping": "GET",
	}

	startPoint := node.StartPosition()
	for _, ann := range allAnnotations {
		for annName, method := range springMappings {
			if !strings.Contains(ann, annName) {
				continue
			}
			path := extractAnnotationStringValue(ann)
			endpoints = append(endpoints, domain.APIEndpoint{
				Method:      method,
				Path:        path,
				HandlerName: handlerName,
				Framework:   "spring_mvc",
				Line:        int(startPoint.Row) + 1,
			})
			break
		}
	}
	return endpoints
}

// extractAnnotationStringValue extracts the string value from an annotation like @GetMapping("/path").
func extractAnnotationStringValue(annotation string) string {
	// Look for "value" or the first string literal
	if idx := strings.Index(annotation, `"`); idx >= 0 {
		end := strings.Index(annotation[idx+1:], `"`)
		if end >= 0 {
			return annotation[idx+1 : idx+1+end]
		}
	}
	return ""
}
