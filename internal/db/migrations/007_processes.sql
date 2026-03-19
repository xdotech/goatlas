-- +goose Up
CREATE TABLE processes (
    id          BIGSERIAL PRIMARY KEY,
    repo_id     BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    entry_point TEXT NOT NULL,
    file_path   TEXT,
    computed_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_proc_repo ON processes(repo_id);
CREATE INDEX IF NOT EXISTS idx_proc_entry ON processes(entry_point);

CREATE TABLE process_steps (
    id          BIGSERIAL PRIMARY KEY,
    process_id  BIGINT NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
    step_order  INT NOT NULL,
    symbol_name TEXT NOT NULL,
    file_path   TEXT,
    line        INT
);

CREATE INDEX IF NOT EXISTS idx_ps_process ON process_steps(process_id);

-- +goose Down
DROP TABLE IF EXISTS process_steps;
DROP TABLE IF EXISTS processes;
