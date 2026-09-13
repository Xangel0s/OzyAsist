CREATE TABLE IF NOT EXISTS mcp_connectors (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    command TEXT NOT NULL,
    args TEXT NOT NULL, /* JSON array */
    env TEXT NOT NULL, /* JSON array */
    status TEXT DEFAULT 'disconnected',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mcp_connectors_user ON mcp_connectors(user_id);
