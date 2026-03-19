package postgres

import (
	"context"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProcessRepo implements domain.ProcessRepository using PostgreSQL.
type ProcessRepo struct {
	pool *pgxpool.Pool
}

// NewProcessRepo creates a new ProcessRepo.
func NewProcessRepo(pool *pgxpool.Pool) *ProcessRepo {
	return &ProcessRepo{pool: pool}
}

// Insert creates a process record and returns its ID.
func (r *ProcessRepo) Insert(ctx context.Context, p *domain.Process) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO processes (repo_id, name, entry_point, file_path) VALUES ($1, $2, $3, $4) RETURNING id`,
		p.RepoID, p.Name, p.EntryPoint, p.FilePath,
	).Scan(&id)
	return id, err
}

// InsertSteps bulk-inserts process steps using COPY protocol.
func (r *ProcessRepo) InsertSteps(ctx context.Context, steps []domain.ProcessStep) error {
	if len(steps) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(steps))
	for i, s := range steps {
		rows[i] = []interface{}{s.ProcessID, s.StepOrder, s.SymbolName, s.FilePath, s.Line}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"process_steps"},
		[]string{"process_id", "step_order", "symbol_name", "file_path", "line"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByRepoID removes all processes (and cascading steps) for a repo.
func (r *ProcessRepo) DeleteByRepoID(ctx context.Context, repoID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM processes WHERE repo_id = $1`, repoID)
	return err
}

// List returns all processes for a given repository.
func (r *ProcessRepo) List(ctx context.Context, repoID int64) ([]domain.Process, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, repo_id, name, entry_point, file_path, computed_at FROM processes WHERE repo_id = $1 ORDER BY name`,
		repoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var procs []domain.Process
	for rows.Next() {
		var p domain.Process
		if err := rows.Scan(&p.ID, &p.RepoID, &p.Name, &p.EntryPoint, &p.FilePath, &p.ComputedAt); err != nil {
			return nil, err
		}
		procs = append(procs, p)
	}
	return procs, rows.Err()
}

// GetSteps returns all steps for a given process.
func (r *ProcessRepo) GetSteps(ctx context.Context, processID int64) ([]domain.ProcessStep, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, process_id, step_order, symbol_name, file_path, line FROM process_steps WHERE process_id = $1 ORDER BY step_order`,
		processID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []domain.ProcessStep
	for rows.Next() {
		var s domain.ProcessStep
		if err := rows.Scan(&s.ID, &s.ProcessID, &s.StepOrder, &s.SymbolName, &s.FilePath, &s.Line); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

// GetByName returns a process by name within a repository.
func (r *ProcessRepo) GetByName(ctx context.Context, repoID int64, name string) (*domain.Process, error) {
	var p domain.Process
	err := r.pool.QueryRow(ctx,
		`SELECT id, repo_id, name, entry_point, file_path, computed_at FROM processes WHERE repo_id = $1 AND name = $2`,
		repoID, name,
	).Scan(&p.ID, &p.RepoID, &p.Name, &p.EntryPoint, &p.FilePath, &p.ComputedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
