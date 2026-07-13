package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db/models"
)

var defaultUserID string

func EnsureDefaultUser() error {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count > 0 {
		row := DB.QueryRow("SELECT id FROM users LIMIT 1")
		return row.Scan(&defaultUserID)
	}

	id := uuid.NewString()
	_, err := DB.Exec(
		`INSERT INTO users (id, name, created_at) VALUES (?, ?, ?)`,
		id, "Usuario Local", time.Now())
	if err != nil {
		return err
	}
	defaultUserID = id
	return nil
}

func DefaultUserID() string {
	return defaultUserID
}

func UpdateUserProfile(userID, profileMd string) error {
	_, err := DB.Exec(`UPDATE users SET profile_md = ? WHERE id = ?`, profileMd, userID)
	return err
}

func CreateChat(c *models.Chat) error {
	_, err := DB.Exec(
		`INSERT INTO chats (id, user_id, project_id, name, mode, provider, model, created_at)
		 VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?)`,
		c.ID, c.UserID, c.ProjectID, c.Name, c.Mode, c.Provider, c.Model, c.CreatedAt)
	return err
}

func GetChat(chatID string) (*models.Chat, error) {
	row := DB.QueryRow(
		`SELECT id, user_id, COALESCE(project_id,''), COALESCE(name,''), mode, provider, model, created_at
		 FROM chats WHERE id = ?`, chatID)
	var c models.Chat
	err := row.Scan(&c.ID, &c.UserID, &c.ProjectID, &c.Name, &c.Mode, &c.Provider, &c.Model, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func ListChats() ([]models.Chat, error) {
	rows, err := DB.Query(
		`SELECT id, user_id, COALESCE(project_id,''), COALESCE(name,''), mode, provider, model, created_at
		 FROM chats ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var c models.Chat
		if err := rows.Scan(&c.ID, &c.UserID, &c.ProjectID, &c.Name, &c.Mode, &c.Provider, &c.Model, &c.CreatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

func CreateProject(p *models.Project) error {
	_, err := DB.Exec(
		`INSERT INTO projects (id, user_id, name, root_path, instructions_md, file_tree_json, permission_level, agent_consent, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Name, p.RootPath, p.InstructionsMd, p.FileTreeJSON, p.PermissionLevel, p.AgentConsent, p.CreatedAt)
	return err
}

func GetProject(id string) (*models.Project, error) {
	row := DB.QueryRow(
		`SELECT id, user_id, name, COALESCE(root_path,''), COALESCE(instructions_md,''), COALESCE(file_tree_json,''), COALESCE(permission_level,'sandboxed'), COALESCE(agent_consent,'ask'), created_at
		 FROM projects WHERE id = ?`, id)
	var p models.Project
	err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.RootPath, &p.InstructionsMd, &p.FileTreeJSON, &p.PermissionLevel, &p.AgentConsent, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func ListProjects() ([]models.Project, error) {
	rows, err := DB.Query(
		`SELECT id, user_id, name, COALESCE(root_path,''), COALESCE(instructions_md,''), COALESCE(file_tree_json,''), COALESCE(permission_level,'sandboxed'), COALESCE(agent_consent,'ask'), created_at
		 FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.RootPath, &p.InstructionsMd, &p.FileTreeJSON, &p.PermissionLevel, &p.AgentConsent, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func UpdateProject(id string, p *models.Project) error {
	_, err := DB.Exec(
		`UPDATE projects SET name = ?, root_path = ?, instructions_md = ?, file_tree_json = ?, permission_level = ?, agent_consent = ? WHERE id = ?`,
		p.Name, p.RootPath, p.InstructionsMd, p.FileTreeJSON, p.PermissionLevel, p.AgentConsent, id)
	return err
}

func UpdateProjectConsent(projectID, consent string) error {
	_, err := DB.Exec(`UPDATE projects SET agent_consent = ? WHERE id = ?`, consent, projectID)
	return err
}

func DeleteProject(id string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("UPDATE chats SET project_id = NULL WHERE project_id = ?", id)
	tx.Exec("UPDATE agent_tasks SET project_id = NULL WHERE project_id = ?", id)
	tx.Exec("DELETE FROM code_graph_edges WHERE project_id = ?", id)
	tx.Exec("DELETE FROM projects WHERE id = ?", id)
	return tx.Commit()
}

func DeleteChat(chatID string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM messages WHERE chat_id = ?", chatID)
	tx.Exec("DELETE FROM chats WHERE id = ?", chatID)
	return tx.Commit()
}

func CreateMessage(m *models.Message) error {
	_, err := DB.Exec(
		`INSERT INTO messages (id, chat_id, role, content, attachments_json, tool_calls_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.ChatID, m.Role, m.Content, m.AttachmentsJSON, m.ToolCallsJSON, m.CreatedAt)
	return err
}

type SearchResult struct {
	ID      string  `json:"id"`
	Kind    string  `json:"kind"` // "message", "chat", "project", "memory"
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// FTS5SearchMessages busca en messages_fts y devuelve los resultados con metadata.
func FTS5SearchMessages(query string, limit int) ([]SearchResult, error) {
	sql := `SELECT m.id, c.name, m.content, rank
		FROM messages_fts
		JOIN messages m ON messages_fts.rowid = m.rowid
		LEFT JOIN chats c ON m.chat_id = c.id
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?`

	rows, err := DB.Query(sql, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var title string
		if err := rows.Scan(&r.ID, &title, &r.Snippet, &r.Score); err != nil {
			return nil, err
		}
		r.Kind = "message"
		r.Title = title
		results = append(results, r)
	}
	return results, rows.Err()
}

// SearchChats busca chats por nombre (LIKE).
func SearchChats(query string, limit int) ([]SearchResult, error) {
	rows, err := DB.Query(
		`SELECT id, name, '' as snippet FROM chats WHERE name LIKE ? ORDER BY created_at DESC LIMIT ?`,
		"%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Snippet); err != nil {
			return nil, err
		}
		r.Kind = "chat"
		r.Score = 0.5
		results = append(results, r)
	}
	return results, rows.Err()
}

// SearchProjects busca proyectos por nombre (LIKE).
func SearchProjects(query string, limit int) ([]SearchResult, error) {
	rows, err := DB.Query(
		`SELECT id, name, COALESCE(root_path,'') FROM projects WHERE name LIKE ? ORDER BY created_at DESC LIMIT ?`,
		"%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Snippet); err != nil {
			return nil, err
		}
		r.Kind = "project"
		r.Score = 0.5
		results = append(results, r)
	}
	return results, rows.Err()
}

func CreateMemoryEntry(m *models.MemoryEntry) error {
	_, err := DB.Exec(
		`INSERT INTO memory_entries (id, user_id, project_id, source, source_id, topic, content, created_at)
		 VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?)`,
		m.ID, m.UserID, m.ProjectID, m.Source, m.SourceID, m.Topic, m.Content, m.CreatedAt)
	return err
}

func ListMemoryEntries(userID string, limit int) ([]models.MemoryEntry, error) {
	rows, err := DB.Query(
		`SELECT id, user_id, COALESCE(project_id,''), source, COALESCE(source_id,''), COALESCE(topic,''), content, created_at
		 FROM memory_entries WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`,
		userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []models.MemoryEntry
	for rows.Next() {
		var e models.MemoryEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.ProjectID, &e.Source, &e.SourceID, &e.Topic, &e.Content, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func FTS5SearchMemory(query string, limit int) ([]SearchResult, error) {
	sql := `SELECT m.id, m.topic, m.content, rank
		FROM memory_entries_fts
		JOIN memory_entries m ON memory_entries_fts.rowid = m.rowid
		WHERE memory_entries_fts MATCH ?
		ORDER BY rank
		LIMIT ?`

	rows, err := DB.Query(sql, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Title, &r.Snippet, &r.Score); err != nil {
			return nil, err
		}
		r.Kind = "memory"
		results = append(results, r)
	}
	return results, rows.Err()
}

func CreateSkill(s *models.Skill) error {
	_, err := DB.Exec(
		`INSERT INTO skills (id, user_id, name, description, trigger_pattern, execution_type, config_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.Name, s.Description, s.TriggerPattern, s.ExecutionType, s.ConfigJSON, s.CreatedAt)
	return err
}

func ListSkills(userID string) ([]models.Skill, error) {
	rows, err := DB.Query(
		`SELECT id, user_id, name, COALESCE(description,''), COALESCE(trigger_pattern,''), execution_type, COALESCE(config_json,'{}'), created_at
		 FROM skills WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []models.Skill
	for rows.Next() {
		var s models.Skill
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Description, &s.TriggerPattern, &s.ExecutionType, &s.ConfigJSON, &s.CreatedAt); err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, rows.Err()
}

func GetSkill(id string) (*models.Skill, error) {
	row := DB.QueryRow(
		`SELECT id, user_id, name, COALESCE(description,''), COALESCE(trigger_pattern,''), execution_type, COALESCE(config_json,'{}'), created_at
		 FROM skills WHERE id = ?`, id)
	var s models.Skill
	err := row.Scan(&s.ID, &s.UserID, &s.Name, &s.Description, &s.TriggerPattern, &s.ExecutionType, &s.ConfigJSON, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func DeleteSkill(id string) error {
	_, err := DB.Exec("DELETE FROM skills WHERE id = ?", id)
	return err
}

func CreateConnector(c *models.Connector) error {
	_, err := DB.Exec(
		`INSERT INTO connectors (id, user_id, name, type, endpoint, auth_config_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.UserID, c.Name, c.Type, c.Endpoint, c.AuthConfigJSON, c.CreatedAt)
	return err
}

func ListConnectors(userID string) ([]models.Connector, error) {
	rows, err := DB.Query(
		`SELECT id, user_id, name, type, COALESCE(endpoint,''), COALESCE(auth_config_json,'{}'), created_at
		 FROM connectors WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []models.Connector
	for rows.Next() {
		var c models.Connector
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Type, &c.Endpoint, &c.AuthConfigJSON, &c.CreatedAt); err != nil {
			return nil, err
		}
		connectors = append(connectors, c)
	}
	return connectors, rows.Err()
}

func DeleteConnector(id string) error {
	_, err := DB.Exec("DELETE FROM connectors WHERE id = ?", id)
	return err
}

func CreateAgentTask(t *models.AgentTask) error {
	_, err := DB.Exec(
		`INSERT INTO agent_tasks (id, project_id, chat_id, goal, plan_json, status, permission_level, summary, created_at)
		 VALUES (?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.ChatID, t.Goal, t.PlanJSON, t.Status, t.PermissionLevel, t.Summary, t.CreatedAt)
	return err
}

func UpdateTaskStatus(id, status, summary string) error {
	_, err := DB.Exec(
		`UPDATE agent_tasks SET status = ?, summary = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, summary, id)
	return err
}

func GetTask(id string) (*models.AgentTask, error) {
	row := DB.QueryRow(
		`SELECT id, COALESCE(project_id,''), COALESCE(chat_id,''), goal, COALESCE(plan_json,'[]'), status, permission_level, COALESCE(summary,''), created_at
		 FROM agent_tasks WHERE id = ?`, id)
	var t models.AgentTask
	err := row.Scan(&t.ID, &t.ProjectID, &t.ChatID, &t.Goal, &t.PlanJSON, &t.Status, &t.PermissionLevel, &t.Summary, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CreateAgentAction(a *models.AgentAction) error {
	_, err := DB.Exec(
		`INSERT INTO agent_actions (id, task_id, action_type, target, details_json, requires_confirmation, confirmed_by_user, result, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.TaskID, a.ActionType, a.Target, a.DetailsJSON, a.RequiresConfirmation, a.ConfirmedByUser, a.Result, a.CreatedAt)
	return err
}

func ConfirmAction(actionID string) error {
	_, err := DB.Exec(
		`UPDATE agent_actions SET confirmed_by_user = 1 WHERE id = ?`, actionID)
	return err
}

func GetMessages(chatID string) ([]models.Message, error) {
	rows, err := DB.Query(
		`SELECT id, chat_id, role, content, COALESCE(attachments_json,'[]'), COALESCE(tool_calls_json,'[]'), COALESCE(feedback,''), created_at
		 FROM messages WHERE chat_id = ? ORDER BY created_at ASC`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.ChatID, &m.Role, &m.Content, &m.AttachmentsJSON, &m.ToolCallsJSON, &m.Feedback, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func UpdateMessageFeedback(messageID, feedback string) error {
	_, err := DB.Exec(`UPDATE messages SET feedback = ? WHERE id = ?`, feedback, messageID)
	return err
}

func InsertGraphEdge(edge *models.CodeGraphEdge) error {
	_, err := DB.Exec(
		`INSERT INTO code_graph_edges (id, project_id, from_symbol, to_symbol, edge_type, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		edge.ID, edge.ProjectID, edge.FromSymbol, edge.ToSymbol, edge.EdgeType, edge.CreatedAt)
	return err
}

func GetGraphNeighbors(projectID, filePath string) ([]models.CodeGraphEdge, error) {
	rows, err := DB.Query(
		`SELECT id, project_id, from_symbol, to_symbol, edge_type, created_at
		 FROM code_graph_edges
		 WHERE project_id = ? AND (from_symbol = ? OR to_symbol = ?)
		 ORDER BY created_at`,
		projectID, filePath, filePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []models.CodeGraphEdge
	for rows.Next() {
		var e models.CodeGraphEdge
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.FromSymbol, &e.ToSymbol, &e.EdgeType, &e.CreatedAt); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func ClearGraphEdges(projectID string) error {
	_, err := DB.Exec(`DELETE FROM code_graph_edges WHERE project_id = ?`, projectID)
	return err
}
