package postgres

import (
	"context"
	"fmt"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InterfaceImplRepo implements domain.InterfaceImplRepository using PostgreSQL.
type InterfaceImplRepo struct {
	pool *pgxpool.Pool
}

// NewInterfaceImplRepo creates a new InterfaceImplRepo.
func NewInterfaceImplRepo(pool *pgxpool.Pool) *InterfaceImplRepo {
	return &InterfaceImplRepo{pool: pool}
}

// BulkInsert inserts multiple interface implementation records using COPY protocol.
func (r *InterfaceImplRepo) BulkInsert(ctx context.Context, impls []domain.InterfaceImpl) error {
	if len(impls) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(impls))
	for i, impl := range impls {
		rows[i] = []interface{}{impl.FileID, impl.InterfaceName, impl.StructName, impl.MethodName, impl.Confidence}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"interface_impls"},
		[]string{"file_id", "interface_name", "struct_name", "method_name", "confidence"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByFileID removes all interface implementations for the given file.
func (r *InterfaceImplRepo) DeleteByFileID(ctx context.Context, fileID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM interface_impls WHERE file_id = $1`, fileID)
	return err
}

// FindImplementations returns struct methods that implement the given interface method.
func (r *InterfaceImplRepo) FindImplementations(ctx context.Context, interfaceName, methodName string) ([]domain.InterfaceImpl, error) {
	query := `SELECT id, file_id, interface_name, struct_name, method_name, COALESCE(confidence, 0.85)
	           FROM interface_impls WHERE 1=1`
	args := []interface{}{}
	argN := 1

	if interfaceName != "" {
		query += fmt.Sprintf(" AND interface_name ILIKE '%%' || $%d || '%%'", argN)
		args = append(args, interfaceName)
		argN++
	}
	if methodName != "" {
		query += fmt.Sprintf(" AND method_name = $%d", argN)
		args = append(args, methodName)
		argN++
	}
	_ = argN

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var impls []domain.InterfaceImpl
	for rows.Next() {
		var impl domain.InterfaceImpl
		if err := rows.Scan(&impl.ID, &impl.FileID, &impl.InterfaceName, &impl.StructName, &impl.MethodName, &impl.Confidence); err != nil {
			return nil, err
		}
		impls = append(impls, impl)
	}
	return impls, rows.Err()
}
