-- +goose Up
-- Interface implementation tracking for interface-aware caller resolution.
CREATE TABLE IF NOT EXISTS interface_impls (
    id              BIGSERIAL PRIMARY KEY,
    file_id         BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    interface_name  TEXT NOT NULL,  -- e.g. "CompetitorKeywordRepository"
    struct_name     TEXT NOT NULL,  -- e.g. "competitorKeywordRepo"
    method_name     TEXT NOT NULL   -- e.g. "FindKeywordsByCompetitorId"
);

CREATE INDEX IF NOT EXISTS idx_ii_file      ON interface_impls(file_id);
CREATE INDEX IF NOT EXISTS idx_ii_interface ON interface_impls(interface_name);
CREATE INDEX IF NOT EXISTS idx_ii_struct    ON interface_impls(struct_name);

-- +goose Down
DROP TABLE IF EXISTS interface_impls;
