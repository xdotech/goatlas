package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// FindSymbolUseCase finds a symbol by name and optional kind filter.
type FindSymbolUseCase struct {
	symbolRepo domain.SymbolRepository
	repoRoot   string
}

// NewFindSymbolUseCase constructs a FindSymbolUseCase.
func NewFindSymbolUseCase(sr domain.SymbolRepository, repoRoot string) *FindSymbolUseCase {
	return &FindSymbolUseCase{symbolRepo: sr, repoRoot: repoRoot}
}

// Execute searches for a symbol by name and returns formatted details with source snippets.
func (uc *FindSymbolUseCase) Execute(ctx context.Context, name, kind string) (string, error) {
	symbols, err := uc.symbolRepo.SearchWithFile(ctx, name, 10, kind)
	if err != nil {
		return "", err
	}
	if len(symbols) == 0 {
		return fmt.Sprintf("Symbol %q not found", name), nil
	}
	var sb strings.Builder
	for _, s := range symbols {
		sb.WriteString(fmt.Sprintf("[%s] %s\n", s.Kind, s.QualifiedName))
		fmt.Fprintf(&sb, "  file: %s:%d\n", s.FilePath, s.Line)
		if s.Signature != "" {
			sb.WriteString(fmt.Sprintf("  signature: %s\n", s.Signature))
		}
		if s.Receiver != "" {
			sb.WriteString(fmt.Sprintf("  receiver: %s\n", s.Receiver))
		}
		if s.DocComment != "" {
			sb.WriteString(fmt.Sprintf("  doc: %s\n", s.DocComment))
		}
		// Add source snippet
		if snippet := ExtractSnippet(uc.repoRoot, s.FilePath, s.Line, defaultSnippetLines); snippet != "" {
			sb.WriteString("  ───────\n")
			sb.WriteString(snippet)
			sb.WriteString("  ───────\n")
		}
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
