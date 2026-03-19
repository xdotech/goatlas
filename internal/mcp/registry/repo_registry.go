package registry

import (
	"context"
	"errors"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RepoRegistry resolves repository names to their indexed metadata.
// It wraps the repositories table and provides backward-compatible defaults.
type RepoRegistry struct {
	pool *pgxpool.Pool
}

// NewRepoRegistry creates a new RepoRegistry backed by the given pool.
func NewRepoRegistry(pool *pgxpool.Pool) *RepoRegistry {
	return &RepoRegistry{pool: pool}
}

// Resolve returns the repository matching the given name.
// If name is empty, returns the first indexed repo (backward compatibility).
func (r *RepoRegistry) Resolve(ctx context.Context, name string) (*domain.Repository, error) {
	repo := &domain.Repository{}

	var query string
	var args []any

	if name == "" {
		// Backward compatibility: return first (most recently indexed) repo
		query = `SELECT id, name, path, last_indexed_at, last_commit FROM repositories ORDER BY last_indexed_at DESC NULLS LAST LIMIT 1`
	} else {
		// Try exact name match first, then path substring match
		query = `SELECT id, name, path, last_indexed_at, last_commit FROM repositories WHERE name = $1 OR path ILIKE '%' || $1 || '%' ORDER BY (name = $1)::int DESC LIMIT 1`
		args = append(args, name)
	}

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&repo.ID, &repo.Name, &repo.Path, &repo.LastIndexedAt, &repo.LastCommit,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return repo, err
}

// List returns all indexed repositories ordered by most recently indexed first.
func (r *RepoRegistry) List(ctx context.Context) ([]domain.Repository, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, path, last_indexed_at, last_commit
		FROM repositories ORDER BY last_indexed_at DESC NULLS LAST
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []domain.Repository
	for rows.Next() {
		var repo domain.Repository
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.Path, &repo.LastIndexedAt, &repo.LastCommit); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}
