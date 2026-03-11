-- +goose Up
-- Type usage tracking for type flow analysis.
CREATE TABLE IF NOT EXISTS type_usages (
    id          BIGSERIAL PRIMARY KEY,
    file_id     BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    symbol_name TEXT NOT NULL,
    type_name   TEXT NOT NULL,
    direction   TEXT NOT NULL,  -- 'input' | 'output' | 'internal'
    position    INT DEFAULT 0,
    line        INT
);

CREATE INDEX IF NOT EXISTS idx_tu_file   ON type_usages(file_id);
CREATE INDEX IF NOT EXISTS idx_tu_symbol ON type_usages(symbol_name);
CREATE INDEX IF NOT EXISTS idx_tu_type   ON type_usages(type_name);

-- +goose Down
DROP TABLE IF EXISTS type_usages;
