package domain

import "context"

// InterfaceImpl represents a detected interface-struct implementation relationship.
type InterfaceImpl struct {
	ID            int64
	FileID        int64
	InterfaceName string // e.g. "CompetitorKeywordRepository"
	StructName    string // e.g. "competitorKeywordRepo"
	MethodName    string // e.g. "FindKeywordsByCompetitorId"
}

// InterfaceImplRepository handles persistence of interface implementation records.
type InterfaceImplRepository interface {
	BulkInsert(ctx context.Context, impls []InterfaceImpl) error
	DeleteByFileID(ctx context.Context, fileID int64) error
	// FindImplementations returns struct methods that implement the given interface method.
	FindImplementations(ctx context.Context, interfaceName, methodName string) ([]InterfaceImpl, error)
}
