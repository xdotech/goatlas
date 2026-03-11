-- +goose Up
-- Component-to-API call mapping for frontend↔backend cross-reference.
CREATE TABLE IF NOT EXISTS component_api_calls (
    id              BIGSERIAL PRIMARY KEY,
    file_id         BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    component       TEXT NOT NULL,
    http_method     TEXT,
    api_path        TEXT NOT NULL,
    target_service  TEXT,
    line            INT,
    col             INT
);

CREATE INDEX IF NOT EXISTS idx_cac_file      ON component_api_calls(file_id);
CREATE INDEX IF NOT EXISTS idx_cac_component ON component_api_calls(component);
CREATE INDEX IF NOT EXISTS idx_cac_api       ON component_api_calls(api_path);

-- +goose Down
DROP TABLE IF EXISTS component_api_calls;
