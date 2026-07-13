CREATE TABLE IF NOT EXISTS memory_entries (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    project_id TEXT REFERENCES projects(id),
    source TEXT NOT NULL DEFAULT 'chat',
    source_id TEXT DEFAULT '',
    topic TEXT DEFAULT '',
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS memory_entries_fts USING fts5(
    topic,
    content,
    content=memory_entries,
    content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS memory_entries_fts_insert AFTER INSERT ON memory_entries BEGIN
    INSERT INTO memory_entries_fts(rowid, topic, content) VALUES (new.rowid, new.topic, new.content);
END;

CREATE TRIGGER IF NOT EXISTS memory_entries_fts_delete AFTER DELETE ON memory_entries BEGIN
    INSERT INTO memory_entries_fts(memory_entries_fts, rowid, topic, content) VALUES('delete', old.rowid, old.topic, old.content);
END;

CREATE TRIGGER IF NOT EXISTS memory_entries_fts_update AFTER UPDATE ON memory_entries BEGIN
    INSERT INTO memory_entries_fts(memory_entries_fts, rowid, topic, content) VALUES('delete', old.rowid, old.topic, old.content);
    INSERT INTO memory_entries_fts(rowid, topic, content) VALUES (new.rowid, new.topic, new.content);
END;
