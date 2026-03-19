package parser

import "context"

// GoHandler implements LanguageHandler for Go source files.
type GoHandler struct{}

func (h *GoHandler) Language() Language   { return LangGo }
func (h *GoHandler) Extensions() []string { return []string{".go"} }

func (h *GoHandler) Parse(ctx context.Context, filePath string, cfg *PatternConfig) (*FileResult, error) {
	result := &FileResult{}

	parsed, err := ParseFile(filePath)
	if err != nil {
		return nil, err
	}
	result.Symbols = parsed.Symbols
	result.Imports = parsed.Imports
	result.Module = parsed.Module

	if routes, err := ExtractRoutes(filePath, parsed.Imports); err == nil {
		result.Endpoints = routes
	}

	if connResult, err := DetectConnections(filePath, cfg); err == nil {
		result.Connections = connResult.Connections
	}

	if calls, err := ExtractFunctionCalls(filePath); err == nil {
		result.FuncCalls = calls
	}

	if usages, err := ExtractTypeUsages(filePath); err == nil {
		result.TypeUsages = usages
	}

	if impls, err := ExtractInterfaceImpls(filePath); err == nil {
		result.IfaceImpls = impls
	}

	return result, nil
}
