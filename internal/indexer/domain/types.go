package domain

import (
	"context"
	"time"
)

// File represents an indexed Go source file.
type File struct {
	ID          int64
	Path        string
	Module      string
	Hash        string
	LastScanned time.Time
}

// Symbol represents a Go code symbol extracted from AST parsing.
type Symbol struct {
	ID            int64
	FileID        int64
	Kind          string
	Name          string
	QualifiedName string
	Signature     string
	Receiver      string
	Line          int
	Col           int
	DocComment    string
}

// APIEndpoint represents an HTTP route detected in source code.
type APIEndpoint struct {
	ID          int64
	FileID      int64
	Method      string
	Path        string
	HandlerName string
	Framework   string
	Line        int
}

// Import represents a Go import statement.
type Import struct {
	ID         int64
	FileID     int64
	ImportPath string
	Alias      string
}

// FileRepository handles persistence of file records.
type FileRepository interface {
	Upsert(ctx context.Context, f *File) error
	GetByPath(ctx context.Context, path string) (*File, error)
	DeleteByID(ctx context.Context, id int64) error
}

// SymbolRepository handles persistence and search of symbols.
type SymbolRepository interface {
	BulkInsert(ctx context.Context, symbols []Symbol) error
	DeleteByFileID(ctx context.Context, fileID int64) error
	Search(ctx context.Context, query string, limit int, kind string) ([]Symbol, error)
	GetByFile(ctx context.Context, fileID int64) ([]Symbol, error)
	ListByKinds(ctx context.Context, kinds []string, limit int) ([]Symbol, error)
}

// EndpointRepository handles persistence of API endpoint records.
type EndpointRepository interface {
	BulkInsert(ctx context.Context, endpoints []APIEndpoint) error
	DeleteByFileID(ctx context.Context, fileID int64) error
	List(ctx context.Context, method, service string) ([]APIEndpoint, error)
}

// ImportRepository handles persistence of import records.
type ImportRepository interface {
	BulkInsert(ctx context.Context, imports []Import) error
	DeleteByFileID(ctx context.Context, fileID int64) error
}
