package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// ExtractInterfaceImpls parses a Go file and detects interface implementation
// relationships by matching struct methods against interface method sets defined
// in the same file.
//
// Approach: within a single file, collect all interface definitions and all
// methods on struct receivers. For each interface method, if a struct has a
// method with the same name, we record it as an implementation.
// This is a heuristic (same-file, name-based) that works well for typical
// Go service code where interfaces and implementations coexist.
func ExtractInterfaceImpls(filePath string) ([]domain.InterfaceImpl, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	pkgName := f.Name.Name

	// Phase 1: Collect all interface definitions and their method names
	type ifaceDef struct {
		name    string
		methods []string
	}
	var interfaces []ifaceDef

	// Phase 2: Collect all struct methods (receiver → method names)
	type methodDef struct {
		receiver string
		name     string
	}
	var structMethods []methodDef

	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				iface, ok := ts.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				var methods []string
				if iface.Methods != nil {
					for _, m := range iface.Methods.List {
						for _, name := range m.Names {
							methods = append(methods, name.Name)
						}
					}
				}
				if len(methods) > 0 {
					interfaces = append(interfaces, ifaceDef{
						name:    ts.Name.Name,
						methods: methods,
					})
				}
			}

		case *ast.FuncDecl:
			if node.Recv != nil && len(node.Recv.List) > 0 {
				recvType := typeToString(node.Recv.List[0].Type)
				structMethods = append(structMethods, methodDef{
					receiver: recvType,
					name:     node.Name.Name,
				})
			}
		}
		return true
	})

	// Phase 3: Match — for each interface method, find structs that have a method
	// with the same name
	methodsByReceiver := make(map[string]map[string]bool)
	for _, sm := range structMethods {
		if methodsByReceiver[sm.receiver] == nil {
			methodsByReceiver[sm.receiver] = make(map[string]bool)
		}
		methodsByReceiver[sm.receiver][sm.name] = true
	}

	var impls []domain.InterfaceImpl
	for _, iface := range interfaces {
		for recv, methods := range methodsByReceiver {
			// Check if this struct implements all methods of the interface
			matchCount := 0
			for _, im := range iface.methods {
				if methods[im] {
					matchCount++
				}
			}
			// Determine confidence based on match completeness
			var confidence float64
			if matchCount == len(iface.methods) {
				confidence = 0.85 // same-file, all methods matched
			} else {
				continue // skip partial matches in same-file mode
			}
			for _, im := range iface.methods {
				impls = append(impls, domain.InterfaceImpl{
					InterfaceName: pkgName + "." + iface.name,
					StructName:    pkgName + ".(" + recv + ")",
					MethodName:    im,
					Confidence:    confidence,
				})
			}
		}
	}

	return impls, nil
}
