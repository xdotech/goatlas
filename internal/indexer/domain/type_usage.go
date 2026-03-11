package domain

import "context"

// TypeUsage represents a function's use of a named type as input or output.
type TypeUsage struct {
	ID         int64
	FileID     int64
	SymbolName string // qualified function name
	TypeName   string // the type being used (e.g. "CreateOrderRequest")
	Direction  string // "input", "output", "internal"
	Position   int    // parameter position (0-based)
	Line       int
}

// TypeUsageRepository handles persistence of type usage records.
type TypeUsageRepository interface {
	BulkInsert(ctx context.Context, usages []TypeUsage) error
	DeleteByFileID(ctx context.Context, fileID int64) error
	FindByType(ctx context.Context, typeName string) ([]TypeUsage, error)
	FindBySymbol(ctx context.Context, symbolName string) ([]TypeUsage, error)
}
