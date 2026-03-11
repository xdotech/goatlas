package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BuildSystemPrompt creates the context starvation system prompt.
func BuildSystemPrompt(ctx context.Context, pool *pgxpool.Pool, repoName string, toolNames []string) string {
	services := fetchServices(ctx, pool)

	var sb strings.Builder
	fmt.Fprintf(&sb, "You are a code intelligence assistant for the %s codebase.\n\n", repoName)
	sb.WriteString("IMPORTANT: Do NOT hallucinate APIs, functions, or code that you haven't verified.\n")
	sb.WriteString("ALWAYS use tools to verify before answering. Never guess implementation details.\n\n")

	if len(services) > 0 {
		sb.WriteString("Available services/packages:\n")
		for _, s := range services {
			fmt.Fprintf(&sb, "  - %s\n", s)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Available tools:\n")
	for _, t := range toolNames {
		fmt.Fprintf(&sb, "  - %s\n", t)
	}
	sb.WriteString("\nTool usage rules:\n")
	sb.WriteString("1. Use search_code to find relevant functions before answering\n")
	sb.WriteString("2. Use read_file to read actual implementation details\n")
	sb.WriteString("3. Use get_service_dependencies to understand service relationships\n")
	sb.WriteString("4. Use list_api_endpoints to find actual API routes\n")
	sb.WriteString("5. Only mention code that you've verified through tool calls\n")

	return sb.String()
}

func fetchServices(ctx context.Context, pool *pgxpool.Pool) []string {
	if pool == nil {
		return nil
	}
	rows, err := pool.Query(ctx, `SELECT DISTINCT module FROM files WHERE module != '' ORDER BY module LIMIT 20`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var services []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			services = append(services, s)
		}
	}
	return services
}
