package postgres

import (
	"context"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ComponentAPICallRepo implements domain.ComponentAPICallRepository using PostgreSQL.
type ComponentAPICallRepo struct {
	pool *pgxpool.Pool
}

// NewComponentAPICallRepo creates a new ComponentAPICallRepo.
func NewComponentAPICallRepo(pool *pgxpool.Pool) *ComponentAPICallRepo {
	return &ComponentAPICallRepo{pool: pool}
}

// BulkInsert inserts multiple component API call records using COPY protocol.
func (r *ComponentAPICallRepo) BulkInsert(ctx context.Context, calls []domain.ComponentAPICall) error {
	if len(calls) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(calls))
	for i, c := range calls {
		rows[i] = []interface{}{c.FileID, c.Component, c.HttpMethod, c.APIPath, c.TargetService, c.Line, c.Col}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"component_api_calls"},
		[]string{"file_id", "component", "http_method", "api_path", "target_service", "line", "col"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByFileID removes all component API calls for the given file.
func (r *ComponentAPICallRepo) DeleteByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM component_api_calls WHERE file_id = $1`, fileID)
	return err
}

// FindByComponent returns all API calls made by a given component name.
func (r *ComponentAPICallRepo) FindByComponent(ctx context.Context, component string) ([]domain.ComponentAPICall, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.file_id, c.component, c.http_method, c.api_path, c.target_service, c.line, c.col
		FROM component_api_calls c
		WHERE c.component ILIKE '%' || $1 || '%'
		ORDER BY c.api_path
	`, component)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComponentAPICalls(rows)
}

// FindByAPIPath returns all component API calls matching a given API path pattern.
func (r *ComponentAPICallRepo) FindByAPIPath(ctx context.Context, apiPath string) ([]domain.ComponentAPICall, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.file_id, c.component, c.http_method, c.api_path, c.target_service, c.line, c.col
		FROM component_api_calls c
		WHERE c.api_path ILIKE '%' || $1 || '%'
		ORDER BY c.component
	`, apiPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanComponentAPICalls(rows)
}

func scanComponentAPICalls(rows pgx.Rows) ([]domain.ComponentAPICall, error) {
	var calls []domain.ComponentAPICall
	for rows.Next() {
		var c domain.ComponentAPICall
		if err := rows.Scan(&c.ID, &c.FileID, &c.Component, &c.HttpMethod, &c.APIPath, &c.TargetService, &c.Line, &c.Col); err != nil {
			return nil, err
		}
		calls = append(calls, c)
	}
	return calls, rows.Err()
}
