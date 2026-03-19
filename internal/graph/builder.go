package graph

import (
	"context"
	"fmt"

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
	}
	for _, idx := range indexes {
		// Non-fatal: older Neo4j versions may use different syntax.
		_ = b.client.RunCypher(ctx, idx, nil)
	}
	return nil
}

func (b *Builder) buildFileNodes(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `SELECT id, path, module FROM files`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type fileRec struct {
		ID     int64
		Path   string
		Module string
	}

	var batch []fileRec
	for rows.Next() {
		var r fileRec
		if err := rows.Scan(&r.ID, &r.Path, &r.Module); err != nil {
			return err
		}
		batch = append(batch, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows.Close()

	pkgs := map[string]struct{}{}
	for _, r := range batch {
		cypher := `
			MERGE (pkg:Package {name: $module})
			MERGE (f:File {path: $path})
			SET f.package = $module
			MERGE (pkg)-[:DEFINES]->(f)
		`
		if err := b.client.RunCypher(ctx, cypher, map[string]any{
			"module": r.Module,
			"path":   r.Path,
		}); err != nil {
			return err
		}
		pkgs[r.Module] = struct{}{}
		result.FileNodes++
	}
	result.PackageNodes = len(pkgs)
	return nil
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

	for rows.Next() {
		var kind, name, qualifiedName, signature, filePath string
		var line int
		if err := rows.Scan(&kind, &name, &qualifiedName, &signature, &line, &filePath); err != nil {
			return err
		}

		var cypher string
		var params map[string]any

		switch kind {
		case "func", "method":
			cypher = `
				MERGE (fn:Function {qualified: $qualified})
				SET fn.name = $name, fn.signature = $signature, fn.line = $line
				MERGE (f:File {path: $file})
				MERGE (f)-[:DEFINES]->(fn)
			`
			params = map[string]any{
				"qualified": qualifiedName,
				"name":      name,
				"signature": signature,
				"line":      line,
				"file":      filePath,
			}
			result.FunctionNodes++
		default:
			cypher = `
				MERGE (t:Type {qualified: $qualified})
				SET t.name = $name, t.kind = $kind, t.line = $line
				MERGE (f:File {path: $file})
				MERGE (f)-[:DEFINES]->(t)
			`
			params = map[string]any{
				"qualified": qualifiedName,
				"name":      name,
				"kind":      kind,
				"line":      line,
				"file":      filePath,
			}
			result.TypeNodes++
		}

		if err := b.client.RunCypher(ctx, cypher, params); err != nil {
			return err
		}
	}
	return rows.Err()
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

	for rows.Next() {
		var fromPath, importPath string
		if err := rows.Scan(&fromPath, &importPath); err != nil {
			return err
		}
		cypher := `
			MERGE (importer:File {path: $fromPath})
			MERGE (imported:Package {name: $importPath})
			MERGE (importer)-[:IMPORTS]->(imported)
		`
		if err := b.client.RunCypher(ctx, cypher, map[string]any{
			"fromPath":   fromPath,
			"importPath": importPath,
		}); err != nil {
			return err
		}
		result.ImportEdges++
	}
	return rows.Err()
}

func (b *Builder) buildServiceConnections(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT r.name, sc.conn_type, sc.target, sc.line, f.path
		FROM service_connections sc
		JOIN repositories r ON sc.repo_id = r.id
		LEFT JOIN files f ON sc.file_id = f.id
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	services := map[string]struct{}{}
	for rows.Next() {
		var repoName, connType, target string
		var line int
		var filePath *string
		if err := rows.Scan(&repoName, &connType, &target, &line, &filePath); err != nil {
			return err
		}

		// Create Service node for the source repo
		if _, exists := services[repoName]; !exists {
			cypher := `MERGE (s:Service {name: $name})`
			if err := b.client.RunCypher(ctx, cypher, map[string]any{"name": repoName}); err != nil {
				return err
			}
			services[repoName] = struct{}{}
			result.ServiceNodes++
		}

		// Create appropriate edge based on connection type
		var cypher string
		params := map[string]any{
			"source": repoName,
			"target": target,
			"line":   line,
		}

		switch connType {
		case "grpc":
			cypher = `
				MERGE (src:Service {name: $source})
				MERGE (tgt:Service {name: $target})
				MERGE (src)-[:CALLS_GRPC {target: $target, line: $line}]->(tgt)
			`
		case "kafka_publish":
			cypher = `
				MERGE (src:Service {name: $source})
				MERGE (topic:Topic {name: $target})
				MERGE (src)-[:PUBLISHES {line: $line}]->(topic)
			`
		case "kafka_consume":
			cypher = `
				MERGE (src:Service {name: $source})
				MERGE (topic:Topic {name: $target})
				MERGE (src)-[:SUBSCRIBES {line: $line}]->(topic)
			`
		case "http_api":
			cypher = `
				MERGE (src:Service {name: $source})
				MERGE (tgt:Service {name: $target})
				MERGE (src)-[:CALLS_API {line: $line}]->(tgt)
			`
		default:
			continue
		}

		if err := b.client.RunCypher(ctx, cypher, params); err != nil {
			return err
		}
		result.ConnectionEdges++
	}
	return rows.Err()
}

// buildComponentAPIEdges creates Component nodes and CALLS_API edges
// from the component_api_calls table.
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

	for rows.Next() {
		var component, method, apiPath, targetService, filePath string
		if err := rows.Scan(&component, &method, &apiPath, &targetService, &filePath); err != nil {
			return err
		}

		cypher := `
			MERGE (comp:Component {name: $component, file: $file})
			MERGE (ep:Endpoint {path: $api_path})
			MERGE (comp)-[:CALLS_API {method: $method, service: $service}]->(ep)
		`
		params := map[string]any{
			"component": component,
			"file":      filePath,
			"api_path":  apiPath,
			"method":    method,
			"service":   targetService,
		}
		if err := b.client.RunCypher(ctx, cypher, params); err != nil {
			return err
		}
		result.ComponentEdges++
	}
	return rows.Err()
}

// buildCallEdges creates CALLS edges between Function nodes
// from the function_calls table.
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

		cypher := `
			MATCH (caller:Function {qualified: $caller})
			MATCH (callee:Function)
			WHERE callee.qualified = $callee_qualified
			   OR callee.name = $callee_name
			MERGE (caller)-[:CALLS {line: $line, confidence: $conf}]->(callee)
		`
		params := map[string]any{
			"caller":           callerQName,
			"callee_qualified": cq,
			"callee_name":      calleeName,
			"line":             line,
			"conf":             confidence,
		}
		if err := b.client.RunCypher(ctx, cypher, params); err != nil {
			return err
		}
		result.CallEdges++
	}
	return rows.Err()

}

