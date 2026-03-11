package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// ExtractFunctionCalls parses a Go file and extracts all function/method calls
// made within each function body, building a caller→callee call graph.
func ExtractFunctionCalls(filePath string) ([]domain.FunctionCall, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	pkgName := f.Name.Name

	// Build import alias map: alias/pkgName → import path
	importAliases := make(map[string]string)
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			importAliases[imp.Name.Name] = path
		} else {
			// Use last segment as default alias
			parts := strings.Split(path, "/")
			importAliases[parts[len(parts)-1]] = path
		}
	}

	var calls []domain.FunctionCall

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Determine the caller's qualified name
		receiver := ""
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			receiver = typeToString(fn.Recv.List[0].Type)
		}
		callerQName := buildQualifiedName(pkgName, receiver, fn.Name.Name)

		if fn.Body == nil {
			return true
		}

		// Walk the function body for call expressions
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			call, ok := inner.(*ast.CallExpr)
			if !ok {
				return true
			}

			pos := fset.Position(call.Pos())

			switch fun := call.Fun.(type) {
			case *ast.Ident:
				// Direct function call: funcName(...)
				calls = append(calls, domain.FunctionCall{
					CallerQualifiedName: callerQName,
					CalleeName:          fun.Name,
					CalleePackage:       pkgName,
					Line:                pos.Line,
					Col:                 pos.Column,
				})

			case *ast.SelectorExpr:
				// Method or package call: pkg.Func() or obj.Method()
				calleeName := fun.Sel.Name
				calleePkg := ""

				if ident, ok := fun.X.(*ast.Ident); ok {
					// Could be: pkg.Func() or variable.Method()
					if _, isImport := importAliases[ident.Name]; isImport {
						calleePkg = ident.Name
					} else {
						// It's a method call on a variable — use the variable name as context
						calleePkg = ident.Name
					}
				}

				calls = append(calls, domain.FunctionCall{
					CallerQualifiedName: callerQName,
					CalleeName:          calleeName,
					CalleePackage:       calleePkg,
					Line:                pos.Line,
					Col:                 pos.Column,
				})
			}

			return true
		})

		return true
	})

	return calls, nil
}
