package postgres

import (
	"context"
	"errors"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RepoRepo implements domain.RepositoryRepository using PostgreSQL.
type RepoRepo struct {
	pool *pgxpool.Pool
}

// NewRepoRepo creates a new RepoRepo backed by the given pool.
func NewRepoRepo(pool *pgxpool.Pool) *RepoRepo {
	return &RepoRepo{pool: pool}
}

// Upsert inserts or updates a repository record and sets r.ID from the returned id.
func (rr *RepoRepo) Upsert(ctx context.Context, r *domain.Repository) error {
	row := rr.pool.QueryRow(ctx, `
		INSERT INTO repositories (name, path, last_indexed_at, last_commit)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (name) DO UPDATE
			SET path = EXCLUDED.path,
			    last_indexed_at = now(),
			    last_commit = EXCLUDED.last_commit
		RETURNING id
	`, r.Name, r.Path, r.LastCommit)
	return row.Scan(&r.ID)
}

// GetByName returns the repository with the given name, or nil if not found.
func (rr *RepoRepo) GetByName(ctx context.Context, name string) (*domain.Repository, error) {
	r := &domain.Repository{}
	err := rr.pool.QueryRow(ctx, `
		SELECT id, name, path, last_indexed_at, last_commit
		FROM repositories WHERE name = $1
	`, name).Scan(&r.ID, &r.Name, &r.Path, &r.LastIndexedAt, &r.LastCommit)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return r, err
}

// List returns all indexed repositories.
func (rr *RepoRepo) List(ctx context.Context) ([]domain.Repository, error) {
	rows, err := rr.pool.Query(ctx, `
		SELECT id, name, path, last_indexed_at, last_commit
		FROM repositories ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var r domain.Repository
		if err := rows.Scan(&r.ID, &r.Name, &r.Path, &r.LastIndexedAt, &r.LastCommit); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}
