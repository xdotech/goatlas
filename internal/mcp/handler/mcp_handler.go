package handler

import (
	"context"

	"github.com/goatlas/goatlas/internal/mcp/usecase"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// MCPHandler registers all MCP tools with the server.
type MCPHandler struct {
	searchCode         *usecase.SearchCodeUseCase
	readFile           *usecase.ReadFileUseCase
	findSymbol         *usecase.FindSymbolUseCase
	findCallers        *usecase.FindCallersUseCase
	listEndpoints      *usecase.ListEndpointsUseCase
	getFileSymbols     *usecase.GetFileSymbolsUseCase
	listServices       *usecase.ListServicesUseCase
	getServiceDeps     *usecase.GetServiceDepsUseCase
	getAPIHandlers     *usecase.GetAPIHandlersUseCase
	listComponents     *usecase.ListComponentsUseCase
	indexRepo          *usecase.IndexRepoUseCase
	generateEmbeddings *usecase.GenerateEmbeddingsUseCase
	buildGraph         *usecase.BuildGraphUseCase
	getComponentAPIs   *usecase.GetComponentAPIsUseCase
	getAPIConsumers    *usecase.GetAPIConsumersUseCase
	analyzeImpact      *usecase.AnalyzeImpactUseCase
	traceTypeFlow      *usecase.TraceTypeFlowUseCase
	checkStaleness     *usecase.CheckStalenessUseCase
	listProcesses      *usecase.ListProcessesUseCase
	getProcessFlow     *usecase.GetProcessFlowUseCase
	listCommunities    *usecase.ListCommunitiesUseCase
	detectProcesses    *usecase.DetectProcessesUseCase
	listRepos          *usecase.ListReposUseCase
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
	ir *usecase.IndexRepoUseCase,
	ge *usecase.GenerateEmbeddingsUseCase,
	bg *usecase.BuildGraphUseCase,
	gca *usecase.GetComponentAPIsUseCase,
	gac *usecase.GetAPIConsumersUseCase,
	ai *usecase.AnalyzeImpactUseCase,
	ttf *usecase.TraceTypeFlowUseCase,
	cs *usecase.CheckStalenessUseCase,
	lp *usecase.ListProcessesUseCase,
	gpf *usecase.GetProcessFlowUseCase,
	lcomm *usecase.ListCommunitiesUseCase,
	dp *usecase.DetectProcessesUseCase,
	lr *usecase.ListReposUseCase,
) *MCPHandler {
	return &MCPHandler{
		searchCode:         sc,
		readFile:           rf,
		findSymbol:         fs,
		findCallers:        fc,
		listEndpoints:      le,
		getFileSymbols:     gfs,
		listServices:       ls,
		getServiceDeps:     gsd,
		getAPIHandlers:     gah,
		listComponents:     lc,
		indexRepo:          ir,
		generateEmbeddings: ge,
		buildGraph:         bg,
		getComponentAPIs:   gca,
		getAPIConsumers:    gac,
		analyzeImpact:      ai,
		traceTypeFlow:      ttf,
		checkStaleness:     cs,
		listProcesses:      lp,
		getProcessFlow:     gpf,
		listCommunities:    lcomm,
		detectProcesses:    dp,
		listRepos:          lr,
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
		mcp.WithNumber("min_confidence", mcp.Description("Minimum confidence threshold 0.0-1.0 (default: 0.0 = all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fnName := req.GetString("function_name", "")
		_ = req.GetFloat("min_confidence", 0.0) // TODO: pass to graph-based FindCallers when available
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

	// --- Admin tools ---

	srv.AddTool(mcp.NewTool("index_repository",
		mcp.WithDescription("Index or re-index a Go/TypeScript repository. Parses AST to extract symbols (functions, types, methods, interfaces) and API endpoints."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Absolute path to the repository to index")),
		mcp.WithBoolean("force", mcp.Description("Force re-index all files, even if unchanged (default: false)")),
		mcp.WithBoolean("incremental", mcp.Description("Only re-index files changed since the last indexed commit (default: false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path := req.GetString("path", "")
		force := req.GetBool("force", false)
		incremental := req.GetBool("incremental", false)
		result, err := h.indexRepo.Execute(ctx, path, force, incremental)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("generate_embeddings",
		mcp.WithDescription("Generate Gemini vector embeddings for all indexed symbols and store in Qdrant for semantic search. Requires GEMINI_API_KEY."),
		mcp.WithBoolean("force", mcp.Description("Re-embed all symbols, including already-embedded ones (default: false)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		force := req.GetBool("force", false)
		result, err := h.generateEmbeddings.Execute(ctx, force)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("build_graph",
		mcp.WithDescription("Build the Neo4j knowledge graph from indexed data. Creates package, file, function, and type nodes with their relationships."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := h.buildGraph.Execute(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	// --- Phase 1: Frontend ↔ Backend Cross-Reference ---

	srv.AddTool(mcp.NewTool("get_component_apis",
		mcp.WithDescription("Get API endpoints called by a React/TS component. Maps frontend components to backend APIs."),
		mcp.WithString("component", mcp.Required(), mcp.Description("React component name to look up")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		component := req.GetString("component", "")
		result, err := h.getComponentAPIs.Execute(ctx, component)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("get_api_consumers",
		mcp.WithDescription("Find UI components that call a given API endpoint. Reverse lookup from backend API to frontend consumers."),
		mcp.WithString("api_path", mcp.Required(), mcp.Description("API path pattern to search for")),
		mcp.WithString("method", mcp.Description("Filter by HTTP method: GET|POST|PUT|DELETE (optional)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		apiPath := req.GetString("api_path", "")
		method := req.GetString("method", "")
		result, err := h.getAPIConsumers.Execute(ctx, apiPath, method)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	// --- Phase 2: Change Impact Analysis ---

	srv.AddTool(mcp.NewTool("analyze_impact",
		mcp.WithDescription("Analyze change impact: when modifying a function, find all affected callers, API endpoints, and UI components. Uses recursive call graph traversal."),
		mcp.WithString("symbol", mcp.Required(), mcp.Description("Function or method name to analyze impact for")),
		mcp.WithNumber("max_depth", mcp.Description("Maximum call graph depth to traverse (default: 5)")),
		mcp.WithNumber("min_confidence", mcp.Description("Minimum confidence threshold 0.0-1.0 (default: 0.0 = all)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol := req.GetString("symbol", "")
		maxDepth := int(req.GetFloat("max_depth", 5))
		_ = req.GetFloat("min_confidence", 0.0) // TODO: pass through when AnalyzeImpact supports it
		result, err := h.analyzeImpact.Execute(ctx, symbol, maxDepth)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	// --- Phase 3: Type Flow Tracking ---

	srv.AddTool(mcp.NewTool("trace_type_flow",
		mcp.WithDescription("Trace how a data type flows through the codebase: which functions produce (return) and consume (accept) it. Maps the full data flow chain from API handler to repository."),
		mcp.WithString("type_name", mcp.Required(), mcp.Description("Type name to trace (e.g. 'CreateOrderRequest')")),
		mcp.WithString("direction", mcp.Description("Flow direction: upstream (producers) | downstream (consumers) | both (default)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		typeName := req.GetString("type_name", "")
		direction := req.GetString("direction", "both")
		result, err := h.traceTypeFlow.Execute(ctx, typeName, direction)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	// --- Incremental Indexing ---

	srv.AddTool(mcp.NewTool("check_staleness",
		mcp.WithDescription("Check whether a repository's index is up to date with the current git HEAD commit."),
		mcp.WithString("repo", mcp.Description("Repository name to check (optional, defaults to first indexed repo)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		repo := req.GetString("repo", "")
		result, err := h.checkStaleness.Execute(ctx, repo)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	// --- Phase 02: Process & Community Detection ---

	srv.AddTool(mcp.NewTool("list_processes",
		mcp.WithDescription("List all detected execution processes (HTTP handler flows, Kafka consumer chains, etc.)"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := h.listProcesses.Execute(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("get_process_flow",
		mcp.WithDescription("Get the ordered execution flow (call chain) for a named process"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Process name or partial match (from list_processes)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		result, err := h.getProcessFlow.Execute(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("list_communities",
		mcp.WithDescription("List detected code communities (clusters of highly-interconnected functions via Louvain algorithm)"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := h.listCommunities.Execute(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("detect_processes",
		mcp.WithDescription("Trigger process detection (HTTP handlers, Kafka consumers, main()) and community detection (Louvain). Run after index_repository."),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := h.detectProcesses.Execute(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})

	srv.AddTool(mcp.NewTool("list_repos",
		mcp.WithDescription("List all indexed repositories with metadata (name, path, last indexed time, last commit)"),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := h.listRepos.Execute(ctx)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(result), nil
	})
}
