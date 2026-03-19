package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/xdotech/goatlas/internal/graph"
)

// AnalyzeImpactUseCase analyzes the change impact of modifying a function.
type AnalyzeImpactUseCase struct {
	querier *graph.Querier
}

// NewAnalyzeImpactUseCase constructs an AnalyzeImpactUseCase.
func NewAnalyzeImpactUseCase(querier *graph.Querier) *AnalyzeImpactUseCase {
	return &AnalyzeImpactUseCase{querier: querier}
}

// Execute performs impact analysis for a given symbol.
func (uc *AnalyzeImpactUseCase) Execute(ctx context.Context, symbol string, maxDepth int) (string, error) {
	if uc.querier == nil {
		return "Impact analysis requires Neo4j graph. Build the graph first with build_graph.", nil
	}
	if maxDepth <= 0 {
		maxDepth = 5
	}

	report, err := uc.querier.AnalyzeImpact(ctx, symbol, maxDepth)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Impact analysis for %q:\n\n", symbol)

	// Group callers by depth
	if len(report.Callers) == 0 {
		sb.WriteString("  No callers found in the graph.\n")
	} else {
		// Direct callers (depth 1)
		var direct, transitive []graph.CallerResult
		for _, c := range report.Callers {
			if c.Depth == 1 {
				direct = append(direct, c)
			} else {
				transitive = append(transitive, c)
			}
		}

		if len(direct) > 0 {
			fmt.Fprintf(&sb, "Direct callers (depth 1): %d\n", len(direct))
			for _, c := range direct {
				fmt.Fprintf(&sb, "  - [conf %.2f] %s", c.Confidence, c.QualifiedName)
				if c.File != "" {
					fmt.Fprintf(&sb, " @ %s:%d", c.File, c.Line)
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		if len(transitive) > 0 {
			fmt.Fprintf(&sb, "Transitive callers (depth 2-%d): %d\n", maxDepth, len(transitive))
			for _, c := range transitive {
				fmt.Fprintf(&sb, "  - [depth %d, conf %.2f] %s", c.Depth, c.Confidence, c.QualifiedName)
				if c.File != "" {
					fmt.Fprintf(&sb, " @ %s:%d", c.File, c.Line)
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// Affected endpoints
	if len(report.AffectedEndpoints) > 0 {
		fmt.Fprintf(&sb, "Affected API endpoints: %d\n", len(report.AffectedEndpoints))
		for _, ep := range report.AffectedEndpoints {
			fmt.Fprintf(&sb, "  - %s %s\n", ep.Method, ep.Path)
		}
		sb.WriteString("\n")
	}

	// Affected UI components
	if len(report.AffectedComponents) > 0 {
		fmt.Fprintf(&sb, "Affected UI components: %d\n", len(report.AffectedComponents))
		for _, comp := range report.AffectedComponents {
			fmt.Fprintf(&sb, "  - %s\n", comp)
		}
		sb.WriteString("\n")
	}

	fmt.Fprintf(&sb, "Total impact: %d functions, %d endpoints, %d UI components\n",
		len(report.Callers), len(report.AffectedEndpoints), len(report.AffectedComponents))

	return sb.String(), nil
}
