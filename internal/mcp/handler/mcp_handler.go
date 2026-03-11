package handler

import (
	"context"

	"github.com/goatlas/goatlas/internal/mcp/usecase"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPHandler registers all MCP tools with the server.
type MCPHandler struct {
	searchCode     *usecase.SearchCodeUseCase
	readFile       *usecase.ReadFileUseCase
	findSymbol     *usecase.FindSymbolUseCase
	findCallers    *usecase.FindCallersUseCase
	listEndpoints  *usecase.ListEndpointsUseCase
	getFileSymbols *usecase.GetFileSymbolsUseCase
	listServices   *usecase.ListServicesUseCase
	getServiceDeps *usecase.GetServiceDepsUseCase
	getAPIHandlers *usecase.GetAPIHandlersUseCase
	listComponents *usecase.ListComponentsUseCase
}

// NewMCPHandler constructs an MCPHandler with all use cases.
func NewMCPHandler(
	sc *usecase.SearchCodeUseCase,
	rf *usecase.ReadFileUseCase,
	fs *usecase.FindSymbolUseCase,
	fc *usecase.FindCallersUseCase,
	le *usecase.ListEndpointsUseCase,
	gfs *usecase.GetFileSymbolsUseCase,
	ls *usecase.ListServicesUseCase,
	gsd *usecase.GetServiceDepsUseCase,
	gah *usecase.GetAPIHandlersUseCase,
	lc *usecase.ListComponentsUseCase,
) *MCPHandler {
	return &MCPHandler{
		searchCode:     sc,
		readFile:       rf,
		findSymbol:     fs,
		findCallers:    fc,
		listEndpoints:  le,
		getFileSymbols: gfs,
		listServices:   ls,
		getServiceDeps: gsd,
		getAPIHandlers: gah,
		listComponents: lc,
	}
}

// RegisterTools adds all GoAtlas tools to the MCP server.
func (h *MCPHandler) RegisterTools(srv *server.MCPServer) {
	srv.AddTool(mcp.NewTool("search_code",
		mcp.WithDescription("Search code symbols by keyword or name"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 20)")),
		mcp.WithString("kind", mcp.Description("Symbol kind filter: func|method|type|interface|struct|const|var")),
		mcp.WithString("mode", mcp.Description("Search mode: keyword (default)|semantic|hybrid")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		limit := req.GetInt("limit", 20)
		kind := req.GetString("kind", "")
		mode := req.GetString("mode", "keyword")
		result, err := h.searchCode.Execute(ctx, query, limit, kind, mode)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("read_file",
		mcp.WithDescription("Read a file from the indexed repository with optional line range"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative file path from repo root")),
		mcp.WithNumber("start_line", mcp.Description("Start line, 1-based (optional)")),
		mcp.WithNumber("end_line", mcp.Description("End line, 1-based (optional)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := req.GetString("path", "")
		startLine := req.GetInt("start_line", 0)
		endLine := req.GetInt("end_line", 0)
		result, err := h.readFile.Execute(ctx, path, startLine, endLine)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("find_symbol",
		mcp.WithDescription("Find a symbol by name and optional kind filter"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Symbol name to search for")),
		mcp.WithString("kind", mcp.Description("Symbol kind filter: func|method|type|interface|struct|const|var")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		kind := req.GetString("kind", "")
		result, err := h.findSymbol.Execute(ctx, name, kind)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("find_callers",
		mcp.WithDescription("Find functions that reference the given function name"),
		mcp.WithString("function_name", mcp.Required(), mcp.Description("Function name to find callers for")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fnName := req.GetString("function_name", "")
		result, err := h.findCallers.Execute(ctx, fnName)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("list_api_endpoints",
		mcp.WithDescription("List all detected API endpoints in the indexed repository"),
		mcp.WithString("method", mcp.Description("Filter by HTTP method: GET|POST|PUT|DELETE|PATCH")),
		mcp.WithString("service", mcp.Description("Filter by service/package name")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		method := req.GetString("method", "")
		service := req.GetString("service", "")
		result, err := h.listEndpoints.Execute(ctx, method, service)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("get_file_symbols",
		mcp.WithDescription("Get all symbols defined in a specific file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Relative file path from repo root")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := req.GetString("path", "")
		result, err := h.getFileSymbols.Execute(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("list_services",
		mcp.WithDescription("List all top-level packages/services in the indexed repository"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := h.listServices.Execute(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("get_service_dependencies",
		mcp.WithDescription("Get all packages imported by a service using the Neo4j knowledge graph"),
		mcp.WithString("service", mcp.Required(), mcp.Description("Package/module name of the service")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		service := req.GetString("service", "")
		result, err := h.getServiceDeps.Execute(ctx, service)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("get_api_handlers",
		mcp.WithDescription("Find API handler functions matching a pattern using the Neo4j knowledge graph"),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Pattern to match against handler function names")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pattern := req.GetString("pattern", "")
		result, err := h.getAPIHandlers.Execute(ctx, pattern)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("list_components",
		mcp.WithDescription("List React components, hooks, TypeScript interfaces, and type aliases from indexed JS/TS files"),
		mcp.WithString("kind", mcp.Description("Filter by kind: component|hook|interface|type_alias (default: all)")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 100)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		kind := req.GetString("kind", "")
		limit := req.GetInt("limit", 100)
		result, err := h.listComponents.Execute(ctx, kind, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})
}
