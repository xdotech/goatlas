package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Builder reads the PostgreSQL index and populates the Neo4j knowledge graph.
type Builder struct {
	pool   *pgxpool.Pool
	client *Client
}

// NewBuilder constructs a Builder.
func NewBuilder(pool *pgxpool.Pool, client *Client) *Builder {
	return &Builder{pool: pool, client: client}
}

// BuildResult holds counts of nodes and edges created.
type BuildResult struct {
	PackageNodes    int
	FileNodes       int
	FunctionNodes   int
	TypeNodes       int
	ImportEdges     int
	ServiceNodes    int
	ConnectionEdges int
	ComponentEdges  int
	CallEdges       int
	TypeFlowEdges   int
	ImplementsEdges int
}

// batchSize controls how many rows are sent per UNWIND transaction.
// Kept small to avoid CPU spikes on the Neo4j container.
const batchSize = 100

// batchDelay is the pause between batches to let Neo4j breathe.
const batchDelay = 50 * time.Millisecond

// runBatch executes cypher with UNWIND $rows in chunks to avoid large transactions.
func runBatch(ctx context.Context, client *Client, cypher string, rows []map[string]any) error {
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if err := client.RunCypher(ctx, cypher, map[string]any{"rows": rows[i:end]}); err != nil {
			return err
		}
		if end < len(rows) {
			time.Sleep(batchDelay)
		}
	}
	return nil
}

// Build populates the Neo4j graph from the PostgreSQL index.
func (b *Builder) Build(ctx context.Context) (*BuildResult, error) {
	result := &BuildResult{}

	if err := b.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("create indexes: %w", err)
	}
	if err := b.buildFileNodes(ctx, result); err != nil {
		return nil, fmt.Errorf("build file nodes: %w", err)
	}
	if err := b.buildSymbolNodes(ctx, result); err != nil {
		return nil, fmt.Errorf("build symbol nodes: %w", err)
	}
	if err := b.buildImportEdges(ctx, result); err != nil {
		return nil, fmt.Errorf("build import edges: %w", err)
	}
	if err := b.buildServiceConnections(ctx, result); err != nil {
		return nil, fmt.Errorf("build service connections: %w", err)
	}
	if err := b.buildComponentAPIEdges(ctx, result); err != nil {
		return nil, fmt.Errorf("build component API edges: %w", err)
	}
	if err := b.buildCallEdges(ctx, result); err != nil {
		return nil, fmt.Errorf("build call edges: %w", err)
	}
	if err := b.buildTypeFlowEdges(ctx, result); err != nil {
		return nil, fmt.Errorf("build type flow edges: %w", err)
	}
	if err := b.buildImplementsEdges(ctx, result); err != nil {
		return nil, fmt.Errorf("build implements edges: %w", err)
	}

	return result, nil
}

