package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/goatlas/goatlas/internal/agent"
	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/db"
	"github.com/goatlas/goatlas/internal/graph"
	"github.com/goatlas/goatlas/internal/indexer"
	mcpusecase "github.com/goatlas/goatlas/internal/mcp/usecase"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask a question about the codebase using Gemini AI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		question := args[0]
		ctx := context.Background()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.GeminiAPIKey == "" {
			return fmt.Errorf("GEMINI_API_KEY not set")
		}

		pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer pool.Close()

		var querier *graph.Querier
		if cfg.Neo4jURL != "" {
			gc, gcErr := graph.NewClient(ctx, cfg.Neo4jURL, cfg.Neo4jUser, cfg.Neo4jPass)
			if gcErr == nil {
				defer gc.Close(ctx)
				querier = graph.NewQuerier(gc)
			}
		}

		indexerSvc := indexer.NewService(pool)

		uc := &agent.UseCases{
			SearchCode:     mcpusecase.NewSearchCodeUseCase(indexerSvc.SymbolRepo, nil, cfg.RepoPath),
			ReadFile:       mcpusecase.NewReadFileUseCase(pool),
			FindSymbol:     mcpusecase.NewFindSymbolUseCase(indexerSvc.SymbolRepo, cfg.RepoPath),
			FindCallers:    mcpusecase.NewFindCallersUseCase(indexerSvc.SymbolRepo, indexerSvc.IIRepo),
			ListEndpoints:  mcpusecase.NewListEndpointsUseCase(indexerSvc.EndpointRepo),
			GetFileSymbols: mcpusecase.NewGetFileSymbolsUseCase(pool, indexerSvc.SymbolRepo),
			ListServices:   mcpusecase.NewListServicesUseCase(pool),
			GetServiceDeps: mcpusecase.NewGetServiceDepsUseCase(querier),
			GetAPIHandlers: mcpusecase.NewGetAPIHandlersUseCase(querier),
		}

		bridge := agent.NewToolBridge(uc)
		systemPrompt := agent.BuildSystemPrompt(ctx, pool, cfg.RepoPath, bridge.ToolNames())

		agentCfg := agent.DefaultConfig()
		if cfg.RepoPath != "" {
			agentCfg.RepoName = cfg.RepoPath
		}

		a, err := agent.NewAgent(ctx, agentCfg, cfg.GeminiAPIKey, bridge, systemPrompt)
		if err != nil {
			return fmt.Errorf("create agent: %w", err)
		}
		defer a.Close()

		fmt.Fprintf(os.Stderr, "Asking: %s\n\n", question)
		answer, err := a.Ask(ctx, question)
		if err != nil {
			return err
		}
		fmt.Println(answer)
		return nil
	},
}
