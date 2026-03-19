package usecase

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// getFirstRepoID returns the ID of the first indexed repository (ordered by id).
func getFirstRepoID(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM repositories ORDER BY id LIMIT 1`).Scan(&id)
	return id, err
}
