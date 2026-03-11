package postgres

import (
	"context"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ServiceConnectionRepo implements domain.ServiceConnectionRepository using PostgreSQL.
type ServiceConnectionRepo struct {
	pool *pgxpool.Pool
}

// NewServiceConnectionRepo creates a new ServiceConnectionRepo backed by the given pool.
func NewServiceConnectionRepo(pool *pgxpool.Pool) *ServiceConnectionRepo {
	return &ServiceConnectionRepo{pool: pool}
}

// BulkInsert inserts a batch of service connections.
func (r *ServiceConnectionRepo) BulkInsert(ctx context.Context, conns []domain.ServiceConnection) error {
	if len(conns) == 0 {
		return nil
	}
	for _, c := range conns {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO service_connections (repo_id, conn_type, target, file_id, line)
			VALUES ($1, $2, $3, $4, $5)
		`, c.RepoID, c.ConnType, c.Target, c.FileID, c.Line)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteByRepoID removes all connections for the given repo (used before re-indexing).
func (r *ServiceConnectionRepo) DeleteByRepoID(ctx context.Context, repoID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM service_connections WHERE repo_id = $1`, repoID)
	return err
}

// List returns all connections, optionally filtered by type.
func (r *ServiceConnectionRepo) List(ctx context.Context, connType string) ([]domain.ServiceConnection, error) {
	query := `SELECT id, repo_id, conn_type, target, file_id, line FROM service_connections`
	args := []any{}
	if connType != "" {
		query += ` WHERE conn_type = $1`
		args = append(args, connType)
	}
	query += ` ORDER BY repo_id, conn_type, target`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []domain.ServiceConnection
	for rows.Next() {
		var c domain.ServiceConnection
		if err := rows.Scan(&c.ID, &c.RepoID, &c.ConnType, &c.Target, &c.FileID, &c.Line); err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}

// ListByRepo returns all connections for a specific repository.
func (r *ServiceConnectionRepo) ListByRepo(ctx context.Context, repoID int64) ([]domain.ServiceConnection, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, repo_id, conn_type, target, file_id, line
		FROM service_connections WHERE repo_id = $1 ORDER BY conn_type, target
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []domain.ServiceConnection
	for rows.Next() {
		var c domain.ServiceConnection
		if err := rows.Scan(&c.ID, &c.RepoID, &c.ConnType, &c.Target, &c.FileID, &c.Line); err != nil {
			return nil, err
		}
		conns = append(conns, c)
	}
	return conns, rows.Err()
}
