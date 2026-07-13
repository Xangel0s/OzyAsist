DROP INDEX IF EXISTS idx_graph_project;
DROP TABLE IF EXISTS code_graph_edges;
ALTER TABLE projects DROP COLUMN file_tree_json;
ALTER TABLE projects DROP COLUMN permission_level;
