CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    profile_md TEXT DEFAULT '',
    default_provider TEXT DEFAULT '',
    default_model TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    name TEXT NOT NULL,
    root_path TEXT DEFAULT '',
    instructions_md TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chats (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    project_id TEXT REFERENCES projects(id),
    name TEXT DEFAULT '',
    mode TEXT CHECK(mode IN ('chat','code')) DEFAULT 'chat',
    provider TEXT DEFAULT 'openai',
    model TEXT DEFAULT 'gpt-4o',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT REFERENCES chats(id),
    role TEXT CHECK(role IN ('user','assistant','tool')) NOT NULL,
    content TEXT DEFAULT '',
    attachments_json TEXT DEFAULT '[]',
    tool_calls_json TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    content=messages,
    content_rowid=rowid
);

CREATE TRIGGER IF NOT EXISTS messages_fts_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content) VALUES (new.rowid, new.content);
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content) VALUES('delete', old.rowid, old.content);
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content) VALUES('delete', old.rowid, old.content);
    INSERT INTO messages_fts(rowid, content) VALUES (new.rowid, new.content);
END;

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    trigger_pattern TEXT DEFAULT '',
    execution_type TEXT CHECK(execution_type IN ('script','prompt_template','api_call')) DEFAULT 'prompt_template',
    config_json TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS connectors (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id),
    name TEXT NOT NULL,
    type TEXT DEFAULT 'custom',
    endpoint TEXT DEFAULT '',
    auth_config_json TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id),
    chat_id TEXT REFERENCES chats(id),
    goal TEXT NOT NULL,
    plan_json TEXT DEFAULT '[]',
    status TEXT CHECK(status IN ('planning','running','completed','failed','cancelled')) DEFAULT 'planning',
    permission_level TEXT CHECK(permission_level IN ('read_only','sandboxed','trusted')) DEFAULT 'sandboxed',
    summary TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE TABLE IF NOT EXISTS agent_actions (
    id TEXT PRIMARY KEY,
    task_id TEXT REFERENCES agent_tasks(id),
    action_type TEXT CHECK(action_type IN ('file_read','file_write','command_exec','browser_action')) NOT NULL,
    target TEXT DEFAULT '',
    details_json TEXT DEFAULT '{}',
    requires_confirmation BOOLEAN DEFAULT 0,
    confirmed_by_user BOOLEAN DEFAULT 0,
    result TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
