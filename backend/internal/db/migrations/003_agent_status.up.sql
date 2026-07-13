PRAGMA foreign_keys = OFF;

CREATE TABLE IF NOT EXISTS agent_tasks_new (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id),
    chat_id TEXT REFERENCES chats(id),
    goal TEXT NOT NULL,
    plan_json TEXT DEFAULT '[]',
    status TEXT CHECK(status IN ('planning','running','awaiting_confirmation','completed','failed','cancelled')) DEFAULT 'planning',
    permission_level TEXT CHECK(permission_level IN ('read_only','sandboxed','trusted')) DEFAULT 'sandboxed',
    summary TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

INSERT INTO agent_tasks_new SELECT * FROM agent_tasks;
DROP TABLE agent_tasks;
ALTER TABLE agent_tasks_new RENAME TO agent_tasks;

PRAGMA foreign_keys = ON;
