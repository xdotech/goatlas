package postgres

import (
	"context"
	"fmt"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EndpointRepo implements domain.EndpointRepository using PostgreSQL.
type EndpointRepo struct {
	pool *pgxpool.Pool
}

// NewEndpointRepo creates a new EndpointRepo backed by the given pool.
func NewEndpointRepo(pool *pgxpool.Pool) *EndpointRepo {
	return &EndpointRepo{pool: pool}
}

// BulkInsert inserts multiple API endpoints using COPY protocol for efficiency.
func (r *EndpointRepo) BulkInsert(ctx context.Context, endpoints []domain.APIEndpoint) error {
	if len(endpoints) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(endpoints))
	for i, e := range endpoints {
		rows[i] = []interface{}{e.FileID, e.Method, e.Path, e.HandlerName, e.Framework, e.Line}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"api_endpoints"},
		[]string{"file_id", "method", "path", "handler_name", "framework", "line"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByFileID removes all endpoints associated with the given file id.
func (r *EndpointRepo) DeleteByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM api_endpoints WHERE file_id = $1`, fileID)
	return err
}

// List returns API endpoints, optionally filtered by HTTP method and service (repository name).
// Uses DISTINCT ON (method, path) to deduplicate endpoints from nested/repeated repos.
func (r *EndpointRepo) List(ctx context.Context, method, service string) ([]domain.APIEndpoint, error) {
	query := `SELECT DISTINCT ON (e.method, e.path)
	                  e.id, e.file_id, e.method, e.path, e.handler_name, e.framework, e.line,
	                  f.path AS file_path, COALESCE(r.name, '') AS repo_name
	           FROM api_endpoints e
	           JOIN files f ON e.file_id = f.id
	           LEFT JOIN repositories r ON f.repo_id = r.id
	           WHERE 1=1`
	args := []interface{}{}
	argN := 1
	if method != "" {
		query += fmt.Sprintf(" AND e.method = $%d", argN)
		args = append(args, method)
		argN++
	}
	if service != "" {
		query += fmt.Sprintf(" AND r.name ILIKE '%%' || $%d || '%%'", argN)
		args = append(args, service)
		argN++
	}
	_ = argN
	query += " ORDER BY e.method, e.path, r.last_indexed_at DESC NULLS LAST"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []domain.APIEndpoint
	for rows.Next() {
		var e domain.APIEndpoint
		if err := rows.Scan(&e.ID, &e.FileID, &e.Method, &e.Path, &e.HandlerName, &e.Framework, &e.Line, &e.FilePath, &e.RepoName); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, rows.Err()
}
