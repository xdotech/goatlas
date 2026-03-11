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
		INSERT INTO files (repo, path, module, hash, last_scanned)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (repo, path) DO UPDATE
			SET module = EXCLUDED.module,
			    hash = EXCLUDED.hash,
			    last_scanned = now()
		RETURNING id
	`, f.Repo, f.Path, f.Module, f.Hash)
	return row.Scan(&f.ID)
}

// GetByPath returns the file with the given repo and path, or nil if not found.
// If repo is empty, it matches by path only (first result).
func (r *FileRepo) GetByPath(ctx context.Context, repo, path string) (*domain.File, error) {
	f := &domain.File{}
	var err error
	if repo == "" {
		err = r.pool.QueryRow(ctx, `
			SELECT id, repo, path, module, hash, last_scanned FROM files WHERE path = $1 LIMIT 1
		`, path).Scan(&f.ID, &f.Repo, &f.Path, &f.Module, &f.Hash, &f.LastScanned)
	} else {
		err = r.pool.QueryRow(ctx, `
			SELECT id, repo, path, module, hash, last_scanned FROM files WHERE repo = $1 AND path = $2
		`, repo, path).Scan(&f.ID, &f.Repo, &f.Path, &f.Module, &f.Hash, &f.LastScanned)
	}
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
