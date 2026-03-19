package domain

import (
	"context"
	"time"
)

// Community represents a cluster of highly-interconnected code symbols detected via Louvain.
type Community struct {
	ID          int64
	RepoID      int64
	CommunityID int
	Name        string
	MemberCount int
	ComputedAt  time.Time
}

// CommunityMember represents a symbol belonging to a community.
type CommunityMember struct {
	ID          int64
	CommunityID int64
	SymbolName  string
	FilePath    string
}

// CommunityRepository handles persistence of community records.
type CommunityRepository interface {
	Insert(ctx context.Context, c *Community) (int64, error)
	InsertMembers(ctx context.Context, members []CommunityMember) error
	DeleteByRepoID(ctx context.Context, repoID int64) error
	List(ctx context.Context, repoID int64) ([]Community, error)
	GetMembers(ctx context.Context, communityDBID int64) ([]CommunityMember, error)
}
