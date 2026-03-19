package agent

import "github.com/google/generative-ai-go/genai"

// toolDef is a provider-agnostic tool definition.
type toolDef struct {
	Name        string
	Description string
	Properties  map[string]toolProp
	Required    []string
}

type toolProp struct {
	Type        string // "string" | "integer"
	Description string
}

// toolSchemas returns all MCP tool definitions in a provider-agnostic format.
func toolSchemas() []toolDef {
	s := func(desc string) toolProp { return toolProp{Type: "string", Description: desc} }
	i := func(desc string) toolProp { return toolProp{Type: "integer", Description: desc} }

	return []toolDef{
		{
			Name:        "search_code",
			Description: "Search code symbols by keyword or name",
			Properties: map[string]toolProp{
				"query": s("Search query"),
				"limit": i("Max results"),
				"kind":  s("Symbol kind: func|method|type|interface"),
				"mode":  s("Search mode: keyword|semantic|hybrid"),
			},
			Required: []string{"query"},
		},
		{
			Name:        "read_file",
			Description: "Read a file from the indexed repository",
			Properties: map[string]toolProp{
				"path":       s("Relative file path"),
				"start_line": i("Start line (optional)"),
				"end_line":   i("End line (optional)"),
			},
			Required: []string{"path"},
		},
		{
			Name:        "find_symbol",
			Description: "Find a symbol by name",
			Properties: map[string]toolProp{
				"name": s("Symbol name"),
				"kind": s("Symbol kind filter"),
			},
			Required: []string{"name"},
		},
		{
			Name:        "find_callers",
			Description: "Find functions that reference the given function",
			Properties: map[string]toolProp{
				"function_name": s("Function name to find callers for"),
			},
			Required: []string{"function_name"},
		},
		{
			Name:        "list_api_endpoints",
			Description: "List all detected API endpoints",
			Properties: map[string]toolProp{
				"method":  s("HTTP method filter"),
				"service": s("Service filter"),
			},
		},
		{
			Name:        "get_file_symbols",
			Description: "Get all symbols defined in a file",
			Properties: map[string]toolProp{
				"path": s("Relative file path"),
			},
			Required: []string{"path"},
		},
		{
			Name:        "list_services",
			Description: "List all top-level packages/services in the repository",
			Properties:  map[string]toolProp{},
		},
		{
			Name:        "get_service_dependencies",
			Description: "Get packages that a service imports/depends on",
			Properties: map[string]toolProp{
				"service": s("Service/package name"),
			},
			Required: []string{"service"},
		},
		{
			Name:        "get_api_handlers",
			Description: "Find handlers for an API endpoint pattern",
			Properties: map[string]toolProp{
				"endpoint_pattern": s("Endpoint pattern to search"),
			},
			Required: []string{"endpoint_pattern"},
		},
	}
}

// toolDeclarations converts tool schemas to Gemini FunctionDeclaration format.
func toolDeclarations() []*genai.FunctionDeclaration {
	schemas := toolSchemas()
	decls := make([]*genai.FunctionDeclaration, 0, len(schemas))

	for _, t := range schemas {
		props := make(map[string]*genai.Schema, len(t.Properties))
		for name, p := range t.Properties {
			switch p.Type {
			case "integer":
				props[name] = &genai.Schema{Type: genai.TypeInteger, Description: p.Description}
			default:
				props[name] = &genai.Schema{Type: genai.TypeString, Description: p.Description}
			}
		}
		decls = append(decls, &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters: &genai.Schema{
				Type:       genai.TypeObject,
				Properties: props,
				Required:   t.Required,
			},
		})
	}
	return decls
}
