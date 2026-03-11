package coverage

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GenerateReport computes summary statistics over feature results.
func GenerateReport(features []FeatureResult, specFile string) *CoverageReport {
	report := &CoverageReport{
		SpecFile:      specFile,
		GeneratedAt:   time.Now(),
		Features:      features,
		TotalFeatures: len(features),
	}
	for _, f := range features {
		switch f.OverallStatus {
		case StatusImplemented:
			report.Implemented++
		case StatusPartial:
			report.Partial++
		case StatusMissing:
			report.Missing++
		}
	}
	if report.TotalFeatures > 0 {
		report.CoveragePercent = float64(report.Implemented) / float64(report.TotalFeatures) * 100
	}
	return report
}

// RenderText formats a coverage report as plain text.
func RenderText(report *CoverageReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Coverage Report: %s\n", report.SpecFile))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))

	for _, fr := range report.Features {
		icon := statusIcon(fr.OverallStatus)
		sb.WriteString(fmt.Sprintf("%s Feature: %s\n", icon, fr.Feature.Name))

		if len(fr.Feature.Backend) > 0 {
			sb.WriteString("  Backend:\n")
			for _, c := range fr.Feature.Backend {
				sb.WriteString(fmt.Sprintf("    %s %s [%s]", statusIcon(c.Status), c.Identifier, c.Type))
				if len(c.Evidence) > 0 {
					sb.WriteString(fmt.Sprintf(" -> %s", strings.Join(c.Evidence, ", ")))
				}
				sb.WriteString("\n")
			}
		}
		if len(fr.Feature.Frontend) > 0 {
			sb.WriteString("  Frontend:\n")
			for _, c := range fr.Feature.Frontend {
				sb.WriteString(fmt.Sprintf("    %s %s [%s]\n", statusIcon(c.Status), c.Identifier, c.Type))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Summary: %d/%d features fully implemented (%.0f%%)\n",
		report.Implemented, report.TotalFeatures, report.CoveragePercent))
	sb.WriteString(fmt.Sprintf("         %d partial, %d missing\n", report.Partial, report.Missing))
	return sb.String()
}

// RenderJSON marshals a coverage report to indented JSON.
func RenderJSON(report *CoverageReport) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}

// RenderMarkdown formats a coverage report as Markdown.
func RenderMarkdown(report *CoverageReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Coverage Report: %s\n\n", report.SpecFile))
	sb.WriteString(fmt.Sprintf("**Coverage:** %.0f%% (%d/%d features)\n\n",
		report.CoveragePercent, report.Implemented, report.TotalFeatures))

	for _, fr := range report.Features {
		sb.WriteString(fmt.Sprintf("## %s %s\n\n", statusIcon(fr.OverallStatus), fr.Feature.Name))
		if len(fr.Feature.Backend) > 0 {
			sb.WriteString("**Backend:**\n")
			for _, c := range fr.Feature.Backend {
				sb.WriteString(fmt.Sprintf("- %s `%s`", statusIcon(c.Status), c.Identifier))
				if len(c.Evidence) > 0 {
					sb.WriteString(fmt.Sprintf(" *(found: %s)*", strings.Join(c.Evidence, ", ")))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func statusIcon(s Status) string {
	switch s {
	case StatusImplemented:
		return "✓"
	case StatusPartial:
		return "⚠"
	default:
		return "✗"
	}
}
