package domain

import "context"

// FunctionCall represents a detected call from one function to another.
type FunctionCall struct {
	ID                  int64
	FileID              int64
	CallerQualifiedName string // e.g. "handler.(ReceiveHandler).Handle"
	CalleeName          string // e.g. "ProcessReceive"
	CalleePackage       string // e.g. "service" (import alias or pkg name)
	Line                int
	Col                 int
}

// FunctionCallRepository handles persistence of function call records.
type FunctionCallRepository interface {
	BulkInsert(ctx context.Context, calls []FunctionCall) error
	DeleteByFileID(ctx context.Context, fileID int64) error
}
