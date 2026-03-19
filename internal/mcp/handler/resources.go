package handler

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterResources adds all MCP resources to the server.
// Resources expose structured, on-demand JSON data that AI can pull without calling a tool.
func (h *MCPHandler) RegisterResources(srv *server.MCPServer, pool *pgxpool.Pool) {

	// ── goatlas://repositories ──────────────────────────────────────────
	srv.AddResource(
		mcp.NewResource("goatlas://repositories",
			"Indexed repositories",
			mcp.WithResourceDescription("List of all indexed repositories with metadata"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			type repoInfo struct {
				ID            int64  `json:"id"`
				Name          string `json:"name"`
				Path          string `json:"path"`
				LastIndexedAt string `json:"last_indexed_at"`
				LastCommit    string `json:"last_commit"`
			}
			rows, err := pool.Query(ctx, `
				SELECT id, name, path,
				       COALESCE(to_char(last_indexed_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
				       COALESCE(last_commit, '')
				FROM repositories ORDER BY name`)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			var repos []repoInfo
			for rows.Next() {
				var r repoInfo
				if err := rows.Scan(&r.ID, &r.Name, &r.Path, &r.LastIndexedAt, &r.LastCommit); err != nil {
					return nil, err
				}
				repos = append(repos, r)
			}
			if err := rows.Err(); err != nil {
				return nil, err
			}
			if repos == nil {
				repos = []repoInfo{}
			}

			data, _ := json.MarshalIndent(repos, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{URI: "goatlas://repositories", MIMEType: "application/json", Text: string(data)},
			}, nil
		},
	)

	// ── goatlas://endpoints ─────────────────────────────────────────────
	srv.AddResource(
		mcp.NewResource("goatlas://endpoints",
			"API endpoints",
			mcp.WithResourceDescription("All detected API endpoints as structured JSON"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			type endpointInfo struct {
				Method  string `json:"method"`
				Path    string `json:"path"`
				Handler string `json:"handler"`
				File    string `json:"file"`
				Line    int    `json:"line"`
			}
			rows, err := pool.Query(ctx, `
				SELECT COALESCE(ae.method,''), COALESCE(ae.path,''), COALESCE(ae.handler_name,''),
				       COALESCE(f.path,''), COALESCE(ae.line, 0)
				FROM api_endpoints ae
				JOIN files f ON f.id = ae.file_id
				ORDER BY ae.path, ae.method LIMIT 500`)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			var endpoints []endpointInfo
			for rows.Next() {
				var e endpointInfo
				if err := rows.Scan(&e.Method, &e.Path, &e.Handler, &e.File, &e.Line); err != nil {
					return nil, err
				}
				endpoints = append(endpoints, e)
			}
			if err := rows.Err(); err != nil {
				return nil, err
			}
			if endpoints == nil {
				endpoints = []endpointInfo{}
			}

			data, _ := json.MarshalIndent(endpoints, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{URI: "goatlas://endpoints", MIMEType: "application/json", Text: string(data)},
			}, nil
		},
	)

	// ── goatlas://schema ────────────────────────────────────────────────
	srv.AddResource(
		mcp.NewResource("goatlas://schema",
			"Database schema",
			mcp.WithResourceDescription("GoAtlas PostgreSQL schema summary with table relationships and example queries"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			schema := map[string]any{
				"tables": []map[string]any{
					{"name": "repositories", "key_columns": []string{"id", "name", "path", "last_indexed_at", "last_commit"}, "description": "Indexed repository roots"},
					{"name": "files", "key_columns": []string{"id", "repo_id", "path", "content_hash"}, "description": "Source files indexed from repositories"},
					{"name": "symbols", "key_columns": []string{"id", "file_id", "name", "kind", "qualified_name"}, "description": "Functions, types, interfaces, methods, vars, consts"},
					{"name": "function_calls", "key_columns": []string{"id", "file_id", "caller_qualified_name", "callee_name", "confidence"}, "description": "Call graph edges between functions"},
					{"name": "interface_impls", "key_columns": []string{"id", "file_id", "interface_name", "struct_name", "method_name", "confidence"}, "description": "Interface implementation relationships"},
					{"name": "api_endpoints", "key_columns": []string{"id", "file_id", "method", "path", "handler_name"}, "description": "Detected REST/RPC API endpoints"},
					{"name": "type_usages", "key_columns": []string{"id", "file_id", "type_name", "symbol_name", "direction"}, "description": "Type input/output flow tracking"},
					{"name": "component_api_calls", "key_columns": []string{"id", "file_id", "component_name", "api_path"}, "description": "Frontend component to API mapping"},
					{"name": "processes", "key_columns": []string{"id", "repo_id", "name", "entry_point"}, "description": "Detected execution flows (Phase 02)"},
					{"name": "process_steps", "key_columns": []string{"id", "process_id", "step_order", "function_name"}, "description": "Ordered steps in execution flows"},
					{"name": "communities", "key_columns": []string{"id", "repo_id", "name", "modularity"}, "description": "Detected code community clusters (Louvain)"},
					{"name": "community_members", "key_columns": []string{"id", "community_id", "symbol_name"}, "description": "Functions belonging to a community"},
				},
				"relationships": []string{
					"files.repo_id → repositories.id",
					"symbols.file_id → files.id",
					"function_calls.file_id → files.id",
					"interface_impls.file_id → files.id",
					"api_endpoints.file_id → files.id",
					"process_steps.process_id → processes.id",
					"community_members.community_id → communities.id",
				},
				"example_queries": []map[string]string{
					{"description": "Find all functions in a file", "sql": "SELECT name, kind, line FROM symbols WHERE file_id = $1 AND kind IN ('func','method')"},
					{"description": "Find high-confidence callers", "sql": "SELECT caller_qualified_name, callee_name, confidence FROM function_calls WHERE confidence >= 0.8"},
					{"description": "List entry points for processes", "sql": "SELECT name, entry_point, step_count FROM processes WHERE repo_id = $1"},
				},
			}
			data, _ := json.MarshalIndent(schema, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{URI: "goatlas://schema", MIMEType: "application/json", Text: string(data)},
			}, nil
		},
	)

	// ── goatlas://communities ───────────────────────────────────────────
	srv.AddResource(
		mcp.NewResource("goatlas://communities",
			"Code communities",
			mcp.WithResourceDescription("Detected code community clusters from Louvain analysis (empty if Phase 02 not run)"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			type communityInfo struct {
				ID          int64   `json:"id"`
				Name        string  `json:"name"`
				Modularity  float64 `json:"modularity"`
				MemberCount int     `json:"member_count"`
			}
			rows, err := pool.Query(ctx, `
				SELECT c.id, c.name, COALESCE(c.modularity, 0),
				       (SELECT COUNT(*) FROM community_members cm WHERE cm.community_id = c.id)
				FROM communities c ORDER BY c.id LIMIT 100`)
			if err != nil {
				// Table might not exist yet — return empty
				return []mcp.ResourceContents{
					mcp.TextResourceContents{URI: "goatlas://communities", MIMEType: "application/json", Text: "[]"},
				}, nil
			}
			defer rows.Close()

			var communities []communityInfo
			for rows.Next() {
				var c communityInfo
				if err := rows.Scan(&c.ID, &c.Name, &c.Modularity, &c.MemberCount); err != nil {
					continue
				}
				communities = append(communities, c)
			}
			if communities == nil {
				communities = []communityInfo{}
			}

			data, _ := json.MarshalIndent(communities, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{URI: "goatlas://communities", MIMEType: "application/json", Text: string(data)},
			}, nil
		},
	)

	// ── goatlas://processes ─────────────────────────────────────────────
	srv.AddResource(
		mcp.NewResource("goatlas://processes",
			"Execution processes",
			mcp.WithResourceDescription("Detected execution flows and their entry points (empty if Phase 02 not run)"),
			mcp.WithMIMEType("application/json"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			type processInfo struct {
				ID         int64  `json:"id"`
				Name       string `json:"name"`
				EntryPoint string `json:"entry_point"`
				StepCount  int    `json:"step_count"`
			}
			rows, err := pool.Query(ctx, `
				SELECT p.id, p.name, COALESCE(p.entry_point, ''),
				       (SELECT COUNT(*) FROM process_steps ps WHERE ps.process_id = p.id)
				FROM processes p ORDER BY p.id LIMIT 100`)
			if err != nil {
				return []mcp.ResourceContents{
					mcp.TextResourceContents{URI: "goatlas://processes", MIMEType: "application/json", Text: "[]"},
				}, nil
			}
			defer rows.Close()

			var processes []processInfo
			for rows.Next() {
				var p processInfo
				if err := rows.Scan(&p.ID, &p.Name, &p.EntryPoint, &p.StepCount); err != nil {
					continue
				}
				processes = append(processes, p)
			}
			if processes == nil {
				processes = []processInfo{}
			}

			data, _ := json.MarshalIndent(processes, "", "  ")
			return []mcp.ResourceContents{
				mcp.TextResourceContents{URI: "goatlas://processes", MIMEType: "application/json", Text: string(data)},
			}, nil
		},
	)

}
