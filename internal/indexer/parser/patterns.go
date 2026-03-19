package parser

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed default_patterns.yaml
var defaultPatternsYAML []byte

// PatternConfig holds all detection patterns.
// Each language has a flat list of patterns — conn_type on each pattern
// determines what kind of connection it represents. No hardcoded categories,
// so any conn_type (grpc, kafka_consume, redis, nats, …) is supported.
type PatternConfig struct {
	Go         []GoCallPattern   `yaml:"go"`
	TypeScript []TSAPIPattern    `yaml:"typescript"`
	Python     []PyCallPattern   `yaml:"python"`
	Java       []JavaCallPattern `yaml:"java"`
}

// GoCallPattern describes a Go function call pattern to detect.
type GoCallPattern struct {
	PackageSuffix   string   `yaml:"package_suffix"`
	PackageContains string   `yaml:"package_contains"`
	Function        string   `yaml:"function"`
	Functions       []string `yaml:"functions"`
	TargetArg       int      `yaml:"target_arg"`
	ConnType        string   `yaml:"conn_type"`
}

// TSAPIPattern describes a regex pattern to detect API service references in TS/JS files.
type TSAPIPattern struct {
	Pattern  string `yaml:"pattern"`
	ConnType string `yaml:"conn_type"`
	FileGlob string `yaml:"file_glob"`
}

// PyCallPattern describes a Python call pattern for connection detection.
type PyCallPattern struct {
	ModuleContains string `yaml:"module_contains"`
	CallPattern    string `yaml:"call_pattern"`
	TargetArgIndex int    `yaml:"target_arg"`
	TargetKeyword  string `yaml:"target_keyword"`
	ConnType       string `yaml:"conn_type"`
}

// JavaCallPattern describes a Java AST pattern for connection detection.
type JavaCallPattern struct {
	ImportContains  string `yaml:"import_contains"`
	Annotation      string `yaml:"annotation"`
	TargetAttribute string `yaml:"target_attribute"`
	MethodCall      string `yaml:"method_call"`
	TargetArgIndex  int    `yaml:"target_arg"`
	ConnType        string `yaml:"conn_type"`
}

// LoadPatterns loads detection patterns with the following merged priority:
//  1. Embedded default_patterns.yaml (base)
//  2. Auto-discovered catalog patterns (from dep files)
//  3. ~/.goatlas/goatlas.yaml (global user override)
//  4. {repoPath}/goatlas.yaml (per-repo override, highest)
func LoadPatterns(repoPath string) (*PatternConfig, error) {
	var cfg PatternConfig
	if err := yaml.Unmarshal(defaultPatternsYAML, &cfg); err != nil {
		return nil, fmt.Errorf("parse default patterns: %w", err)
	}

	// Step 2: auto-discover deps and merge catalog patterns
	if deps, err := DiscoverDependencies(repoPath); err == nil {
		catalogCfg := LookupCatalog(deps)
		mergePatterns(&cfg, catalogCfg)
	}

	// Step 3: global user override (~/.goatlas/goatlas.yaml)
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".goatlas", "goatlas.yaml")
		if fileData, err := os.ReadFile(globalPath); err == nil {
			var overrideCfg PatternConfig
			if err := yaml.Unmarshal(fileData, &overrideCfg); err == nil {
				mergePatterns(&cfg, &overrideCfg)
			}
		}
	}

	// Step 4: per-repo override ({repoPath}/goatlas.yaml)
	repoConfigPath := filepath.Join(repoPath, "goatlas.yaml")
	if fileData, err := os.ReadFile(repoConfigPath); err == nil {
		var overrideCfg PatternConfig
		if err := yaml.Unmarshal(fileData, &overrideCfg); err == nil {
			mergePatterns(&cfg, &overrideCfg)
		}
	}

	return &cfg, nil
}

// mergePatterns appends patterns from src into dst (additive, no dedup).
func mergePatterns(dst, src *PatternConfig) {
	dst.Go = append(dst.Go, src.Go...)
	dst.TypeScript = append(dst.TypeScript, src.TypeScript...)
	dst.Python = append(dst.Python, src.Python...)
	dst.Java = append(dst.Java, src.Java...)
}
