-- Migration 001: initial schema.
--
-- Creates the pgvector extension, the chunks table, and supporting
-- indexes. Renders {{.Table}} and {{.Dims}} from the per-deployment
-- config passed to Migrate().

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE {{.Table}} (
    id           TEXT PRIMARY KEY,
    kb_id        TEXT NOT NULL,
    document_id  TEXT NOT NULL,
    content      TEXT NOT NULL,
    chunk_index  INT  NOT NULL,
    metadata     JSONB,
    model        TEXT,
    embedding    vector({{.Dims}}),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX {{.Table}}_kb_idx   ON {{.Table}} (kb_id);
CREATE INDEX {{.Table}}_hnsw_idx ON {{.Table}} USING hnsw (embedding vector_cosine_ops);
