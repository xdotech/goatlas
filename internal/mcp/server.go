package mcp

import (
	"github.com/xdotech/goatlas/internal/config"
	"github.com/xdotech/goatlas/internal/graph"
	"github.com/xdotech/goatlas/internal/indexer"
	"github.com/xdotech/goatlas/internal/mcp/handler"
	"github.com/xdotech/goatlas/internal/mcp/registry"
	"github.com/xdotech/goatlas/internal/mcp/usecase"
	"github.com/xdotech/goatlas/internal/vector"
	"github.com/jackc/pgx/v5/pgxpool"
	mcpgo "github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server and exposes transport methods.
type Server struct {
	mcpServer *mcpgo.MCPServer
}

// ServerConfig holds all dependencies for the MCP server.
type ServerConfig struct {
	Config      *config.Config
	RepoRoot    string
	IndexerSvc  *indexer.Service
	Pool        *pgxpool.Pool
	Searcher    *vector.Searcher // nil if vector search disabled
	GraphClient *graph.Client    // nil if graph disabled
}

// NewServer wires all use cases and registers tools with the MCP server.
func NewServer(cfg ServerConfig) *Server {
	mcpSrv := mcpgo.NewMCPServer("goatlas", "1.0.0")

	var querier *graph.Querier
	if cfg.GraphClient != nil {
		querier = graph.NewQuerier(cfg.GraphClient)
	}

	repoReg := registry.NewRepoRegistry(cfg.Pool)

	h := handler.NewMCPHandler(
		usecase.NewSearchCodeUseCase(cfg.IndexerSvc.SymbolRepo, cfg.Searcher, cfg.RepoRoot, cfg.Config.RRFK),
		usecase.NewReadFileUseCase(cfg.Pool),
		usecase.NewFindSymbolUseCase(cfg.IndexerSvc.SymbolRepo, cfg.RepoRoot),
		usecase.NewFindCallersUseCase(cfg.IndexerSvc.SymbolRepo, cfg.IndexerSvc.IIRepo),
		usecase.NewListEndpointsUseCase(cfg.IndexerSvc.EndpointRepo),
		usecase.NewGetFileSymbolsUseCase(cfg.Pool, cfg.IndexerSvc.SymbolRepo),
		usecase.NewListServicesUseCase(cfg.Pool),
		usecase.NewGetServiceDepsUseCase(querier),
		usecase.NewGetAPIHandlersUseCase(querier),
		usecase.NewListComponentsUseCase(cfg.IndexerSvc.SymbolRepo),
		usecase.NewIndexRepoUseCase(cfg.IndexerSvc),
		usecase.NewGenerateEmbeddingsUseCase(cfg.Pool, cfg.Config.QdrantURL, vector.EmbedConfig{
			Provider:    cfg.Config.EmbedProvider,
			GeminiKey:   cfg.Config.GeminiAPIKey,
			OllamaURL:   cfg.Config.OllamaURL,
			OllamaModel: cfg.Config.OllamaEmbedModel,
		}),
		usecase.NewBuildGraphUseCase(cfg.GraphClient, cfg.Pool),
		usecase.NewGetComponentAPIsUseCase(cfg.IndexerSvc.CACRepo),
		usecase.NewGetAPIConsumersUseCase(cfg.IndexerSvc.CACRepo),
		usecase.NewAnalyzeImpactUseCase(querier),
		usecase.NewTraceTypeFlowUseCase(cfg.IndexerSvc.TURepo, querier),
		usecase.NewCheckStalenessUseCase(cfg.IndexerSvc.RepoRepo),
		usecase.NewListProcessesUseCase(cfg.IndexerSvc.ProcessRepo, cfg.Pool),
		usecase.NewGetProcessFlowUseCase(cfg.IndexerSvc.ProcessRepo, cfg.Pool),
		usecase.NewListCommunitiesUseCase(cfg.IndexerSvc.CommunityRepo, cfg.Pool),
		usecase.NewDetectProcessesUseCase(cfg.Pool, cfg.GraphClient, cfg.IndexerSvc.ProcessRepo, cfg.IndexerSvc.CommunityRepo),
		usecase.NewListReposUseCase(repoReg),
	)
	h.RegisterTools(mcpSrv)
	h.RegisterResources(mcpSrv, cfg.Pool)
	h.RegisterPrompts(mcpSrv, cfg.Pool)

	return &Server{mcpServer: mcpSrv}
}

// RunStdio starts the MCP server using stdio transport (blocking).
func (s *Server) RunStdio() error {
	return mcpgo.ServeStdio(s.mcpServer)
}
