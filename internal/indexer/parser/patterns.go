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
	HTTPClient    []GoCallPattern `yaml:"http_client"`
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

// LoadPatterns loads detection patterns with the following priority:
//  1. {repoPath}/goatlas.yaml           (per-repo override)
//  2. ~/.goatlas/goatlas.yaml             (global user config)
//  3. Embedded default_patterns.yaml     (fallback)
func LoadPatterns(repoPath string) (*PatternConfig, error) {
	data := defaultPatternsYAML

	// Priority 2: global config
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".goatlas", "goatlas.yaml")
		if fileData, err := os.ReadFile(globalPath); err == nil {
			data = fileData
		}
	}

	// Priority 1: repo-local config (overrides global)
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
