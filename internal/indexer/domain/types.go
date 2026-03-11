package domain

import (
	"context"
	"time"
)

// Repository represents an indexed code repository.
type Repository struct {
	ID            int64
	Name          string
	Path          string
	LastIndexedAt *time.Time
	LastCommit    string
}

// File represents an indexed source file, scoped to a repository.
type File struct {
	ID          int64
	RepoID      int64
	Path        string
	Module      string
	Hash        string
	LastScanned time.Time
}

// Symbol represents a code symbol extracted from AST parsing.
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

// SymbolWithFile extends Symbol with file path for snippet extraction.
type SymbolWithFile struct {
	Symbol
	FilePath string // relative path from repo root
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

// Import represents an import statement.
type Import struct {
	ID         int64
	FileID     int64
	ImportPath string
	Alias      string
}

// ServiceConnection represents a cross-service dependency (gRPC call or Kafka pub/sub).
type ServiceConnection struct {
	ID       int64
	RepoID   int64
	ConnType string // "grpc", "kafka_publish", "kafka_consume"
	Target   string // proto client name or topic name
	FileID   int64
	Line     int
}

// RepositoryRepository handles persistence of repository records.
type RepositoryRepository interface {
	Upsert(ctx context.Context, r *Repository) error
	GetByName(ctx context.Context, name string) (*Repository, error)
	List(ctx context.Context) ([]Repository, error)
}

// FileRepository handles persistence of file records.
type FileRepository interface {
	Upsert(ctx context.Context, f *File) error
	GetByPath(ctx context.Context, repoID int64, path string) (*File, error)
	DeleteByID(ctx context.Context, id int64) error
}

// SymbolRepository handles persistence and search of symbols.
type SymbolRepository interface {
	BulkInsert(ctx context.Context, symbols []Symbol) error
	DeleteByFileID(ctx context.Context, fileID int64) error
	Search(ctx context.Context, query string, limit int, kind string) ([]Symbol, error)
	SearchWithFile(ctx context.Context, query string, limit int, kind string) ([]SymbolWithFile, error)
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

// ServiceConnectionRepository handles persistence of cross-service connections.
type ServiceConnectionRepository interface {
	BulkInsert(ctx context.Context, conns []ServiceConnection) error
	DeleteByRepoID(ctx context.Context, repoID int64) error
	List(ctx context.Context, connType string) ([]ServiceConnection, error)
	ListByRepo(ctx context.Context, repoID int64) ([]ServiceConnection, error)
}
