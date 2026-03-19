package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/xdotech/goatlas/internal/agent"
	"github.com/xdotech/goatlas/internal/config"
	"github.com/xdotech/goatlas/internal/db"
	"github.com/xdotech/goatlas/internal/graph"
	"github.com/xdotech/goatlas/internal/indexer"
	mcpusecase "github.com/xdotech/goatlas/internal/mcp/usecase"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat session with an LLM agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.LLMProvider != "ollama" && cfg.GeminiAPIKey == "" {
			return fmt.Errorf("GEMINI_API_KEY not set (or set LLM_PROVIDER=ollama)")
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
			SearchCode:     mcpusecase.NewSearchCodeUseCase(indexerSvc.SymbolRepo, nil, cfg.RepoPath, 0),
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

		provCfg := agent.ProviderConfig{
			Provider:    cfg.LLMProvider,
			GeminiKey:   cfg.GeminiAPIKey,
			OllamaURL:   cfg.OllamaURL,
			OllamaModel: cfg.OllamaModel,
		}
		a, err := agent.NewAgent(ctx, agentCfg, provCfg, bridge, systemPrompt)
		if err != nil {
			return fmt.Errorf("create agent: %w", err)
		}
		defer a.Close()

		fmt.Println("GoAtlas Chat (type 'exit' to quit)")
		fmt.Println("=====================================")

		var history []agent.ConversationMessage
		scanner := bufio.NewScanner(os.Stdin)

		for {
			fmt.Print("\nYou: ")
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}
			if input == "exit" || input == "quit" {
				fmt.Println("Goodbye!")
				break
			}

			answer, newHistory, err := a.Chat(ctx, history, input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			history = newHistory
			fmt.Printf("\nAssistant: %s\n", answer)
		}
		return nil
	},
}
