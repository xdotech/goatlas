package postgres

import (
	"context"
	"errors"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FileRepo implements domain.FileRepository using PostgreSQL.
type FileRepo struct {
	pool *pgxpool.Pool
}

// NewFileRepo creates a new FileRepo backed by the given pool.
func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{pool: pool}
}

// Upsert inserts or updates a file record and sets f.ID from the returned id.
func (r *FileRepo) Upsert(ctx context.Context, f *domain.File) error {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO files (repo_id, path, module, hash, last_scanned)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (repo_id, path) DO UPDATE
			SET module = EXCLUDED.module,
			    hash = EXCLUDED.hash,
			    last_scanned = now()
		RETURNING id
	`, f.RepoID, f.Path, f.Module, f.Hash)
	return row.Scan(&f.ID)
}

// GetByPath returns the file with the given repoID and path, or nil if not found.
func (r *FileRepo) GetByPath(ctx context.Context, repoID int64, path string) (*domain.File, error) {
	f := &domain.File{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, repo_id, path, module, hash, last_scanned
		FROM files WHERE repo_id = $1 AND path = $2
	`, repoID, path).Scan(&f.ID, &f.RepoID, &f.Path, &f.Module, &f.Hash, &f.LastScanned)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return f, err
}

// DeleteByID removes the file record with the given id (cascades to symbols etc.).
func (r *FileRepo) DeleteByID(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM files WHERE id = $1`, id)
	return err
}

// DeleteByPath removes the file record for the given repo and relative path.
func (r *FileRepo) DeleteByPath(ctx context.Context, repoID int64, path string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM files WHERE repo_id = $1 AND path = $2`, repoID, path)
	return err
}
