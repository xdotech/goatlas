package parser

import (
	"path/filepath"
	"strings"
)

// FrameworkHint carries a detected framework and entry point multiplier.
type FrameworkHint struct {
	Framework            string
	EntryPointMultiplier float64
	Reason               string
}

// DetectFrameworkFromPath infers a framework from the file path alone.
// Returns nil if no known framework pattern matches.
func DetectFrameworkFromPath(filePath string) *FrameworkHint {
	base := filepath.Base(filePath)
	dir := filepath.Dir(filePath)
	dirBase := filepath.Base(dir)
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".py":
		return detectPythonFramework(base, dirBase)
	case ".java":
		return detectJavaFramework(base, dirBase)
	}
	return nil
}

func detectPythonFramework(base, dirBase string) *FrameworkHint {
	switch base {
	case "views.py":
		return &FrameworkHint{Framework: "django", EntryPointMultiplier: 3.0, Reason: "Django views file"}
	case "urls.py":
		return &FrameworkHint{Framework: "django", EntryPointMultiplier: 2.0, Reason: "Django URL config"}
	}

	if base == "__init__.py" {
		return nil
	}

	switch dirBase {
	case "routers", "endpoints":
		return &FrameworkHint{Framework: "fastapi", EntryPointMultiplier: 2.5, Reason: "FastAPI " + dirBase + " directory"}
	case "routes":
		return &FrameworkHint{Framework: "flask", EntryPointMultiplier: 2.5, Reason: "Flask routes directory"}
	case "api":
		return &FrameworkHint{Framework: "python-api", EntryPointMultiplier: 2.0, Reason: "Python API directory"}
	}
	return nil
}

func detectJavaFramework(base, dirBase string) *FrameworkHint {
	switch dirBase {
	case "controller", "controllers":
		return &FrameworkHint{Framework: "spring", EntryPointMultiplier: 3.0, Reason: "Spring controller directory"}
	case "handler":
		return &FrameworkHint{Framework: "spring", EntryPointMultiplier: 2.5, Reason: "Spring handler directory"}
	case "service":
		return &FrameworkHint{Framework: "spring", EntryPointMultiplier: 1.8, Reason: "Spring service directory"}
	case "resource":
		return &FrameworkHint{Framework: "jax-rs", EntryPointMultiplier: 3.0, Reason: "JAX-RS resource directory"}
	}
	_ = base
	return nil
}
