package usecase

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ListServicesUseCase lists all distinct modules/packages in the indexed repository.
type ListServicesUseCase struct {
	pool *pgxpool.Pool
}

// NewListServicesUseCase constructs a ListServicesUseCase.
func NewListServicesUseCase(pool *pgxpool.Pool) *ListServicesUseCase {
	return &ListServicesUseCase{pool: pool}
}

// Execute queries distinct modules from the files table.
// Falls back to indexed repository names when no modules are populated.
func (uc *ListServicesUseCase) Execute(ctx context.Context) (string, error) {
	rows, err := uc.pool.Query(ctx, `SELECT DISTINCT module FROM files WHERE module != '' ORDER BY module`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var module string
		if err := rows.Scan(&module); err != nil {
			return "", err
		}
		services = append(services, module)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	if len(services) > 0 {
		return "Services/packages:\n\n" + strings.Join(services, "\n"), nil
	}

	// Fallback: list indexed repositories as services
	repoRows, err := uc.pool.Query(ctx, `SELECT name, path FROM repositories ORDER BY name`)
	if err != nil {
		return "No services found. Run 'goatlas index <repo-path>' first.", nil
	}
	defer repoRows.Close()

	var repos []string
	for repoRows.Next() {
		var name, path string
		if err := repoRows.Scan(&name, &path); err != nil {
			continue
		}
		repos = append(repos, name+" ("+path+")")
	}

	if len(repos) == 0 {
		return "No services found. Run 'goatlas index <repo-path>' first.", nil
	}
	return "Indexed repositories (run build_graph to detect service connections):\n\n" + strings.Join(repos, "\n"), nil
}
