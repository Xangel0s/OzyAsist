-- backend/internal/db/migrations/011_user_memories_fts.up.sql

CREATE TABLE IF NOT EXISTS user_memories (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    category TEXT NOT NULL CHECK(category IN ('preferencia', 'stack', 'hardware', 'regla', 'contexto')),
    content TEXT NOT NULL,
    confidence REAL DEFAULT 1.0,
    access_count INTEGER DEFAULT 1,
    last_recalled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_memories_user_cat ON user_memories(user_id, category);

-- Tabla virtual FTS5 para búsqueda léxica BM25
CREATE VIRTUAL TABLE IF NOT EXISTS user_memories_fts USING fts5(
    id UNINDEXED,
    content,
    category,
    tokenize = 'porter unicode61'
);

-- Triggers de sincronización automática entre user_memories y user_memories_fts
CREATE TRIGGER IF NOT EXISTS trg_user_memories_ai AFTER INSERT ON user_memories BEGIN
    INSERT INTO user_memories_fts(id, content, category) VALUES (new.id, new.content, new.category);
END;

CREATE TRIGGER IF NOT EXISTS trg_user_memories_ad AFTER DELETE ON user_memories BEGIN
    DELETE FROM user_memories_fts WHERE id = old.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_user_memories_au AFTER UPDATE ON user_memories BEGIN
    DELETE FROM user_memories_fts WHERE id = old.id;
    INSERT INTO user_memories_fts(id, content, category) VALUES (new.id, new.content, new.category);
END;
