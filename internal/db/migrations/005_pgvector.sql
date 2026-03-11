-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS symbol_embeddings;
DROP EXTENSION IF EXISTS vector;
