package postgres

import (
	"context"

	"github.com/goatlas/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FunctionCallRepo implements domain.FunctionCallRepository using PostgreSQL.
type FunctionCallRepo struct {
	pool *pgxpool.Pool
}

// NewFunctionCallRepo creates a new FunctionCallRepo.
func NewFunctionCallRepo(pool *pgxpool.Pool) *FunctionCallRepo {
	return &FunctionCallRepo{pool: pool}
}

// BulkInsert inserts multiple function call records using COPY protocol.
func (r *FunctionCallRepo) BulkInsert(ctx context.Context, calls []domain.FunctionCall) error {
	if len(calls) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(calls))
	for i, c := range calls {
		rows[i] = []interface{}{c.FileID, c.CallerQualifiedName, c.CalleeName, c.CalleePackage, c.Line, c.Col, c.Confidence}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"function_calls"},
		[]string{"file_id", "caller_qualified_name", "callee_name", "callee_package", "line", "col", "confidence"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByFileID removes all function calls for the given file.
func (r *FunctionCallRepo) DeleteByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM function_calls WHERE file_id = $1`, fileID)
	return err
}
