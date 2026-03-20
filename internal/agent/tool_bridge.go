package agent

import (
	"context"
	"fmt"

	"github.com/xdotech/goatlas/internal/mcp/usecase"
)

// ToolHandler executes a tool and returns a string result.
type ToolHandler func(ctx context.Context, args map[string]interface{}) (string, error)

// ToolBridge holds all tool handlers for direct invocation.
type ToolBridge struct {
	handlers map[string]ToolHandler
}

// UseCases groups all MCP use cases needed by the bridge.
type UseCases struct {
	SearchCode     *usecase.SearchCodeUseCase
	ReadFile       *usecase.ReadFileUseCase
	FindSymbol     *usecase.FindSymbolUseCase
	FindCallers    *usecase.FindCallersUseCase
	ListEndpoints  *usecase.ListEndpointsUseCase
	GetFileSymbols *usecase.GetFileSymbolsUseCase
	ListServices   *usecase.ListServicesUseCase
	GetServiceDeps *usecase.GetServiceDepsUseCase
	GetAPIHandlers *usecase.GetAPIHandlersUseCase
}

// NewToolBridge wires all use cases into a callable tool map.
func NewToolBridge(uc *UseCases) *ToolBridge {
	b := &ToolBridge{handlers: make(map[string]ToolHandler)}

	b.handlers["search_code"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.SearchCode.Execute(ctx,
			getString(args, "query", ""),
			getInt(args, "limit", 20),
			getString(args, "kind", ""),
			getString(args, "mode", "keyword"),
		)
	}

	b.handlers["read_file"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.ReadFile.Execute(ctx,
			getString(args, "path", ""),
			getInt(args, "start_line", 0),
			getInt(args, "end_line", 0),
		)
	}

	b.handlers["find_symbol"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.FindSymbol.Execute(ctx,
			getString(args, "name", ""),
			getString(args, "kind", ""),
		)
	}

	b.handlers["find_callers"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.FindCallers.Execute(ctx, getString(args, "function_name", ""), getInt(args, "depth", 5), getString(args, "repo", ""))
	}

	b.handlers["list_api_endpoints"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.ListEndpoints.Execute(ctx,
			getString(args, "method", ""),
			getString(args, "service", ""),
		)
	}

	b.handlers["get_file_symbols"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.GetFileSymbols.Execute(ctx, getString(args, "path", ""))
	}

	b.handlers["list_services"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.ListServices.Execute(ctx)
	}

	b.handlers["get_service_dependencies"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.GetServiceDeps.Execute(ctx, getString(args, "service", ""))
	}

	b.handlers["get_api_handlers"] = func(ctx context.Context, args map[string]interface{}) (string, error) {
		return uc.GetAPIHandlers.Execute(ctx, getString(args, "endpoint_pattern", ""))
	}

	return b
}

// Execute runs the named tool with the given arguments.
func (b *ToolBridge) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	h, ok := b.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	result, err := h(ctx, args)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if len(result) > 10000 {
		result = result[:10000] + "\n[... truncated at 10KB ...]"
	}
	return result, nil
}

// ToolNames returns the list of registered tool names.
func (b *ToolBridge) ToolNames() []string {
	names := make([]string, 0, len(b.handlers))
	for name := range b.handlers {
		names = append(names, name)
	}
	return names
}

func getString(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getInt(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}
