-- +goose Up

CREATE TABLE files (
    id          BIGSERIAL PRIMARY KEY,
    path        TEXT UNIQUE NOT NULL,
    module      TEXT,
    hash        TEXT,
    last_scanned TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE symbols (
    id              BIGSERIAL PRIMARY KEY,
    file_id         BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    kind            TEXT NOT NULL,
    name            TEXT NOT NULL,
    qualified_name  TEXT,
    signature       TEXT,
    receiver        TEXT,
    line            INT,
    col             INT,
    doc_comment     TEXT,
    embedded_at     TIMESTAMPTZ,
    search_vector   TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(name,'') || ' ' ||
            coalesce(qualified_name,'') || ' ' ||
            coalesce(signature,'') || ' ' ||
            coalesce(doc_comment,'')
        )
    ) STORED
);

CREATE INDEX idx_symbols_search ON symbols USING GIN(search_vector);
CREATE INDEX idx_symbols_file_id ON symbols(file_id);
CREATE INDEX idx_symbols_kind ON symbols(kind);

CREATE TABLE api_endpoints (
    id           BIGSERIAL PRIMARY KEY,
    file_id      BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    method       TEXT,
    path         TEXT,
    handler_name TEXT,
    framework    TEXT,
    line         INT
);

CREATE TABLE imports (
    id          BIGSERIAL PRIMARY KEY,
    file_id     BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    import_path TEXT NOT NULL,
    alias       TEXT
);

-- +goose Down
DROP TABLE imports;
DROP TABLE api_endpoints;
DROP TABLE symbols;
DROP TABLE files;
