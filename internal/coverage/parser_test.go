package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

const specFixture = `# GoAtlas Spec

## Feature: User Authentication

Allow users to log in.

**Backend:**
- POST /auth/login
- AuthService.Login()

**Frontend:**
- LoginScreen

## Feature: Order Management

Manage orders.

**Backend:**
- GET /orders
- POST /orders
- OrderService.Create()

## Not a Feature

This section should not be captured.

### Feature: Nested Feature

Should also be captured.
`

func TestParseSpecFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(f, []byte(specFixture), 0600); err != nil {
		t.Fatal(err)
	}

	sections, err := ParseSpecFile(f)
	if err != nil {
		t.Fatalf("ParseSpecFile error: %v", err)
	}

	if len(sections) < 2 {
		t.Fatalf("expected at least 2 sections, got %d", len(sections))
	}

	names := map[string]bool{}
	for _, s := range sections {
		names[s.Name] = true
		if s.RawText == "" {
			t.Errorf("section %q has empty RawText", s.Name)
		}
	}

	if !names["User Authentication"] {
		t.Error("expected section 'User Authentication'")
	}
	if !names["Order Management"] {
		t.Error("expected section 'Order Management'")
	}
}

func TestParseSpecFile_NotFound(t *testing.T) {
	_, err := ParseSpecFile("/nonexistent/spec.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestIsFeatureHeader(t *testing.T) {
	cases := []struct {
		line     string
		expected bool
	}{
		{"## Feature: User Auth", true},
		{"### Feature: Something", true},
		{"## feature: lowercase", true},
		{"## Feature Management", true},
		{"## Something Else", false},
		{"# Feature: Top Level", false},
		{"#### Feature: Too Deep", false},
	}

	for _, tc := range cases {
		got := isFeatureHeader(tc.line)
		if got != tc.expected {
			t.Errorf("isFeatureHeader(%q) = %v, want %v", tc.line, got, tc.expected)
		}
	}
}

func TestExtractFeatureName(t *testing.T) {
	cases := []struct {
		line     string
		expected string
	}{
		{"## Feature: User Authentication", "User Authentication"},
		{"### Feature: Order Management", "Order Management"},
		{"## feature: lowercase test", "lowercase test"},
	}

	for _, tc := range cases {
		got := extractFeatureName(tc.line)
		if got != tc.expected {
			t.Errorf("extractFeatureName(%q) = %q, want %q", tc.line, got, tc.expected)
		}
	}
}
