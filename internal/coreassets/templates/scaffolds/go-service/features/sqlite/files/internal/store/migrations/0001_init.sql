-- The initial schema. A settings KV table — a common daemon configuration store.
-- Replace this with your own tables; add later changes as new NNNN_name.sql files.
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
