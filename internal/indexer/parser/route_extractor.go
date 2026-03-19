package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
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

	// Go-zero uses composite literals instead of method calls
	if framework == "go_zero" {
		return extractGoZeroRoutes(f, fset, framework), nil
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

// extractGoZeroRoutes parses go-zero rest.Route composite literals.
// Handles both string literal methods ("GET") and http.MethodXxx selector expressions.
// Go-zero routes look like: []rest.Route{{Method: "GET", Path: "/..."}, {...}}
// The inner `{...}` composite literals have nil Type — only the outer array has the type.
func extractGoZeroRoutes(f *ast.File, fset *token.FileSet, framework string) []domain.APIEndpoint {
	var endpoints []domain.APIEndpoint

	// Map http.MethodXxx constants to uppercase method strings
	httpMethodConstants := map[string]string{
		"MethodGet":     "GET",
		"MethodPost":    "POST",
		"MethodPut":     "PUT",
		"MethodDelete":  "DELETE",
		"MethodPatch":   "PATCH",
		"MethodHead":    "HEAD",
		"MethodOptions": "OPTIONS",
	}

	ast.Inspect(f, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Look for []rest.Route{...} array composite literals
		arrType, ok := comp.Type.(*ast.ArrayType)
		if !ok {
			return true
		}
		if !isRestRouteSelector(arrType.Elt) {
			return true
		}

		// Each element is an untyped {Method: ..., Path: ..., Handler: ...} composite literal
		for _, elt := range comp.Elts {
			inner, ok := elt.(*ast.CompositeLit)
			if !ok {
				continue
			}

			ep := domain.APIEndpoint{Framework: framework}
			pos := fset.Position(inner.Pos())
			ep.Line = pos.Line

			for _, field := range inner.Elts {
				kv, ok := field.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}

				switch key.Name {
				case "Method":
					// String literal: "GET"
					if lit, ok := kv.Value.(*ast.BasicLit); ok {
						ep.Method = strings.ToUpper(strings.Trim(lit.Value, `"`))
					}
					// Selector: http.MethodGet
					if sel, ok := kv.Value.(*ast.SelectorExpr); ok {
						if m, found := httpMethodConstants[sel.Sel.Name]; found {
							ep.Method = m
						}
					}
				case "Path":
					if lit, ok := kv.Value.(*ast.BasicLit); ok {
						ep.Path = strings.Trim(lit.Value, `"`)
					}
				case "Handler":
					ep.HandlerName = exprToString(kv.Value)
				}
			}

			if ep.Method != "" && ep.Path != "" {
				endpoints = append(endpoints, ep)
			}
		}

		// Don't descend into children — we already processed elements
		return false
	})

	return endpoints
}

// isRestRouteSelector checks if an expression is `rest.Route`.
func isRestRouteSelector(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "rest" && sel.Sel.Name == "Route"
}

func detectFramework(imports []domain.Import) string {
	for _, imp := range imports {
		switch {
		case strings.Contains(imp.ImportPath, "zeromicro/go-zero/rest"):
			return "go_zero"
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
	case *ast.CallExpr:
		return exprToString(e.Fun)
	default:
		return ""
	}
}

