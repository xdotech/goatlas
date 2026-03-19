package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// ListComponentsUseCase lists React components and hooks from the index.
type ListComponentsUseCase struct {
	symbolRepo domain.SymbolRepository
}

// NewListComponentsUseCase creates a new ListComponentsUseCase.
func NewListComponentsUseCase(sr domain.SymbolRepository) *ListComponentsUseCase {
	return &ListComponentsUseCase{symbolRepo: sr}
}

// Execute returns a formatted list of React components and hooks.
// kind: "" (all), "component", "hook", "interface", "type_alias"
func (uc *ListComponentsUseCase) Execute(ctx context.Context, kind string, limit int) (string, error) {
	if limit <= 0 {
		limit = 100
	}

	var kinds []string
	switch kind {
	case "component":
		kinds = []string{"component"}
	case "hook":
		kinds = []string{"hook"}
	case "interface":
		kinds = []string{"interface"}
	case "type_alias":
		kinds = []string{"type_alias"}
	default:
		kinds = []string{"component", "hook", "interface", "type_alias"}
	}

	symbols, err := uc.symbolRepo.ListByKinds(ctx, kinds, limit)
	if err != nil {
		return "", fmt.Errorf("list components: %w", err)
	}

	if len(symbols) == 0 {
		return "No React/TS symbols found. Run `goatlas index <path>` on a React/RN project first.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d React/TS symbol(s):\n\n", len(symbols)))

	// Group by kind
	byKind := map[string][]domain.Symbol{}
	for _, s := range symbols {
		byKind[s.Kind] = append(byKind[s.Kind], s)
	}

	order := []string{"component", "hook", "interface", "type_alias"}
	labels := map[string]string{
		"component":  "Components",
		"hook":       "Hooks",
		"interface":  "Interfaces",
		"type_alias": "Type Aliases",
	}

	for _, k := range order {
		syms, ok := byKind[k]
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s (%d)\n", labels[k], len(syms)))
		for _, s := range syms {
			sb.WriteString(fmt.Sprintf("  %-40s  line %-5d  %s\n", s.Name, s.Line, s.QualifiedName))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
