package mcp

import (
	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/graph"
	"github.com/goatlas/goatlas/internal/indexer"
	"github.com/goatlas/goatlas/internal/mcp/handler"
	"github.com/goatlas/goatlas/internal/mcp/usecase"
	"github.com/goatlas/goatlas/internal/vector"
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

	h := handler.NewMCPHandler(
		usecase.NewSearchCodeUseCase(cfg.IndexerSvc.SymbolRepo, cfg.Searcher),
		usecase.NewReadFileUseCase(cfg.RepoRoot),
		usecase.NewFindSymbolUseCase(cfg.IndexerSvc.SymbolRepo),
		usecase.NewFindCallersUseCase(cfg.IndexerSvc.SymbolRepo),
		usecase.NewListEndpointsUseCase(cfg.IndexerSvc.EndpointRepo),
		usecase.NewGetFileSymbolsUseCase(cfg.IndexerSvc.FileRepo, cfg.IndexerSvc.SymbolRepo),
		usecase.NewListServicesUseCase(cfg.Pool),
		usecase.NewGetServiceDepsUseCase(querier),
		usecase.NewGetAPIHandlersUseCase(querier),
		usecase.NewListComponentsUseCase(cfg.IndexerSvc.SymbolRepo),
		usecase.NewIndexRepoUseCase(cfg.IndexerSvc),
		usecase.NewGenerateEmbeddingsUseCase(cfg.Pool, cfg.Config.QdrantURL, cfg.Config.GeminiAPIKey),
		usecase.NewBuildGraphUseCase(cfg.GraphClient, cfg.Pool),
	)
	h.RegisterTools(mcpSrv)

	return &Server{mcpServer: mcpSrv}
}

// RunStdio starts the MCP server using stdio transport (blocking).
func (s *Server) RunStdio() error {
	return mcpgo.ServeStdio(s.mcpServer)
}

