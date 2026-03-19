package parser

import "context"

// PythonHandler implements LanguageHandler for Python source files.
type PythonHandler struct{}

func (h *PythonHandler) Language() Language   { return LangPython }
func (h *PythonHandler) Extensions() []string { return []string{".py"} }

func (h *PythonHandler) Parse(ctx context.Context, filePath string, cfg *PatternConfig) (*FileResult, error) {
	result := &FileResult{}

	parsed, err := ParsePythonFile(filePath)
	if err != nil {
		return nil, err
	}
	result.Symbols = parsed.Symbols
	result.Imports = parsed.Imports
	result.Endpoints = parsed.Endpoints
	result.Module = parsed.Module

	result.Framework = DetectFrameworkFromPath(filePath)

	if cfg != nil {
		if conns, err := DetectPythonConnections(filePath, cfg.Python); err == nil {
			result.Connections = conns
		}
	}


	return result, nil
}
