CREATE INDEX IF NOT EXISTS idx_messages_chat_id_created ON messages(chat_id, created_at);
CREATE INDEX IF NOT EXISTS idx_memory_entries_user_created ON memory_entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_skills_user_created ON skills(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_connectors_user_created ON connectors(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chats_created ON chats(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_projects_created ON projects(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_tasks_chat ON agent_tasks(chat_id);
CREATE INDEX IF NOT EXISTS idx_agent_actions_task ON agent_actions(task_id);
CREATE INDEX IF NOT EXISTS idx_code_graph_edges_project ON code_graph_edges(project_id, from_symbol, to_symbol);
