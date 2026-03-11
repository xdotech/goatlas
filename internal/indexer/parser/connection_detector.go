package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// ConnectionResult holds cross-service connections detected in a Go source file.
type ConnectionResult struct {
	Connections []domain.ServiceConnection
}

// DetectConnections parses a Go file and detects cross-service connections:
// - gRPC clients via grpcx.NewClientManager calls
// - Kafka consumers via kafka.MustNewListener / kafka.NewQueue calls
// - Kafka producers via queue.Must / queue.New calls
func DetectConnections(filePath string) (*ConnectionResult, error) {
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
			// Use last segment of import path as default alias
			parts := strings.Split(importPath, "/")
			importAliases[parts[len(parts)-1]] = importPath
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Detect gRPC: grpcx.NewClientManager(conf, pb.NewXXXClient)
		if conn := detectGRPCClient(call, fset, importAliases); conn != nil {
			result.Connections = append(result.Connections, *conn)
		}

		// Detect Kafka consumer: kafka.MustNewListener(cfg, handler) or kafka.NewQueue(cfg, handler)
		if conn := detectKafkaConsumer(call, fset, importAliases); conn != nil {
			result.Connections = append(result.Connections, *conn)
		}

		// Detect Kafka producer: queue.Must(cfg) or queue.New(cfg)
		if conn := detectKafkaProducer(call, fset, importAliases); conn != nil {
			result.Connections = append(result.Connections, *conn)
		}

		return true
	})

	return result, nil
}

// detectGRPCClient detects grpcx.NewClientManager(conf, pb.NewXXXClient) pattern.
// Extracts the second argument (builder function) to identify the target service.
func detectGRPCClient(call *ast.CallExpr, fset *token.FileSet, imports map[string]string) *domain.ServiceConnection {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	// Check for grpcx.NewClientManager
	importPath, hasImport := imports[ident.Name]
	if !hasImport || !strings.HasSuffix(importPath, "grpcx") {
		return nil
	}
	if sel.Sel.Name != "NewClientManager" {
		return nil
	}

	// Extract builder function from second argument (e.g., transferorderpb.NewTransferOrderClient)
	target := "unknown"
	if len(call.Args) >= 2 {
		target = connExprToString(call.Args[1])
	}

	pos := fset.Position(call.Pos())
	return &domain.ServiceConnection{
		ConnType: "grpc",
		Target:   target,
		Line:     pos.Line,
	}
}

// detectKafkaConsumer detects kafka.MustNewListener or kafka.NewQueue patterns.
func detectKafkaConsumer(call *ast.CallExpr, fset *token.FileSet, imports map[string]string) *domain.ServiceConnection {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	importPath, hasImport := imports[ident.Name]
	if !hasImport {
		return nil
	}

	// Check for kafka.MustNewListener or kafka.NewQueue
	isKafkaConsumer := strings.Contains(importPath, "queue/kafka") &&
		(sel.Sel.Name == "MustNewListener" || sel.Sel.Name == "NewQueue")

	if !isKafkaConsumer {
		return nil
	}

	// Extract topic info from first argument if possible
	target := "kafka_consumer"
	if len(call.Args) >= 1 {
		target = "kafka_consumer:" + connExprToString(call.Args[0])
	}

	pos := fset.Position(call.Pos())
	return &domain.ServiceConnection{
		ConnType: "kafka_consume",
		Target:   target,
		Line:     pos.Line,
	}
}

// detectKafkaProducer detects queue.Must or queue.New patterns.
func detectKafkaProducer(call *ast.CallExpr, fset *token.FileSet, imports map[string]string) *domain.ServiceConnection {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return nil
	}

	importPath, hasImport := imports[ident.Name]
	if !hasImport {
		return nil
	}

	// Check for queue.Must or queue.New (the producer factory functions)
	isProducer := strings.HasSuffix(importPath, "/queue") &&
		(sel.Sel.Name == "Must" || sel.Sel.Name == "New")

	if !isProducer {
		return nil
	}

	target := "kafka_producer"
	if len(call.Args) >= 1 {
		target = "kafka_producer:" + connExprToString(call.Args[0])
	}

	pos := fset.Position(call.Pos())
	return &domain.ServiceConnection{
		ConnType: "kafka_publish",
		Target:   target,
		Line:     pos.Line,
	}
}

// connExprToString converts an AST expression to a readable string, used to extract
// function names and identifiers from call arguments.
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
