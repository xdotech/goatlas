package postgres

import (
	"context"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TypeUsageRepo implements domain.TypeUsageRepository using PostgreSQL.
type TypeUsageRepo struct {
	pool *pgxpool.Pool
}

// NewTypeUsageRepo creates a new TypeUsageRepo.
func NewTypeUsageRepo(pool *pgxpool.Pool) *TypeUsageRepo {
	return &TypeUsageRepo{pool: pool}
}

// BulkInsert inserts multiple type usage records using COPY protocol.
func (r *TypeUsageRepo) BulkInsert(ctx context.Context, usages []domain.TypeUsage) error {
	if len(usages) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(usages))
	for i, u := range usages {
		rows[i] = []interface{}{u.FileID, u.SymbolName, u.TypeName, u.Direction, u.Position, u.Line}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"type_usages"},
		[]string{"file_id", "symbol_name", "type_name", "direction", "position", "line"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByFileID removes all type usages for the given file.
func (r *TypeUsageRepo) DeleteByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM type_usages WHERE file_id = $1`, fileID)
	return err
}

// FindByType returns all functions that use the given type name.
func (r *TypeUsageRepo) FindByType(ctx context.Context, typeName string) ([]domain.TypeUsage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, file_id, symbol_name, type_name, direction, position, line
		FROM type_usages
		WHERE type_name ILIKE '%' || $1 || '%'
		ORDER BY direction, symbol_name
		LIMIT 50
	`, typeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []domain.TypeUsage
	for rows.Next() {
		var u domain.TypeUsage
		if err := rows.Scan(&u.ID, &u.FileID, &u.SymbolName, &u.TypeName, &u.Direction, &u.Position, &u.Line); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, rows.Err()
}

// FindBySymbol returns all types used by the given function.
func (r *TypeUsageRepo) FindBySymbol(ctx context.Context, symbolName string) ([]domain.TypeUsage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, file_id, symbol_name, type_name, direction, position, line
		FROM type_usages
		WHERE symbol_name ILIKE '%' || $1 || '%'
		ORDER BY direction, position
		LIMIT 50
	`, symbolName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var usages []domain.TypeUsage
	for rows.Next() {
		var u domain.TypeUsage
		if err := rows.Scan(&u.ID, &u.FileID, &u.SymbolName, &u.TypeName, &u.Direction, &u.Position, &u.Line); err != nil {
			return nil, err
		}
		usages = append(usages, u)
	}
	return usages, rows.Err()
}
