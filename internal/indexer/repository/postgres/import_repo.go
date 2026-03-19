package postgres

import (
	"context"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImportRepo implements domain.ImportRepository using PostgreSQL.
type ImportRepo struct {
	pool *pgxpool.Pool
}

// NewImportRepo creates a new ImportRepo backed by the given pool.
func NewImportRepo(pool *pgxpool.Pool) *ImportRepo {
	return &ImportRepo{pool: pool}
}

// BulkInsert inserts multiple import records using COPY protocol for efficiency.
func (r *ImportRepo) BulkInsert(ctx context.Context, imports []domain.Import) error {
	if len(imports) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(imports))
	for i, imp := range imports {
		rows[i] = []interface{}{imp.FileID, imp.ImportPath, imp.Alias}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"imports"},
		[]string{"file_id", "import_path", "alias"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByFileID removes all imports associated with the given file id.
func (r *ImportRepo) DeleteByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM imports WHERE file_id = $1`, fileID)
	return err
}
