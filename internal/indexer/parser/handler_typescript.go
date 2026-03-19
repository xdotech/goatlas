package parser

import "context"

// TypeScriptHandler implements LanguageHandler for JS/TS/JSX/TSX files.
type TypeScriptHandler struct{}

func (h *TypeScriptHandler) Language() Language   { return LangTypeScript }
func (h *TypeScriptHandler) Extensions() []string { return []string{".ts", ".tsx", ".js", ".jsx"} }

func (h *TypeScriptHandler) Parse(ctx context.Context, filePath string, cfg *PatternConfig) (*FileResult, error) {
	result := &FileResult{}

	parsed, err := ParseJSXFile(filePath)
	if err != nil {
		return nil, err
	}
	result.Symbols = parsed.Symbols
	result.Imports = parsed.Imports

	if routes, err := ExtractReactRoutes(filePath, parsed.Imports); err == nil {
		result.Endpoints = routes
	}

	if cfg != nil {
		if conns, err := DetectTSAPIConnections(filePath, cfg.TypeScript); err == nil {
			result.Connections = conns
		}

		if calls, err := DetectComponentAPICalls(filePath, cfg.TypeScript); err == nil {
			result.CompAPICalls = calls
		}
	}

	return result, nil
}
