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

// PatternConfig holds all detection patterns loaded from goatlas.yaml.
type PatternConfig struct {
	Go         GoPatterns         `yaml:"go"`
	TypeScript TypeScriptPatterns `yaml:"typescript"`
}

// GoPatterns groups Go AST-based detection patterns.
type GoPatterns struct {
	GRPC          []GoCallPattern `yaml:"grpc"`
	KafkaConsumer []GoCallPattern `yaml:"kafka_consumer"`
	KafkaProducer []GoCallPattern `yaml:"kafka_producer"`
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

// TypeScriptPatterns groups TypeScript detection patterns.
type TypeScriptPatterns struct {
	APIPrefix []TSAPIPattern `yaml:"api_prefix"`
}

// TSAPIPattern describes a regex pattern to detect API service references in TS files.
type TSAPIPattern struct {
	Pattern  string `yaml:"pattern"`
	ConnType string `yaml:"conn_type"`
	FileGlob string `yaml:"file_glob"`
}

// LoadPatterns loads detection patterns from goatlas.yaml in the repo root,
// falling back to embedded defaults if the file doesn't exist.
func LoadPatterns(repoPath string) (*PatternConfig, error) {
	data := defaultPatternsYAML

	// Try to load from repo-local goatlas.yaml
	configPath := filepath.Join(repoPath, "goatlas.yaml")
	if fileData, err := os.ReadFile(configPath); err == nil {
		data = fileData
	}

	var cfg PatternConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse patterns config: %w", err)
	}
	return &cfg, nil
}
