package agent

import "github.com/google/generative-ai-go/genai"

// toolDeclarations returns the Gemini FunctionDeclaration schema for all 9 MCP tools.
func toolDeclarations() []*genai.FunctionDeclaration {
	str := genai.TypeString
	integer := genai.TypeInteger
	obj := genai.TypeObject

	prop := func(props map[string]*genai.Schema, required []string) *genai.Schema {
		return &genai.Schema{Type: obj, Properties: props, Required: required}
	}
	s := func(desc string) *genai.Schema { return &genai.Schema{Type: str, Description: desc} }
	i := func(desc string) *genai.Schema { return &genai.Schema{Type: integer, Description: desc} }

	return []*genai.FunctionDeclaration{
		{
			Name:        "search_code",
			Description: "Search code symbols by keyword or name",
			Parameters: prop(map[string]*genai.Schema{
				"query": s("Search query"),
				"limit": i("Max results"),
				"kind":  s("Symbol kind: func|method|type|interface"),
				"mode":  s("Search mode: keyword|semantic|hybrid"),
			}, []string{"query"}),
		},
		{
			Name:        "read_file",
			Description: "Read a file from the indexed repository",
			Parameters: prop(map[string]*genai.Schema{
				"path":       s("Relative file path"),
				"start_line": i("Start line (optional)"),
				"end_line":   i("End line (optional)"),
			}, []string{"path"}),
		},
		{
			Name:        "find_symbol",
			Description: "Find a symbol by name",
			Parameters: prop(map[string]*genai.Schema{
				"name": s("Symbol name"),
				"kind": s("Symbol kind filter"),
			}, []string{"name"}),
		},
		{
			Name:        "find_callers",
			Description: "Find functions that reference the given function",
			Parameters: prop(map[string]*genai.Schema{
				"function_name": s("Function name to find callers for"),
			}, []string{"function_name"}),
		},
		{
			Name:        "list_api_endpoints",
			Description: "List all detected API endpoints",
			Parameters: prop(map[string]*genai.Schema{
				"method":  s("HTTP method filter"),
				"service": s("Service filter"),
			}, nil),
		},
		{
			Name:        "get_file_symbols",
			Description: "Get all symbols defined in a file",
			Parameters: prop(map[string]*genai.Schema{
				"path": s("Relative file path"),
			}, []string{"path"}),
		},
		{
			Name:        "list_services",
			Description: "List all top-level packages/services in the repository",
			Parameters:  prop(map[string]*genai.Schema{}, nil),
		},
		{
			Name:        "get_service_dependencies",
			Description: "Get packages that a service imports/depends on",
			Parameters: prop(map[string]*genai.Schema{
				"service": s("Service/package name"),
			}, []string{"service"}),
		},
		{
			Name:        "get_api_handlers",
			Description: "Find handlers for an API endpoint pattern",
			Parameters: prop(map[string]*genai.Schema{
				"endpoint_pattern": s("Endpoint pattern to search"),
			}, []string{"endpoint_pattern"}),
		},
	}
}
