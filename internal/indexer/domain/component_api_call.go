package domain

import "context"

// ComponentAPICall represents a detected API call from a frontend component.
type ComponentAPICall struct {
	ID            int64
	FileID        int64
	Component     string // React component name, e.g. "UserListPage"
	HttpMethod    string // GET, POST, PUT, DELETE, etc.
	APIPath       string // /api/v1/users, {SVC_PREFIX}/items
	TargetService string // resolved backend service name
	Line          int
	Col           int
}

// ComponentAPICallRepository handles persistence of component API call records.
type ComponentAPICallRepository interface {
	BulkInsert(ctx context.Context, calls []ComponentAPICall) error
	DeleteByFileID(ctx context.Context, fileID int64) error
	FindByComponent(ctx context.Context, component string) ([]ComponentAPICall, error)
	FindByAPIPath(ctx context.Context, apiPath string) ([]ComponentAPICall, error)
}
