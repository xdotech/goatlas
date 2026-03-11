package domain

// SearchCodeInput for search_code tool.
type SearchCodeInput struct {
	Query string
	Limit int
	Kind  string
	Mode  string // "keyword" | "semantic" | "hybrid" (default: keyword)
}

// ReadFileInput for read_file tool.
type ReadFileInput struct {
	Path      string
	StartLine int
	EndLine   int
}

// FindSymbolInput for find_symbol tool.
type FindSymbolInput struct {
	Name string
	Kind string
}

// FindCallersInput for find_callers tool.
type FindCallersInput struct {
	FunctionName string
}

// ListEndpointsInput for list_api_endpoints tool.
type ListEndpointsInput struct {
	Service string
	Method  string
}

// GetFileSymbolsInput for get_file_symbols tool.
type GetFileSymbolsInput struct {
	Path string
}
