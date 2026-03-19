package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// ConnectionResult holds cross-service connections detected in a Go source file.
type ConnectionResult struct {
	Connections []domain.ServiceConnection
}

// DetectConnections parses a Go file and detects cross-service connections
// using patterns from the config. If cfg is nil, returns empty result.
func DetectConnections(filePath string, cfg *PatternConfig) (*ConnectionResult, error) {
	if cfg == nil {
		return &ConnectionResult{}, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	result := &ConnectionResult{}

	// Collect import aliases for detecting which packages are imported
	importAliases := map[string]string{} // alias/name -> import path
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if imp.Name != nil {
			importAliases[imp.Name.Name] = importPath
		} else {
			parts := strings.Split(importPath, "/")
			importAliases[parts[len(parts)-1]] = importPath
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check gRPC patterns
		for _, p := range cfg.Go.GRPC {
			if conn := matchGoCall(call, fset, importAliases, p); conn != nil {
				result.Connections = append(result.Connections, *conn)
				break
			}
		}

		// Check Kafka consumer patterns
		for _, p := range cfg.Go.KafkaConsumer {
			if conn := matchGoCall(call, fset, importAliases, p); conn != nil {
				result.Connections = append(result.Connections, *conn)
				break
			}
		}

		// Check Kafka producer patterns
		for _, p := range cfg.Go.KafkaProducer {
			if conn := matchGoCall(call, fset, importAliases, p); conn != nil {
				result.Connections = append(result.Connections, *conn)
				break
			}
		}

		// Check HTTP client patterns
		for _, p := range cfg.Go.HTTPClient {
			if conn := matchGoCall(call, fset, importAliases, p); conn != nil {
				result.Connections = append(result.Connections, *conn)
				break
			}
		}

		return true
	})

	return result, nil
}

// matchGoCall checks if a function call matches a GoCallPattern from the config.
func matchGoCall(call *ast.CallExpr, fset *token.FileSet, imports map[string]string, pattern GoCallPattern) *domain.ServiceConnection {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	// Check import path matches
	importPath, hasImport := imports[ident.Name]
	if !hasImport {
		return nil
	}

	if pattern.PackageSuffix != "" && !strings.HasSuffix(importPath, pattern.PackageSuffix) {
		return nil
	}
	if pattern.PackageContains != "" && !strings.Contains(importPath, pattern.PackageContains) {
		return nil
	}

	// Check function name matches
	fnName := sel.Sel.Name
	matched := false
	if pattern.Function != "" && fnName == pattern.Function {
		matched = true
	}
	for _, fn := range pattern.Functions {
		if fnName == fn {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}

	// Extract target from specified argument
	target := pattern.ConnType
	if pattern.TargetArg < len(call.Args) {
		target = connExprToString(call.Args[pattern.TargetArg])
	}

	pos := fset.Position(call.Pos())
	return &domain.ServiceConnection{
		ConnType: pattern.ConnType,
		Target:   target,
		Line:     pos.Line,
	}
}

// connExprToString converts an AST expression to a readable string.
func connExprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return connExprToString(e.X) + "." + e.Sel.Name
	case *ast.CallExpr:
		return connExprToString(e.Fun) + "(...)"
	case *ast.UnaryExpr:
		return connExprToString(e.X)
	case *ast.CompositeLit:
		return typeToString(e.Type) + "{}"
	default:
		return fmt.Sprintf("%T", expr)
	}
}
