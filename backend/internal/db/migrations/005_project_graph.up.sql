ALTER TABLE projects ADD COLUMN file_tree_json TEXT DEFAULT '';
ALTER TABLE projects ADD COLUMN permission_level TEXT DEFAULT 'sandboxed' CHECK(permission_level IN ('read_only', 'sandboxed', 'trusted'));

CREATE TABLE IF NOT EXISTS code_graph_edges (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id),
    from_symbol TEXT,
    to_symbol TEXT,
    edge_type TEXT CHECK(edge_type IN ('imports', 'calls')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_graph_project ON code_graph_edges(project_id);
