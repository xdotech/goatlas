package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// ExtractTypeUsages parses a Go file and extracts type usage information
// from function/method signatures — which types are accepted as parameters
// and which types are returned.
func ExtractTypeUsages(filePath string) ([]domain.TypeUsage, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	pkgName := f.Name.Name
	var usages []domain.TypeUsage

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		pos := fset.Position(fn.Pos())

		// Build caller qualified name
		receiver := ""
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			receiver = typeToString(fn.Recv.List[0].Type)
		}
		qualifiedName := buildQualifiedName(pkgName, receiver, fn.Name.Name)

		// Extract input types from parameters
		if fn.Type.Params != nil {
			paramPos := 0
			for _, field := range fn.Type.Params.List {
				typeName := extractNamedType(field.Type)
				if typeName == "" || isBuiltinType(typeName) {
					paramPos++
					continue
				}

				count := len(field.Names)
				if count == 0 {
					count = 1
				}
				for i := 0; i < count; i++ {
					usages = append(usages, domain.TypeUsage{
						SymbolName: qualifiedName,
						TypeName:   typeName,
						Direction:  "input",
						Position:   paramPos,
						Line:       pos.Line,
					})
					paramPos++
				}
			}
		}

		// Extract output types from return values
		if fn.Type.Results != nil {
			for retPos, field := range fn.Type.Results.List {
				typeName := extractNamedType(field.Type)
				if typeName == "" || isBuiltinType(typeName) {
					continue
				}

				usages = append(usages, domain.TypeUsage{
					SymbolName: qualifiedName,
					TypeName:   typeName,
					Direction:  "output",
					Position:   retPos,
					Line:       pos.Line,
				})
			}
		}

		return true
	})

	return usages, nil
}

// extractNamedType extracts the base named type from an AST expression,
// stripping pointers, slices, maps, etc. Returns "" for anonymous/builtin types.
func extractNamedType(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractNamedType(t.X)
	case *ast.SelectorExpr:
		// pkg.Type → return "pkg.Type"
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name + "." + t.Sel.Name
		}
		return t.Sel.Name
	case *ast.ArrayType:
		return extractNamedType(t.Elt)
	case *ast.MapType:
		// Use value type for maps
		return extractNamedType(t.Value)
	default:
		return ""
	}
}

// isBuiltinType returns true if the type name is a Go builtin that we don't
// want to track in type flow analysis.
func isBuiltinType(name string) bool {
	switch name {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "error", "any",
		"interface{}", "struct{}":
		return true
	}
	// Also skip context.Context
	if strings.HasSuffix(name, "Context") {
		return true
	}
	return false
}
