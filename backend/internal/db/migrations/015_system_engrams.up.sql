CREATE TABLE IF NOT EXISTS system_engrams (
    id TEXT PRIMARY KEY,
    use_case TEXT NOT NULL,
    trigger_words_json TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    default_args_json TEXT NOT NULL,
    co_occurring_tools_json TEXT DEFAULT '[]',
    fast_track INTEGER NOT NULL DEFAULT 1,
    destructive INTEGER NOT NULL DEFAULT 0,
    confidence REAL NOT NULL DEFAULT 0.95,
    maturity TEXT NOT NULL DEFAULT 'reflex',
    success_count INTEGER NOT NULL DEFAULT 1,
    feedback_templates_json TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_system_engrams_use_case ON system_engrams(use_case);
CREATE INDEX IF NOT EXISTS idx_system_engrams_fast_track ON system_engrams(fast_track);
