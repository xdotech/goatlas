package vector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// VectorIndexer fetches symbols from Postgres, embeds them, and upserts into
// the configured vector store (pgvector or Qdrant).
type VectorIndexer struct {
	pool     *pgxpool.Pool
	store    VectorStore
	embedder Embedder
}

// NewVectorIndexer creates a VectorIndexer.
func NewVectorIndexer(pool *pgxpool.Pool, store VectorStore, embedder Embedder) *VectorIndexer {
	return &VectorIndexer{pool: pool, store: store, embedder: embedder}
}

// IndexResult summarises an embedding run.
type IndexResult struct {
	EmbeddedCount int
	SkippedCount  int
}

type symRow struct {
	ID            int64
	FileID        int64
	Kind          string
	Name          string
	QualifiedName string
	Signature     string
	DocComment    string
}

// IndexRepository embeds all (or only new) function/method/interface symbols.
func (vi *VectorIndexer) IndexRepository(ctx context.Context, force bool) (*IndexResult, error) {
	query := `
		SELECT id, file_id, kind, name, qualified_name, signature, doc_comment
		FROM symbols
		WHERE kind IN ('func', 'method', 'interface')`
	if !force {
		query += ` AND embedded_at IS NULL`
	}
	query += ` ORDER BY id`

	rows, err := vi.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("fetch symbols: %w", err)
	}

	var symbols []symRow
	for rows.Next() {
		var s symRow
		if err := rows.Scan(&s.ID, &s.FileID, &s.Kind, &s.Name, &s.QualifiedName, &s.Signature, &s.DocComment); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan symbol: %w", err)
		}
		symbols = append(symbols, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate symbols: %w", err)
	}

	if len(symbols) == 0 {
		return &IndexResult{}, nil
	}

	// Collect unique file IDs to resolve paths.
	fileIDSet := map[int64]bool{}
	for _, s := range symbols {
		fileIDSet[s.FileID] = true
	}
	fileIDs := make([]int64, 0, len(fileIDSet))
	for id := range fileIDSet {
		fileIDs = append(fileIDs, id)
	}

	filePaths := map[int64]string{}
	fileModules := map[int64]string{}
	fileRows, err := vi.pool.Query(ctx, `SELECT id, path, module FROM files WHERE id = ANY($1)`, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("fetch files: %w", err)
	}
	for fileRows.Next() {
		var id int64
		var path, module string
		if err := fileRows.Scan(&id, &path, &module); err != nil {
			fileRows.Close()
			return nil, fmt.Errorf("scan file: %w", err)
		}
		filePaths[id] = path
		fileModules[id] = module
	}
	fileRows.Close()

	result := &IndexResult{}
	const chunkSize = 32

	for i := 0; i < len(symbols); i += chunkSize {
		end := i + chunkSize
		if end > len(symbols) {
			end = len(symbols)
		}
		chunk := symbols[i:end]

		texts := make([]string, len(chunk))
		for j, s := range chunk {
			texts[j] = buildEmbeddingText(s.QualifiedName, s.Signature, s.DocComment, filePaths[s.FileID])
		}

		vectors, err := vi.embedder.EmbedBatch(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("embed batch starting at %d: %w", i, err)
		}

		points := make([]CodePoint, len(chunk))
		ids := make([]int64, len(chunk))
		for j, s := range chunk {
			points[j] = CodePoint{
				ID:     uint64(s.ID),
				Vector: vectors[j],
				Payload: map[string]interface{}{
					"symbol_id": s.ID,
					"file":      filePaths[s.FileID],
					"kind":      s.Kind,
					"name":      s.Name,
					"service":   fileModules[s.FileID],
					"line":      int64(0),
				},
			}
			ids[j] = s.ID
		}

		if err := vi.store.UpsertPoints(ctx, points); err != nil {
			return nil, fmt.Errorf("upsert batch starting at %d: %w", i, err)
		}

		if _, err := vi.pool.Exec(ctx, `UPDATE symbols SET embedded_at = now() WHERE id = ANY($1)`, ids); err != nil {
			return nil, fmt.Errorf("mark embedded_at: %w", err)
		}

		result.EmbeddedCount += len(chunk)
	}

	return result, nil
}

func buildEmbeddingText(qualifiedName, signature, docComment, filePath string) string {
	text := qualifiedName
	if signature != "" {
		text += "\n" + signature
	}
	if docComment != "" {
		text += "\n// " + docComment
	}
	if filePath != "" {
		text += "\n// File: " + filePath
	}
	return text
}
