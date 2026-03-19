-- +goose Up
CREATE TABLE communities (
    id           BIGSERIAL PRIMARY KEY,
    repo_id      BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    community_id INT NOT NULL,
    name         TEXT,
    member_count INT DEFAULT 0,
    computed_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comm_repo ON communities(repo_id);

CREATE TABLE community_members (
    id           BIGSERIAL PRIMARY KEY,
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    symbol_name  TEXT NOT NULL,
    file_path    TEXT
);

CREATE INDEX IF NOT EXISTS idx_cm_community ON community_members(community_id);

-- +goose Down
DROP TABLE IF EXISTS community_members;
DROP TABLE IF EXISTS communities;
