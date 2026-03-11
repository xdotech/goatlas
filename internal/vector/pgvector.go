package vector

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"
)

// PgVectorStore implements VectorStore using PostgreSQL with pgvector extension.
type PgVectorStore struct {
	pool *pgxpool.Pool
}

// NewPgVectorStore creates a PgVectorStore backed by the given connection pool.
func NewPgVectorStore(pool *pgxpool.Pool) *PgVectorStore {
	return &PgVectorStore{pool: pool}
}

// UpsertPoints inserts or updates embeddings in the symbol_embeddings table.
func (s *PgVectorStore) UpsertPoints(ctx context.Context, points []CodePoint) error {
	if len(points) == 0 {
		return nil
	}

	const batchSize = 100
	for i := 0; i < len(points); i += batchSize {
		end := i + batchSize
		if end > len(points) {
			end = len(points)
		}
		if err := s.upsertBatch(ctx, points[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *PgVectorStore) upsertBatch(ctx context.Context, points []CodePoint) error {
	// Build a multi-row INSERT ... ON CONFLICT DO UPDATE.
	var sb strings.Builder
	sb.WriteString(`INSERT INTO symbol_embeddings (symbol_id, embedding, file, kind, name, service, line)
VALUES `)

	args := make([]interface{}, 0, len(points)*7)
	for i, p := range points {
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 7
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7)

		file, _ := p.Payload["file"].(string)
		kind, _ := p.Payload["kind"].(string)
		name, _ := p.Payload["name"].(string)
		service, _ := p.Payload["service"].(string)
		line := int64(0)
		if v, ok := p.Payload["line"].(int64); ok {
			line = v
		}

		args = append(args,
			int64(p.ID),
			pgvector.NewVector(p.Vector),
			file, kind, name, service, line,
		)
	}

	sb.WriteString(` ON CONFLICT (symbol_id) DO UPDATE SET
		embedding  = EXCLUDED.embedding,
		file       = EXCLUDED.file,
		kind       = EXCLUDED.kind,
		name       = EXCLUDED.name,
		service    = EXCLUDED.service,
		line       = EXCLUDED.line,
		created_at = now()`)

	_, err := s.pool.Exec(ctx, sb.String(), args...)
	if err != nil {
		return fmt.Errorf("pgvector upsert: %w", err)
	}
	return nil
}

// Search performs cosine similarity search using pgvector's <=> operator.
func (s *PgVectorStore) Search(ctx context.Context, vec []float32, limit int, filter map[string]string) ([]SearchResult, error) {
	// Build query with optional WHERE filters.
	var sb strings.Builder
	sb.WriteString(`SELECT symbol_id, 1 - (embedding <=> $1::vector) AS score,
		file, kind, name, service, line
	FROM symbol_embeddings`)

	args := []interface{}{pgvector.NewVector(vec)}
	argIdx := 2

	if len(filter) > 0 {
		sb.WriteString(" WHERE ")
		first := true
		for k, v := range filter {
			if !first {
				sb.WriteString(" AND ")
			}
			// Only allow known columns to prevent SQL injection.
			col := sanitizeColumn(k)
			if col == "" {
				continue
			}
			fmt.Fprintf(&sb, "%s = $%d", col, argIdx)
			args = append(args, v)
			argIdx++
			first = false
		}
	}

	fmt.Fprintf(&sb, " ORDER BY embedding <=> $1::vector LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("pgvector search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.SymbolID, &r.Score, &r.File, &r.Kind, &r.Name, &r.Service, &r.Line); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// Close is a no-op — the pool is managed externally.
func (s *PgVectorStore) Close() error {
	return nil
}

// sanitizeColumn maps allowed filter keys to column names.
func sanitizeColumn(key string) string {
	allowed := map[string]string{
		"service": "service",
		"kind":    "kind",
		"file":    "file",
		"name":    "name",
	}
	return allowed[key]
}
