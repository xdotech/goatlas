package postgres

import (
	"context"

	"github.com/xdotech/goatlas/internal/indexer/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommunityRepo implements domain.CommunityRepository using PostgreSQL.
type CommunityRepo struct {
	pool *pgxpool.Pool
}

// NewCommunityRepo creates a new CommunityRepo.
func NewCommunityRepo(pool *pgxpool.Pool) *CommunityRepo {
	return &CommunityRepo{pool: pool}
}

// Insert creates a community record and returns its ID.
func (r *CommunityRepo) Insert(ctx context.Context, c *domain.Community) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO communities (repo_id, community_id, name, member_count) VALUES ($1, $2, $3, $4) RETURNING id`,
		c.RepoID, c.CommunityID, c.Name, c.MemberCount,
	).Scan(&id)
	return id, err
}

// InsertMembers bulk-inserts community members using COPY protocol.
func (r *CommunityRepo) InsertMembers(ctx context.Context, members []domain.CommunityMember) error {
	if len(members) == 0 {
		return nil
	}
	rows := make([][]interface{}, len(members))
	for i, m := range members {
		rows[i] = []interface{}{m.CommunityID, m.SymbolName, m.FilePath}
	}
	_, err := r.pool.CopyFrom(ctx,
		pgx.Identifier{"community_members"},
		[]string{"community_id", "symbol_name", "file_path"},
		pgx.CopyFromRows(rows),
	)
	return err
}

// DeleteByRepoID removes all communities (and cascading members) for a repo.
func (r *CommunityRepo) DeleteByRepoID(ctx context.Context, repoID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM communities WHERE repo_id = $1`, repoID)
	return err
}

// List returns all communities for a given repository.
func (r *CommunityRepo) List(ctx context.Context, repoID int64) ([]domain.Community, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, repo_id, community_id, name, member_count, computed_at FROM communities WHERE repo_id = $1 ORDER BY member_count DESC`,
		repoID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comms []domain.Community
	for rows.Next() {
		var c domain.Community
		if err := rows.Scan(&c.ID, &c.RepoID, &c.CommunityID, &c.Name, &c.MemberCount, &c.ComputedAt); err != nil {
			return nil, err
		}
		comms = append(comms, c)
	}
	return comms, rows.Err()
}

// GetMembers returns all members for a given community (by DB ID).
func (r *CommunityRepo) GetMembers(ctx context.Context, communityDBID int64) ([]domain.CommunityMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, community_id, symbol_name, file_path FROM community_members WHERE community_id = $1 ORDER BY symbol_name`,
		communityDBID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.CommunityMember
	for rows.Next() {
		var m domain.CommunityMember
		if err := rows.Scan(&m.ID, &m.CommunityID, &m.SymbolName, &m.FilePath); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
