-- backend/internal/db/migrations/012_security_and_audit.up.sql

-- Registro inmutable de auditoría con encadenamiento criptográfico (Hash Chaining)
CREATE TABLE IF NOT EXISTS audit_trail_chained (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    agent TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT NOT NULL,
    prev_hash TEXT NOT NULL,
    record_hash TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_agent ON audit_trail_chained(agent);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_trail_chained(action);

-- Snapshots para Time-Travel y Rollback de Archivos
CREATE TABLE IF NOT EXISTS shadow_snapshots (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    content_before BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(task_id) REFERENCES agent_tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_snapshots_task ON shadow_snapshots(task_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_path ON shadow_snapshots(file_path);
