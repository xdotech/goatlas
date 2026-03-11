package coverage

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func makeReport() *CoverageReport {
	return &CoverageReport{
		SpecFile:    "spec.md",
		GeneratedAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		TotalFeatures: 3,
		Implemented: 1,
		Partial:     1,
		Missing:     1,
		CoveragePercent: 33.33,
		Features: []FeatureResult{
			{
				Feature: Feature{
					Name: "User Auth",
					Backend: []Component{
						{Type: "api_endpoint", Identifier: "POST /auth/login", Status: StatusImplemented, Evidence: []string{"login_handler.go:42"}},
					},
				},
				OverallStatus: StatusImplemented,
			},
			{
				Feature: Feature{
					Name: "Order Mgmt",
					Backend: []Component{
						{Type: "service_method", Identifier: "OrderService.Create", Status: StatusPartial},
					},
				},
				OverallStatus: StatusPartial,
			},
			{
				Feature: Feature{
					Name:     "Payment",
					Backend:  []Component{{Type: "api_endpoint", Identifier: "POST /pay", Status: StatusMissing}},
					Frontend: []Component{{Type: "ui_screen", Identifier: "PaymentScreen", Status: StatusMissing}},
				},
				OverallStatus: StatusMissing,
			},
		},
	}
}

func TestRenderText(t *testing.T) {
	report := makeReport()
	output := RenderText(report)

	checks := []string{
		"Coverage Report: spec.md",
		"User Auth",
		"Order Mgmt",
		"Payment",
		"1/3",
		"33%",
		"1 partial",
		"1 missing",
		"✓",
		"⚠",
		"✗",
	}

	for _, c := range checks {
		if !strings.Contains(output, c) {
			t.Errorf("RenderText missing %q in output", c)
		}
	}
}

func TestRenderJSON(t *testing.T) {
	report := makeReport()
	data, err := RenderJSON(report)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed["SpecFile"] != "spec.md" {
		t.Errorf("expected SpecFile=spec.md in JSON")
	}
	if parsed["TotalFeatures"].(float64) != 3 {
		t.Errorf("expected TotalFeatures=3")
	}
}

func TestRenderMarkdown(t *testing.T) {
	report := makeReport()
	output := RenderMarkdown(report)

	if !strings.HasPrefix(output, "# Coverage Report:") {
		t.Error("expected Markdown H1 header")
	}
	if !strings.Contains(output, "## ") {
		t.Error("expected H2 section headers")
	}
	if !strings.Contains(output, "User Auth") {
		t.Error("expected 'User Auth' in markdown output")
	}
}

func TestGenerateReport(t *testing.T) {
	features := []FeatureResult{
		{OverallStatus: StatusImplemented},
		{OverallStatus: StatusImplemented},
		{OverallStatus: StatusPartial},
		{OverallStatus: StatusMissing},
	}

	report := GenerateReport(features, "test.md")

	if report.TotalFeatures != 4 {
		t.Errorf("expected 4 total features, got %d", report.TotalFeatures)
	}
	if report.Implemented != 2 {
		t.Errorf("expected 2 implemented, got %d", report.Implemented)
	}
	if report.Partial != 1 {
		t.Errorf("expected 1 partial, got %d", report.Partial)
	}
	if report.Missing != 1 {
		t.Errorf("expected 1 missing, got %d", report.Missing)
	}
	if report.CoveragePercent != 50.0 {
		t.Errorf("expected 50%% coverage, got %.2f", report.CoveragePercent)
	}
}

func TestStatusIcon(t *testing.T) {
	if statusIcon(StatusImplemented) != "✓" {
		t.Error("expected ✓ for implemented")
	}
	if statusIcon(StatusPartial) != "⚠" {
		t.Error("expected ⚠ for partial")
	}
	if statusIcon(StatusMissing) != "✗" {
		t.Error("expected ✗ for missing")
	}
}
