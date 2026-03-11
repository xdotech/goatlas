package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// ExtractRoutes detects HTTP routes based on framework imports present in the file.
func ExtractRoutes(filePath string, imports []domain.Import) ([]domain.APIEndpoint, error) {
	framework := detectFramework(imports)
	if framework == "" {
		return nil, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	var endpoints []domain.APIEndpoint
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		method := strings.ToUpper(sel.Sel.Name)
		httpMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true,
			"PATCH": true, "HEAD": true, "OPTIONS": true,
		}

		if !httpMethods[method] {
			// also check HandleFunc / Handle for net/http
			if sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle" {
				return true
			}
			method = "ANY"
		}

		pos := fset.Position(call.Pos())

		if len(call.Args) >= 1 {
			path := extractStringArg(call.Args[0])
			handlerName := ""
			if len(call.Args) >= 2 {
				handlerName = exprToString(call.Args[1])
			}
			endpoints = append(endpoints, domain.APIEndpoint{
				Method:      method,
				Path:        path,
				HandlerName: handlerName,
				Framework:   framework,
				Line:        pos.Line,
			})
		}

		return true
	})

	return endpoints, nil
}

func detectFramework(imports []domain.Import) string {
	for _, imp := range imports {
		switch {
		case strings.Contains(imp.ImportPath, "gin-gonic/gin"):
			return "gin"
		case strings.Contains(imp.ImportPath, "labstack/echo"):
			return "echo"
		case strings.Contains(imp.ImportPath, "go-chi/chi"):
			return "chi"
		case imp.ImportPath == "net/http":
			return "net_http"
		}
	}
	return ""
}

func extractStringArg(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return ""
	}
	return strings.Trim(lit.Value, `"`)
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	default:
		return ""
	}
}
