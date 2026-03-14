-- +goose Up
-- Fix column name mismatch: migration 003 used caller_name/caller_qualified/callee_qualified
-- but the Go code (FunctionCallRepo.BulkInsert) uses caller_qualified_name/callee_package/col.
-- This caused COPY FROM to silently fail, resulting in an empty table.
DROP TABLE IF EXISTS function_calls;
CREATE TABLE function_calls (
    id                    BIGSERIAL PRIMARY KEY,
    file_id               BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    caller_qualified_name TEXT NOT NULL,
    callee_name           TEXT NOT NULL,
    callee_package        TEXT,
    line                  INT,
    col                   INT
);

CREATE INDEX IF NOT EXISTS idx_fc_file   ON function_calls(file_id);
CREATE INDEX IF NOT EXISTS idx_fc_caller ON function_calls(caller_qualified_name);
CREATE INDEX IF NOT EXISTS idx_fc_callee ON function_calls(callee_name);

-- +goose Down
-- Restore original schema from migration 003.
DROP TABLE IF EXISTS function_calls;
CREATE TABLE function_calls (
    id               BIGSERIAL PRIMARY KEY,
    file_id          BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    caller_name      TEXT NOT NULL,
    caller_qualified TEXT NOT NULL,
    callee_name      TEXT NOT NULL,
    callee_qualified TEXT,
    line             INT
);

CREATE INDEX IF NOT EXISTS idx_fc_file   ON function_calls(file_id);
CREATE INDEX IF NOT EXISTS idx_fc_caller ON function_calls(caller_qualified);
CREATE INDEX IF NOT EXISTS idx_fc_callee ON function_calls(callee_name);
