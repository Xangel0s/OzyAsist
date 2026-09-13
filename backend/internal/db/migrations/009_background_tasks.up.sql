-- 009_background_tasks.up.sql

PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS agent_actions;
DROP TABLE IF EXISTS agent_tasks;

CREATE TABLE IF NOT EXISTS agent_tasks (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    project_id TEXT,
    title TEXT NOT NULL,
    prompt TEXT NOT NULL,
    status TEXT NOT NULL CHECK(status IN ('pending', 'planning', 'running', 'blocked_approval', 'completed', 'failed', 'cancelled')),
    total_steps INTEGER DEFAULT 0,
    current_step INTEGER DEFAULT 0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_steps (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    step_order INTEGER NOT NULL,
    agent_assigned TEXT NOT NULL DEFAULT 'ozy' CHECK(agent_assigned IN ('ozy', 'charc', 'nine')),
    action_type TEXT NOT NULL, -- 'shell_exec', 'mcp_call', 'cdp_action', 'file_patch', 'ble_write', 'google_draft'
    payload TEXT NOT NULL,     -- JSON estructurado con parámetros
    requires_pin BOOLEAN DEFAULT 0,
    pin_authorized BOOLEAN DEFAULT 0,
    verification_rule TEXT,    -- JSON: {"type": "exit_code", "expected": 0} o {"type": "file_exists", "path": "..."}
    status TEXT NOT NULL CHECK(status IN ('pending', 'running', 'verifying', 'recovering', 'completed', 'failed')),
    output TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY(task_id) REFERENCES agent_tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_tasks_status ON agent_tasks(status);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_user ON agent_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_task_steps_order ON task_steps(task_id, step_order);

PRAGMA foreign_keys = ON;
