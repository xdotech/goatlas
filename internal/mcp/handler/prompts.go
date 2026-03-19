package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterPrompts adds all MCP prompts to the server.
// Prompts are reusable prompt templates pre-filled with dynamic codebase context.
func (h *MCPHandler) RegisterPrompts(srv *server.MCPServer, pool *pgxpool.Pool) {

	// ── detect_impact prompt ────────────────────────────────────────────
	srv.AddPrompt(
		mcp.NewPrompt("detect_impact",
			mcp.WithPromptDescription("Analyze the full change impact of modifying a symbol. Pre-fills caller context, affected endpoints, and interface implementations from the indexed graph."),
			mcp.WithArgument("symbol", mcp.ArgumentDescription("Function or type name to analyze"), mcp.RequiredArgument()),
		),
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			symbol := req.Params.Arguments["symbol"]
			if symbol == "" {
				return nil, fmt.Errorf("symbol argument is required")
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Analyze the change impact of modifying `%s`.\n\n", symbol))

			// Gather callers
			callerRows, err := pool.Query(ctx, `
				SELECT DISTINCT caller_qualified_name, callee_name, confidence
				FROM function_calls
				WHERE callee_name ILIKE '%' || $1 || '%'
				ORDER BY confidence DESC
				LIMIT 20`, symbol)
			if err == nil {
				sb.WriteString("## Known Callers (from indexed call graph)\n\n")
				found := false
				for callerRows.Next() {
					var caller, callee string
					var conf float64
					if err := callerRows.Scan(&caller, &callee, &conf); err == nil {
						sb.WriteString(fmt.Sprintf("- `%s` → `%s` (confidence: %.2f)\n", caller, callee, conf))
						found = true
					}
				}
				callerRows.Close()
				if !found {
					sb.WriteString("- (no callers found in index)\n")
				}
				sb.WriteString("\n")
			}

			// Gather affected endpoints
			epRows, err := pool.Query(ctx, `
				SELECT DISTINCT ae.method, ae.path
				FROM api_endpoints ae
				WHERE ae.handler_name ILIKE '%' || $1 || '%'
				LIMIT 10`, symbol)
			if err == nil {
				sb.WriteString("## Affected API Endpoints\n\n")
				found := false
				for epRows.Next() {
					var method, path string
					if err := epRows.Scan(&method, &path); err == nil {
						sb.WriteString(fmt.Sprintf("- %s %s\n", method, path))
						found = true
					}
				}
				epRows.Close()
				if !found {
					sb.WriteString("- (no endpoints directly linked to this symbol)\n")
				}
				sb.WriteString("\n")
			}

			// Gather interface implementations
			implRows, err := pool.Query(ctx, `
				SELECT interface_name, struct_name, method_name, COALESCE(confidence, 0.85)
				FROM interface_impls
				WHERE interface_name ILIKE '%' || $1 || '%'
				   OR struct_name ILIKE '%' || $1 || '%'
				LIMIT 10`, symbol)
			if err == nil {
				sb.WriteString("## Interface Implementations\n\n")
				found := false
				for implRows.Next() {
					var iface, sname, method string
					var conf float64
					if err := implRows.Scan(&iface, &sname, &method, &conf); err == nil {
						sb.WriteString(fmt.Sprintf("- `%s` implements `%s.%s` (conf: %.2f)\n", sname, iface, method, conf))
						found = true
					}
				}
				implRows.Close()
				if !found {
					sb.WriteString("- (no interface implementations found)\n")
				}
				sb.WriteString("\n")
			}

			sb.WriteString("## Instructions\n\n")
			sb.WriteString("Based on the above context, analyze:\n")
			sb.WriteString("1. Which functions will be directly affected by changes to this symbol?\n")
			sb.WriteString("2. Which API endpoints might break?\n")
			sb.WriteString("3. Are there interface contracts that need to be updated?\n")
			sb.WriteString("4. What is the blast radius — how many layers deep does the impact reach?\n")

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Change impact analysis for %s", symbol),
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Type: "text", Text: sb.String()},
					},
				},
			}, nil
		},
	)

	// ── generate_map prompt ─────────────────────────────────────────────
	srv.AddPrompt(
		mcp.NewPrompt("generate_map",
			mcp.WithPromptDescription("Generate a Mermaid architecture diagram from the indexed codebase graph. Produces service-level, package-level, or component-level diagrams."),
			mcp.WithArgument("scope", mcp.ArgumentDescription("Diagram scope: all | services | components (default: all)")),
		),
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			scope := req.Params.Arguments["scope"]
			if scope == "" {
				scope = "all"
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Generate a Mermaid diagram for scope: `%s`.\n\n", scope))

			// Gather package-level data
			pkgRows, err := pool.Query(ctx, `
				SELECT DISTINCT
					CASE WHEN position('/' IN qualified_name) > 0
						THEN substring(qualified_name FROM 1 FOR position('/' IN qualified_name) - 1)
						ELSE split_part(qualified_name, '.', 1)
					END AS pkg,
					COUNT(*) as symbol_count
				FROM symbols
				WHERE kind IN ('func', 'method', 'type', 'interface')
				GROUP BY pkg
				ORDER BY symbol_count DESC
				LIMIT 30`)
			if err == nil {
				sb.WriteString("## Indexed Packages\n\n")
				for pkgRows.Next() {
					var pkg string
					var count int
					if err := pkgRows.Scan(&pkg, &count); err == nil {
						sb.WriteString(fmt.Sprintf("- `%s` (%d symbols)\n", pkg, count))
					}
				}
				pkgRows.Close()
				sb.WriteString("\n")
			}

			// Gather top call relationships
			callRows, err := pool.Query(ctx, `
				SELECT
					split_part(caller_qualified_name, '.', 1) AS caller_pkg,
					COALESCE(callee_package, split_part(callee_name, '.', 1)) AS callee_pkg,
					COUNT(*) AS call_count
				FROM function_calls
				WHERE caller_qualified_name != ''
				GROUP BY caller_pkg, callee_pkg
				HAVING caller_pkg != callee_pkg
				ORDER BY call_count DESC
				LIMIT 20`)
			if err == nil {
				sb.WriteString("## Cross-Package Dependencies (call count)\n\n")
				for callRows.Next() {
					var from, to string
					var count int
					if err := callRows.Scan(&from, &to, &count); err == nil {
						sb.WriteString(fmt.Sprintf("- `%s` → `%s` (%d calls)\n", from, to, count))
					}
				}
				callRows.Close()
				sb.WriteString("\n")
			}

			sb.WriteString("## Instructions\n\n")
			sb.WriteString("Based on the packages and dependencies above, generate a **Mermaid** diagram:\n\n")
			sb.WriteString("1. Use `graph TD` (top-down) layout\n")
			sb.WriteString("2. Group related packages into subgraphs\n")
			sb.WriteString("3. Label edges with call counts\n")
			sb.WriteString("4. Highlight API/handler packages differently\n")
			sb.WriteString("5. Output ONLY the Mermaid code block, no explanation\n")

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Architecture diagram generation (scope: %s)", scope),
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Type: "text", Text: sb.String()},
					},
				},
			}, nil
		},
	)

	// ── explain_community prompt ────────────────────────────────────────
	srv.AddPrompt(
		mcp.NewPrompt("explain_community",
			mcp.WithPromptDescription("Explain what a detected code community cluster does, based on its member functions and their call relationships."),
			mcp.WithArgument("community_name", mcp.ArgumentDescription("Community name from list_communities"), mcp.RequiredArgument()),
		),
		func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			communityName := req.Params.Arguments["community_name"]
			if communityName == "" {
				return nil, fmt.Errorf("community_name argument is required")
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Explain what the code community `%s` does.\n\n", communityName))

			// Get community members
			memberRows, err := pool.Query(ctx, `
				SELECT cm.symbol_name
				FROM community_members cm
				JOIN communities c ON cm.community_id = c.id
				WHERE c.name ILIKE '%' || $1 || '%'
				ORDER BY cm.symbol_name
				LIMIT 50`, communityName)
			if err == nil {
				sb.WriteString("## Community Members\n\n")
				found := false
				for memberRows.Next() {
					var name string
					if err := memberRows.Scan(&name); err == nil {
						sb.WriteString(fmt.Sprintf("- `%s`\n", name))
						found = true
					}
				}
				memberRows.Close()
				if !found {
					sb.WriteString("- (no members found — community may not exist or Phase 02 not run)\n")
				}
				sb.WriteString("\n")
			}

			sb.WriteString("## Instructions\n\n")
			sb.WriteString("Based on the function/method names in this community cluster:\n")
			sb.WriteString("1. What is the primary responsibility of this community?\n")
			sb.WriteString("2. Is it a domain layer, infrastructure layer, or cross-cutting concern?\n")
			sb.WriteString("3. Name the key functions and explain their roles\n")
			sb.WriteString("4. Suggest a descriptive label for this community\n")

			return &mcp.GetPromptResult{
				Description: fmt.Sprintf("Community explanation for %s", communityName),
				Messages: []mcp.PromptMessage{
					{
						Role:    mcp.RoleUser,
						Content: mcp.TextContent{Type: "text", Text: sb.String()},
					},
				},
			}, nil
		},
	)
}
