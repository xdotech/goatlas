package parser

import "context"

// JavaHandler implements LanguageHandler for Java source files.
type JavaHandler struct{}

func (h *JavaHandler) Language() Language   { return LangJava }
func (h *JavaHandler) Extensions() []string { return []string{".java"} }

func (h *JavaHandler) Parse(ctx context.Context, filePath string, cfg *PatternConfig) (*FileResult, error) {
	result := &FileResult{}

	parsed, err := ParseJavaFile(filePath)
	if err != nil {
		return nil, err
	}
	result.Symbols = parsed.Symbols
	result.Imports = parsed.Imports
	result.Endpoints = parsed.Endpoints
	result.Module = parsed.Module

	result.Framework = DetectFrameworkFromPath(filePath)

	if cfg != nil {
		if conns, err := DetectJavaConnections(filePath, cfg.Java); err == nil {
			result.Connections = conns
		}
	}


	return result, nil
}