// buildTypeFlowEdges creates ACCEPTS and RETURNS edges between Function and Type nodes
// from the type_usages table.
func (b *Builder) buildTypeFlowEdges(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT tu.symbol_name, tu.type_name, tu.direction, tu.position
		FROM type_usages tu
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var symbolName, typeName, direction string
		var position int
		if err := rows.Scan(&symbolName, &typeName, &direction, &position); err != nil {
			return err
		}

		var cypher string
		switch direction {
		case "input":
			cypher = `
				MATCH (fn:Function)
				WHERE fn.qualified = $symbol OR fn.name = $symbol
				MERGE (t:Type {name: $type_name})
				MERGE (fn)-[:ACCEPTS {pos: $pos}]->(t)`
		case "output":
			cypher = `
				MATCH (fn:Function)
				WHERE fn.qualified = $symbol OR fn.name = $symbol
				MERGE (t:Type {name: $type_name})
				MERGE (fn)-[:RETURNS {pos: $pos}]->(t)`
		default:
			continue
		}

		params := map[string]any{
			"symbol":    symbolName,
			"type_name": typeName,
			"pos":       position,
		}
		if err := b.client.RunCypher(ctx, cypher, params); err != nil {
			return err
		}
		result.TypeFlowEdges++
	}
	return rows.Err()
}

// buildImplementsEdges creates IMPLEMENTS edges between struct method (Function)
// and interface (Type) nodes, using data from the interface_impls table.
func (b *Builder) buildImplementsEdges(ctx context.Context, result *BuildResult) error {
	rows, err := b.pool.Query(ctx, `
		SELECT interface_name, struct_name, method_name, COALESCE(confidence, 0.85)
		FROM interface_impls
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var interfaceName, structName, methodName string
		var confidence float64
		if err := rows.Scan(&interfaceName, &structName, &methodName, &confidence); err != nil {
			return err
		}

		cypher := `
			MATCH (fn:Function)
			WHERE fn.qualified CONTAINS $struct_name AND fn.name = $method
			MERGE (iface:Type {name: $iface_name})
			MERGE (fn)-[:IMPLEMENTS {method: $method, confidence: $conf}]->(iface)
		`
		params := map[string]any{
			"struct_name": structName,
			"method":      methodName,
			"iface_name":  interfaceName,
			"conf":        confidence,
		}
		if err := b.client.RunCypher(ctx, cypher, params); err != nil {
			return err
		}
		result.ImplementsEdges++
	}
	return rows.Err()
}
