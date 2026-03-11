package coverage

import "time"

// Status of a feature component implementation
type Status string

const (
	StatusImplemented Status = "implemented"
	StatusPartial     Status = "partial"
	StatusMissing     Status = "missing"
)

// Component is a specific implementable piece of a feature
type Component struct {
	Type       string   // api_endpoint|service_method|ui_screen|api_call
	Identifier string   // POST /path, MethodName, ScreenName
	Status     Status
	Evidence   []string // matched symbols/files
	Confidence float64  // 0.0 - 1.0
}

// Feature is a high-level product feature from the spec
type Feature struct {
	Name        string
	Description string
	Backend     []Component
	Frontend    []Component
}

// CoverageReport is the full analysis result
type CoverageReport struct {
	SpecFile        string
	GeneratedAt     time.Time
	Features        []FeatureResult
	TotalFeatures   int
	Implemented     int
	Partial         int
	Missing         int
	CoveragePercent float64
}

// FeatureResult is a feature with its coverage analysis
type FeatureResult struct {
	Feature        Feature
	BackendStatus  Status
	FrontendStatus Status
	OverallStatus  Status
}

// FeatureSection is a raw parsed section from the spec file
type FeatureSection struct {
	Name    string
	RawText string
}
