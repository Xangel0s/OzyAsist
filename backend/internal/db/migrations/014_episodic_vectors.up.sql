CREATE TABLE IF NOT EXISTS episodic_vectors (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    project_id TEXT DEFAULT '',
    source TEXT NOT NULL DEFAULT 'chat',
    source_id TEXT DEFAULT '',
    content TEXT NOT NULL,
    vector_json TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_episodic_vectors_user ON episodic_vectors(user_id);
CREATE INDEX IF NOT EXISTS idx_episodic_vectors_project ON episodic_vectors(project_id);
