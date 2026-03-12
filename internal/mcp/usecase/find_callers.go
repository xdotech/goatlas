package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// FindCallersUseCase finds symbols that reference a given function name.
// It also resolves interface methods to their concrete implementations.
type FindCallersUseCase struct {
	symbolRepo domain.SymbolRepository
	iiRepo     domain.InterfaceImplRepository
}

// NewFindCallersUseCase constructs a FindCallersUseCase.
func NewFindCallersUseCase(sr domain.SymbolRepository, iiRepo domain.InterfaceImplRepository) *FindCallersUseCase {
	return &FindCallersUseCase{symbolRepo: sr, iiRepo: iiRepo}
}

// Execute searches for references/callers of the given function name.
// If the function name contains a dot (e.g. "Repository.FindByID"), it will
// also resolve interface implementations and search for callers of the concrete methods.
func (uc *FindCallersUseCase) Execute(ctx context.Context, functionName string) (string, error) {
	// Resolve interface methods → concrete implementations
	searchNames := []string{functionName}

	if uc.iiRepo != nil {
		interfaceName, methodName := splitInterfaceMethod(functionName)
		if methodName != "" {
			impls, err := uc.iiRepo.FindImplementations(ctx, interfaceName, methodName)
			if err == nil && len(impls) > 0 {
				for _, impl := range impls {
					// Add concrete method qualified names: pkg.(Struct).Method
					searchNames = append(searchNames, impl.StructName+"."+impl.MethodName)
				}
			}
		}
	}

	var allSymbols []domain.Symbol
	seen := make(map[string]bool)

	for _, name := range searchNames {
		symbols, err := uc.symbolRepo.Search(ctx, name, 30, "")
		if err != nil {
			return "", err
		}
		for _, s := range symbols {
			key := fmt.Sprintf("%s:%d", s.QualifiedName, s.Line)
			if !seen[key] {
				seen[key] = true
				allSymbols = append(allSymbols, s)
			}
		}
	}

	if len(allSymbols) == 0 {
		return fmt.Sprintf("No callers found for %q", functionName), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Potential callers/references for %q", functionName))
	if len(searchNames) > 1 {
		sb.WriteString(fmt.Sprintf(" (+ %d concrete implementations)", len(searchNames)-1))
	}
	sb.WriteString(":\n\n")
	for _, s := range allSymbols {
		sb.WriteString(fmt.Sprintf("  [%s] %s (line %d)\n", s.Kind, s.QualifiedName, s.Line))
	}
	return sb.String(), nil
}

// splitInterfaceMethod splits "InterfaceName.MethodName" or just "MethodName".
func splitInterfaceMethod(name string) (interfaceName, methodName string) {
	// Handle qualified names like "pkg.InterfaceName.Method"
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "", name
}
