-- +goose Up

-- Repositories table: tracks indexed repos
CREATE TABLE repositories (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    path            TEXT NOT NULL,
    last_indexed_at TIMESTAMPTZ,
    last_commit     TEXT
);

-- Files table: indexed source files scoped by repo
CREATE TABLE files (
    id           BIGSERIAL PRIMARY KEY,
    repo_id      BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    path         TEXT NOT NULL,
    module       TEXT,
    hash         TEXT,
    last_scanned TIMESTAMPTZ DEFAULT now(),
    UNIQUE (repo_id, path)
);

CREATE INDEX idx_files_repo_id ON files(repo_id);

-- Symbols table: AST-extracted code symbols
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

-- API endpoints table
CREATE TABLE api_endpoints (
    id           BIGSERIAL PRIMARY KEY,
    file_id      BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    method       TEXT,
    path         TEXT,
    handler_name TEXT,
    framework    TEXT,
    line         INT
);

-- Import statements
CREATE TABLE imports (
    id          BIGSERIAL PRIMARY KEY,
    file_id     BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    import_path TEXT NOT NULL,
    alias       TEXT
);

-- pgvector for semantic search
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE symbol_embeddings (
    symbol_id  BIGINT PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
    embedding  vector(768) NOT NULL,
    file       TEXT,
    kind       TEXT,
    name       TEXT,
    service    TEXT,
    line       INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_symbol_embeddings_hnsw ON symbol_embeddings USING hnsw (embedding vector_cosine_ops);

-- Service connections: cross-service dependencies (gRPC, Kafka)
CREATE TABLE service_connections (
    id        BIGSERIAL PRIMARY KEY,
    repo_id   BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    conn_type TEXT NOT NULL,  -- 'grpc' | 'kafka_publish' | 'kafka_consume'
    target    TEXT NOT NULL,  -- proto client name or topic name
    file_id   BIGINT REFERENCES files(id) ON DELETE CASCADE,
    line      INT
);

CREATE INDEX idx_service_connections_repo ON service_connections(repo_id);
CREATE INDEX idx_service_connections_type ON service_connections(conn_type);
CREATE INDEX idx_service_connections_target ON service_connections(target);

-- +goose Down
DROP TABLE IF EXISTS service_connections;
DROP TABLE IF EXISTS symbol_embeddings;
DROP EXTENSION IF EXISTS vector;
DROP TABLE IF EXISTS imports;
DROP TABLE IF EXISTS api_endpoints;
DROP TABLE IF EXISTS symbols;
DROP TABLE IF EXISTS files;
DROP TABLE IF EXISTS repositories;