func (b *Builder) createIndexes(ctx context.Context) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS package_name FOR (p:Package) ON (p.name)`,
		`CREATE INDEX IF NOT EXISTS file_path FOR (f:File) ON (f.path)`,
		`CREATE INDEX IF NOT EXISTS function_qualified FOR (fn:Function) ON (fn.qualified)`,
		`CREATE INDEX IF NOT EXISTS function_name FOR (fn:Function) ON (fn.name)`,
		`CREATE INDEX IF NOT EXISTS type_qualified FOR (t:Type) ON (t.qualified)`,
		`CREATE INDEX IF NOT EXISTS type_name FOR (t:Type) ON (t.name)`,
		`CREATE INDEX IF NOT EXISTS service_name FOR (s:Service) ON (s.name)`,
		`CREATE INDEX IF NOT EXISTS topic_name FOR (tp:Topic) ON (tp.name)`,
		`CREATE INDEX IF NOT EXISTS endpoint_path FOR (ep:Endpoint) ON (ep.path)`,
	}
	for _, idx := range indexes {
		// Non-fatal: older Neo4j versions may use different syntax.
		_ = b.client.RunCypher(ctx, idx, nil)
	}
	return nil
}

func (b *Builder) buildFileNodes(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `SELECT path, module FROM files`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch []map[string]any
	pkgs := map[string]struct{}{}
	for rows.Next() {
		var path, module string
		if err := rows.Scan(&path, &module); err != nil {
			return err
		}
		batch = append(batch, map[string]any{"path": path, "module": module})
		pkgs[module] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.FileNodes = len(batch)
	result.PackageNodes = len(pkgs)
	if len(batch) == 0 {
		return nil
	}

	return runBatch(ctx, b.client, `
		UNWIND $rows AS row
		MERGE (pkg:Package {name: row.module})
		MERGE (f:File {path: row.path})
		SET f.package = row.module
		MERGE (pkg)-[:DEFINES]->(f)
	`, batch)
}

func (b *Builder) buildSymbolNodes(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT s.kind, s.name, s.qualified_name, s.signature, s.line, f.path
		FROM symbols s JOIN files f ON s.file_id = f.id
		WHERE s.kind IN ('func', 'method', 'struct', 'interface', 'type')
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var funcs, types []map[string]any
	for rows.Next() {
		var kind, name, qualifiedName, signature, filePath string
		var line int
		if err := rows.Scan(&kind, &name, &qualifiedName, &signature, &line, &filePath); err != nil {
			return err
		}
		switch kind {
		case "func", "method":
			funcs = append(funcs, map[string]any{
				"qualified": qualifiedName, "name": name,
				"signature": signature, "line": line, "file": filePath,
			})
		default:
			types = append(types, map[string]any{
				"qualified": qualifiedName, "name": name,
				"kind": kind, "line": line, "file": filePath,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.FunctionNodes = len(funcs)
	result.TypeNodes = len(types)

	if len(funcs) > 0 {
		if err := runBatch(ctx, b.client, `
			UNWIND $rows AS row
			MERGE (fn:Function {qualified: row.qualified})
			SET fn.name = row.name, fn.signature = row.signature, fn.line = row.line
			MERGE (f:File {path: row.file})
			MERGE (f)-[:DEFINES]->(fn)
		`, funcs); err != nil {
			return err
		}
	}
	if len(types) > 0 {
		if err := runBatch(ctx, b.client, `
			UNWIND $rows AS row
			MERGE (t:Type {qualified: row.qualified})
			SET t.name = row.name, t.kind = row.kind, t.line = row.line
			MERGE (f:File {path: row.file})
			MERGE (f)-[:DEFINES]->(t)
		`, types); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) buildImportEdges(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT f.path, i.import_path
		FROM imports i JOIN files f ON i.file_id = f.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch []map[string]any
	for rows.Next() {
		var fromPath, importPath string
		if err := rows.Scan(&fromPath, &importPath); err != nil {
			return err
		}
		batch = append(batch, map[string]any{"from_path": fromPath, "import_path": importPath})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.ImportEdges = len(batch)
	if len(batch) == 0 {
		return nil
	}

	return runBatch(ctx, b.client, `
		UNWIND $rows AS row
		MERGE (importer:File {path: row.from_path})
		MERGE (imported:Package {name: row.import_path})
		MERGE (importer)-[:IMPORTS]->(imported)
	`, batch)
}

func (b *Builder) buildServiceConnections(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT r.name, sc.conn_type, sc.target, sc.line
		FROM service_connections sc
		JOIN repositories r ON sc.repo_id = r.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type connRow struct {
		source, connType, target string
		line                     int
	}
	var all []connRow
	services := map[string]struct{}{}
	for rows.Next() {
		var r connRow
		if err := rows.Scan(&r.source, &r.connType, &r.target, &r.line); err != nil {
			return err
		}
		all = append(all, r)
		services[r.source] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.ServiceNodes = len(services)
	result.ConnectionEdges = len(all)
	if len(all) == 0 {
		return nil
	}

	batch := make([]map[string]any, 0, len(all))
	for _, r := range all {
		batch = append(batch, map[string]any{
			"source":    r.source,
			"target":    r.target,
			"conn_type": r.connType,
			"line":      r.line,
		})
	}

	return runBatch(ctx, b.client, `
		UNWIND $rows AS row
		MERGE (src:Service {name: row.source})
		MERGE (tgt:Service {name: row.target})
		MERGE (src)-[r:CONNECTS]->(tgt)
		SET r.conn_type = row.conn_type, r.line = row.line
	`, batch)
}

func (b *Builder) buildComponentAPIEdges(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT c.component, c.http_method, c.api_path, c.target_service, f.path
		FROM component_api_calls c
		JOIN files f ON c.file_id = f.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch []map[string]any
	for rows.Next() {
		var component, method, apiPath, targetService, filePath string
		if err := rows.Scan(&component, &method, &apiPath, &targetService, &filePath); err != nil {
			return err
		}
		batch = append(batch, map[string]any{
			"component": component, "file": filePath,
			"api_path": apiPath, "method": method, "service": targetService,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.ComponentEdges = len(batch)
	if len(batch) == 0 {
		return nil
	}

	return runBatch(ctx, b.client, `
		UNWIND $rows AS row
		MERGE (comp:Component {name: row.component, file: row.file})
		MERGE (ep:Endpoint {path: row.api_path})
		MERGE (comp)-[:CALLS_API {method: row.method, service: row.service}]->(ep)
	`, batch)
}

func (b *Builder) buildCallEdges(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT fc.caller_qualified_name, fc.callee_name, fc.callee_package, fc.line,
		       COALESCE(fc.confidence, 0.5)
		FROM function_calls fc
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch []map[string]any
	for rows.Next() {
		var callerQName, calleeName string
		var calleeQualified *string
		var line int
		var confidence float64
		if err := rows.Scan(&callerQName, &calleeName, &calleeQualified, &line, &confidence); err != nil {
			return err
		}
		cq := ""
		if calleeQualified != nil {
			cq = *calleeQualified
		}
		batch = append(batch, map[string]any{
			"caller": callerQName, "callee_qualified": cq,
			"callee_name": calleeName, "line": line, "conf": confidence,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.CallEdges = len(batch)
	if len(batch) == 0 {
		return nil
	}

	return runBatch(ctx, b.client, `
		UNWIND $rows AS row
		MATCH (caller:Function {qualified: row.caller})
		MATCH (callee:Function)
		WHERE callee.qualified = row.callee_qualified OR callee.name = row.callee_name
		MERGE (caller)-[:CALLS {line: row.line, confidence: row.conf}]->(callee)
	`, batch)
}

func (b *Builder) buildTypeFlowEdges(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT tu.symbol_name, tu.type_name, tu.direction, tu.position
		FROM type_usages tu
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var inputs, outputs []map[string]any
	for rows.Next() {
		var symbolName, typeName, direction string
		var position int
		if err := rows.Scan(&symbolName, &typeName, &direction, &position); err != nil {
			return err
		}
		row := map[string]any{"symbol": symbolName, "type_name": typeName, "pos": position}
		switch direction {
		case "input":
			inputs = append(inputs, row)
		case "output":
			outputs = append(outputs, row)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.TypeFlowEdges = len(inputs) + len(outputs)

	if len(inputs) > 0 {
		if err := runBatch(ctx, b.client, `
			UNWIND $rows AS row
			MATCH (fn:Function)
			WHERE fn.qualified = row.symbol OR fn.name = row.symbol
			MERGE (t:Type {name: row.type_name})
			MERGE (fn)-[:ACCEPTS {pos: row.pos}]->(t)
		`, inputs); err != nil {
			return err
		}
	}
	if len(outputs) > 0 {
		if err := runBatch(ctx, b.client, `
			UNWIND $rows AS row
			MATCH (fn:Function)
			WHERE fn.qualified = row.symbol OR fn.name = row.symbol
			MERGE (t:Type {name: row.type_name})
			MERGE (fn)-[:RETURNS {pos: row.pos}]->(t)
		`, outputs); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) buildImplementsEdges(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT interface_name, struct_name, method_name, COALESCE(confidence, 0.85)
		FROM interface_impls
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch []map[string]any
	for rows.Next() {
		var interfaceName, structName, methodName string
		var confidence float64
		if err := rows.Scan(&interfaceName, &structName, &methodName, &confidence); err != nil {
			return err
		}
		batch = append(batch, map[string]any{
			"iface_name": interfaceName, "struct_name": structName,
			"method": methodName, "conf": confidence,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	result.ImplementsEdges = len(batch)
	if len(batch) == 0 {
		return nil
	}

	return runBatch(ctx, b.client, `
		UNWIND $rows AS row
		MATCH (fn:Function)
		WHERE fn.qualified CONTAINS row.struct_name AND fn.name = row.method
		MERGE (iface:Type {name: row.iface_name})
		MERGE (fn)-[:IMPLEMENTS {method: row.method, confidence: row.conf}]->(iface)
	`, batch)
}
