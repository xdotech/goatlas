package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// ParseResult holds the symbols, imports, and endpoints extracted from a source file.
type ParseResult struct {
	Symbols   []domain.Symbol
	Imports   []domain.Import
	Endpoints []domain.APIEndpoint
	Module    string
}

// ParseFile parses a Go source file and extracts symbols and imports.
func ParseFile(filePath string) (*ParseResult, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	result := &ParseResult{}
	pkgName := f.Name.Name

	// Extract imports
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		result.Imports = append(result.Imports, domain.Import{
			ImportPath: importPath,
			Alias:      alias,
		})
	}

	// Extract symbols via AST walk
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			sym := extractFunc(node, fset, pkgName)
			result.Symbols = append(result.Symbols, sym)

		case *ast.GenDecl:
			syms := extractGenDecl(node, fset, pkgName)
			result.Symbols = append(result.Symbols, syms...)
		}
		return true
	})

	result.Module = pkgName
	return result, nil
}

// ModuleFromGoMod extracts the Go module path from go.mod in the repo root.
func ModuleFromGoMod(repoPath string) string {
	goModPath := filepath.Join(repoPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	return ""
}

func extractFunc(fn *ast.FuncDecl, fset *token.FileSet, pkgName string) domain.Symbol {
	pos := fset.Position(fn.Pos())
	sym := domain.Symbol{
		Kind: "func",
		Name: fn.Name.Name,
		Line: pos.Line,
		Col:  pos.Column,
	}

	if fn.Doc != nil {
		sym.DocComment = strings.TrimSpace(fn.Doc.Text())
	}

	receiver := ""
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		sym.Kind = "method"
		receiver = typeToString(fn.Recv.List[0].Type)
		sym.Receiver = receiver
	}

	sym.QualifiedName = buildQualifiedName(pkgName, receiver, fn.Name.Name)
	sym.Signature = buildFuncSignature(fn)
	return sym
}

func extractGenDecl(decl *ast.GenDecl, fset *token.FileSet, pkgName string) []domain.Symbol {
	var syms []domain.Symbol
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			pos := fset.Position(s.Pos())
			kind := "type"
			switch s.Type.(type) {
			case *ast.InterfaceType:
				kind = "interface"
			case *ast.StructType:
				kind = "struct"
			}
			doc := ""
			if decl.Doc != nil {
				doc = strings.TrimSpace(decl.Doc.Text())
			}
			syms = append(syms, domain.Symbol{
				Kind:          kind,
				Name:          s.Name.Name,
				QualifiedName: pkgName + "." + s.Name.Name,
				Line:          pos.Line,
				Col:           pos.Column,
				DocComment:    doc,
			})
		case *ast.ValueSpec:
			pos := fset.Position(s.Pos())
			kind := "var"
			if decl.Tok == token.CONST {
				kind = "const"
			}
			for _, name := range s.Names {
				syms = append(syms, domain.Symbol{
					Kind:          kind,
					Name:          name.Name,
					QualifiedName: pkgName + "." + name.Name,
					Line:          pos.Line,
					Col:           pos.Column,
				})
			}
		}
	}
	return syms
}

func buildQualifiedName(pkg, receiver, name string) string {
	if receiver != "" {
		return fmt.Sprintf("%s.(%s).%s", pkg, receiver, name)
	}
	return pkg + "." + name
}

func buildFuncSignature(fn *ast.FuncDecl) string {
	var sb strings.Builder
	sb.WriteString("func ")
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		sb.WriteString("(")
		sb.WriteString(typeToString(fn.Recv.List[0].Type))
		sb.WriteString(") ")
	}
	sb.WriteString(fn.Name.Name)
	sb.WriteString("(")
	if fn.Type.Params != nil {
		params := []string{}
		for _, p := range fn.Type.Params.List {
			typeStr := typeToString(p.Type)
			if len(p.Names) == 0 {
				params = append(params, typeStr)
			}
			for _, n := range p.Names {
				params = append(params, n.Name+" "+typeStr)
			}
		}
		sb.WriteString(strings.Join(params, ", "))
	}
	sb.WriteString(")")
	if fn.Type.Results != nil {
		results := []string{}
		for _, r := range fn.Type.Results.List {
			results = append(results, typeToString(r.Type))
		}
		if len(results) == 1 {
			sb.WriteString(" " + results[0])
		} else if len(results) > 1 {
			sb.WriteString(" (" + strings.Join(results, ", ") + ")")
		}
	}
	return sb.String()
}

func typeToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + typeToString(t.Elt)
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + typeToString(t.Value)
	default:
		return fmt.Sprintf("%T", expr)
	}
}
