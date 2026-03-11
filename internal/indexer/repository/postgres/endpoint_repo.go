package postgres

import (
	"context"
	"fmt"

	"github.com/goatlas/goatlas/internal/indexer/domain"
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

// List returns API endpoints, optionally filtered by HTTP method.
// The service parameter is reserved for future path-prefix filtering.
func (r *EndpointRepo) List(ctx context.Context, method, service string) ([]domain.APIEndpoint, error) {
	query := `SELECT id, file_id, method, path, handler_name, framework, line FROM api_endpoints WHERE 1=1`
	args := []interface{}{}
	argN := 1
	if method != "" {
		query += fmt.Sprintf(" AND method = $%d", argN)
		args = append(args, method)
		argN++
	}
	_ = argN // suppress unused warning; reserved for future filters

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []domain.APIEndpoint
	for rows.Next() {
		var e domain.APIEndpoint
		if err := rows.Scan(&e.ID, &e.FileID, &e.Method, &e.Path, &e.HandlerName, &e.Framework, &e.Line); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, rows.Err()
}
