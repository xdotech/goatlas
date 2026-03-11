-- Function call graph for change impact analysis.
CREATE TABLE IF NOT EXISTS function_calls (
    id                    BIGSERIAL PRIMARY KEY,
    file_id               BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    caller_qualified_name TEXT NOT NULL,
    callee_name           TEXT NOT NULL,
    callee_package        TEXT,
    line                  INT,
    col                   INT
);

CREATE INDEX IF NOT EXISTS idx_fc_file    ON function_calls(file_id);
CREATE INDEX IF NOT EXISTS idx_fc_caller  ON function_calls(caller_qualified_name);
CREATE INDEX IF NOT EXISTS idx_fc_callee  ON function_calls(callee_name);
