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
	PackageNodes  int
	FileNodes     int
	FunctionNodes int
	TypeNodes     int
	ImportEdges   int
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
