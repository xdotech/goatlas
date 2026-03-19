package domain

import (
	"context"
	"time"
)

// Process represents a detected execution flow from an entry point through a call chain.
type Process struct {
	ID         int64
	RepoID     int64
	Name       string
	EntryPoint string // qualified function name
	FilePath   string
	ComputedAt time.Time
}

// ProcessStep represents a single step in a process execution flow.
type ProcessStep struct {
	ID         int64
	ProcessID  int64
	StepOrder  int
	SymbolName string
	FilePath   string
	Line       int
}

// ProcessRepository handles persistence of process records.
type ProcessRepository interface {
	Insert(ctx context.Context, p *Process) (int64, error)
	InsertSteps(ctx context.Context, steps []ProcessStep) error
	DeleteByRepoID(ctx context.Context, repoID int64) error
	List(ctx context.Context, repoID int64) ([]Process, error)
	GetSteps(ctx context.Context, processID int64) ([]ProcessStep, error)
	GetByName(ctx context.Context, repoID int64, name string) (*Process, error)
}
