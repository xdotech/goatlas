package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/goatlas/goatlas/internal/graph"
	"github.com/goatlas/goatlas/internal/indexer/domain"
)

// TraceTypeFlowUseCase traces how a type flows through the codebase.
type TraceTypeFlowUseCase struct {
	tuRepo  domain.TypeUsageRepository
	querier *graph.Querier
}

// NewTraceTypeFlowUseCase constructs a TraceTypeFlowUseCase.
func NewTraceTypeFlowUseCase(tuRepo domain.TypeUsageRepository, querier *graph.Querier) *TraceTypeFlowUseCase {
	return &TraceTypeFlowUseCase{tuRepo: tuRepo, querier: querier}
}

// Execute traces type flow for the given type name.
// direction: "upstream" (who produces this type), "downstream" (who consumes), "both"
func (uc *TraceTypeFlowUseCase) Execute(ctx context.Context, typeName, direction string) (string, error) {
	if direction == "" {
		direction = "both"
	}

	usages, err := uc.tuRepo.FindByType(ctx, typeName)
	if err != nil {
		return "", fmt.Errorf("find type usages: %w", err)
	}

	if len(usages) == 0 {
		return fmt.Sprintf("No type flow found for %q. Make sure the repository is indexed.", typeName), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Type flow analysis for %q:\n\n", typeName)

	// Separate producers (output) and consumers (input)
	var producers, consumers []domain.TypeUsage
	for _, u := range usages {
		switch u.Direction {
		case "output":
			producers = append(producers, u)
		case "input":
			consumers = append(consumers, u)
		}
	}

	if direction == "upstream" || direction == "both" {
		fmt.Fprintf(&sb, "── Producers (functions returning this type): %d\n", len(producers))
		if len(producers) == 0 {
			sb.WriteString("   (none found)\n")
		}
		for _, p := range producers {
			fmt.Fprintf(&sb, "   ← %s (return pos %d, line %d)\n", p.SymbolName, p.Position, p.Line)
		}
		sb.WriteString("\n")
	}

	if direction == "downstream" || direction == "both" {
		fmt.Fprintf(&sb, "── Consumers (functions accepting this type): %d\n", len(consumers))
		if len(consumers) == 0 {
			sb.WriteString("   (none found)\n")
		}
		for _, c := range consumers {
			fmt.Fprintf(&sb, "   → %s (param pos %d, line %d)\n", c.SymbolName, c.Position, c.Line)
		}
		sb.WriteString("\n")
	}

	// Try to trace the flow chain via call graph if querier is available
	if uc.querier != nil && len(producers) > 0 && len(consumers) > 0 {
		sb.WriteString("── Data flow chain (producer → consumer via call graph):\n")
		chainFound := false
		for _, prod := range producers {
			for _, cons := range consumers {
				if prod.SymbolName != cons.SymbolName {
					// Check if producer calls consumer (or vice versa)
					callers, _ := uc.querier.FindCallers(ctx, cons.SymbolName, 3)
					for _, c := range callers {
						if c.QualifiedName == prod.SymbolName || c.Name == prod.SymbolName {
							fmt.Fprintf(&sb, "   %s → %s (depth %d)\n", prod.SymbolName, cons.SymbolName, c.Depth)
							chainFound = true
							break
						}
					}
				}
			}
		}
		if !chainFound {
			sb.WriteString("   (no direct call chains found between producers and consumers)\n")
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "Summary: %d producers, %d consumers\n", len(producers), len(consumers))
	return sb.String(), nil
}
