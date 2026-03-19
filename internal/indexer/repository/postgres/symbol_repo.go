package postgres

import (
	"context"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SymbolRepo implements domain.SymbolRepository using PostgreSQL.
type SymbolRepo struct {
	pool *pgxpool.Pool
}

// NewSymbolRepo creates a new SymbolRepo backed by the given pool.
func NewSymbolRepo(pool *pgxpool.Pool) *SymbolRepo {
	return &SymbolRepo{pool: pool}
}

// BulkInsert inserts multiple symbols using COPY protocol for efficiency.
func (r *SymbolRepo) BulkInsert(ctx context.Context, symbols []domain.Symbol) error {
	if len(symbols) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(symbols))
	for i, s := range symbols {
		rows[i] = []interface{}{s.FileID, s.Kind, s.Name, s.QualifiedName, s.Signature, s.Receiver, s.Line, s.Col, s.DocComment}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"symbols"},
		[]string{"file_id", "kind", "name", "qualified_name", "signature", "receiver", "line", "col", "doc_comment"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByFileID removes all symbols associated with the given file id.
func (r *SymbolRepo) DeleteByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM symbols WHERE file_id = $1`, fileID)
	return err
}

// Search performs a full-text search over symbols, optionally filtered by kind.
func (r *SymbolRepo) Search(ctx context.Context, query string, limit int, kind string) ([]domain.Symbol, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows pgx.Rows
	var err error
	if kind != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, file_id, kind, name, qualified_name, signature, receiver, line, col, doc_comment
			FROM symbols
			WHERE search_vector @@ plainto_tsquery('english', $1) AND kind = $3
			ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
			LIMIT $2
		`, query, limit, kind)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, file_id, kind, name, qualified_name, signature, receiver, line, col, doc_comment
			FROM symbols
			WHERE search_vector @@ plainto_tsquery('english', $1)
			ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
			LIMIT $2
		`, query, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// GetByFile returns all symbols for the given file id, ordered by line number.
func (r *SymbolRepo) GetByFile(ctx context.Context, fileID int64) ([]domain.Symbol, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, file_id, kind, name, qualified_name, signature, receiver, line, col, doc_comment
		FROM symbols WHERE file_id = $1 ORDER BY line
	`, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// ListByKinds returns all symbols matching one of the given kinds, ordered by name.
func (r *SymbolRepo) ListByKinds(ctx context.Context, kinds []string, limit int) ([]domain.Symbol, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, file_id, kind, name, qualified_name, signature, receiver, line, col, doc_comment
		FROM symbols
		WHERE kind = ANY($1)
		ORDER BY name
		LIMIT $2
	`, kinds, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSymbols(rows)
}

// SearchWithFile performs full-text search and returns symbols with their file paths.
func (r *SymbolRepo) SearchWithFile(ctx context.Context, query string, limit int, kind string) ([]domain.SymbolWithFile, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows pgx.Rows
	var err error
	if kind != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT s.id, s.file_id, s.kind, s.name, s.qualified_name, s.signature, s.receiver, s.line, s.col, s.doc_comment, f.path
			FROM symbols s
			JOIN files f ON s.file_id = f.id
			WHERE s.search_vector @@ plainto_tsquery('english', $1) AND s.kind = $3
			ORDER BY ts_rank(s.search_vector, plainto_tsquery('english', $1)) DESC
			LIMIT $2
		`, query, limit, kind)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT s.id, s.file_id, s.kind, s.name, s.qualified_name, s.signature, s.receiver, s.line, s.col, s.doc_comment, f.path
			FROM symbols s
			JOIN files f ON s.file_id = f.id
			WHERE s.search_vector @@ plainto_tsquery('english', $1)
			ORDER BY ts_rank(s.search_vector, plainto_tsquery('english', $1)) DESC
			LIMIT $2
		`, query, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.SymbolWithFile
	for rows.Next() {
		var sw domain.SymbolWithFile
		if err := rows.Scan(
			&sw.ID, &sw.FileID, &sw.Kind, &sw.Name, &sw.QualifiedName,
			&sw.Signature, &sw.Receiver, &sw.Line, &sw.Col, &sw.DocComment,
			&sw.FilePath,
		); err != nil {
			return nil, err
		}
		results = append(results, sw)
	}
	return results, rows.Err()
}

// SearchRanked performs full-text search and returns symbols with 1-based BM25 rank for RRF.
func (r *SymbolRepo) SearchRanked(ctx context.Context, query string, limit int, kind string) ([]domain.SymbolWithRank, error) {
	if limit <= 0 {
		limit = 20
	}
	base := `
		SELECT s.id, s.file_id, s.kind, s.name, s.qualified_name, s.signature,
		       s.receiver, s.line, s.col, s.doc_comment, f.path,
		       cast(row_number() OVER (ORDER BY ts_rank(s.search_vector, plainto_tsquery('english', $1)) DESC) AS int) AS rank
		FROM symbols s
		JOIN files f ON s.file_id = f.id
		WHERE s.search_vector @@ plainto_tsquery('english', $1)`
	var rows pgx.Rows
	var err error
	if kind != "" {
		rows, err = r.pool.Query(ctx, base+` AND s.kind = $3 LIMIT $2`, query, limit, kind)
	} else {
		rows, err = r.pool.Query(ctx, base+` LIMIT $2`, query, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []domain.SymbolWithRank
	for rows.Next() {
		var sw domain.SymbolWithRank
		if err := rows.Scan(
			&sw.ID, &sw.FileID, &sw.Kind, &sw.Name, &sw.QualifiedName,
			&sw.Signature, &sw.Receiver, &sw.Line, &sw.Col, &sw.DocComment,
			&sw.FilePath, &sw.Rank,
		); err != nil {
			return nil, err
		}
		results = append(results, sw)
	}
	return results, rows.Err()
}

func scanSymbols(rows pgx.Rows) ([]domain.Symbol, error) {
	var symbols []domain.Symbol
	for rows.Next() {
		var s domain.Symbol
		if err := rows.Scan(&s.ID, &s.FileID, &s.Kind, &s.Name, &s.QualifiedName, &s.Signature, &s.Receiver, &s.Line, &s.Col, &s.DocComment); err != nil {
			return nil, err
		}
		symbols = append(symbols, s)
	}
	return symbols, rows.Err()
}
