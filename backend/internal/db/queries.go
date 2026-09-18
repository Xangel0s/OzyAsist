package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db/models"
	"golang.org/x/crypto/bcrypt"
)

func escapeLike(s string) string {
	return strings.NewReplacer(`%`, `\%`, `_`, `\_`).Replace(s)
}

var defaultUserID string

func EnsureDefaultUser() error {
	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		count = 0
	}
	if count > 0 {
		row := DB.QueryRow("SELECT id FROM users ORDER BY created_at ASC LIMIT 1")
		return row.Scan(&defaultUserID)
	}

	id := uuid.NewString()
	_, err := DB.Exec(
		`INSERT INTO users (id, name, avatar_color, plan, role, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, "Angel", "#d1f107", "free", "developer", time.Now())
	if err != nil {
		return err
	}
	defaultUserID = id
	return nil
}

func DefaultUserID() string {
	return defaultUserID
}

func SetActiveUserID(id string) {
	defaultUserID = id
}

func ListUsers() ([]models.User, error) {
	rows, err := DB.Query(`
		SELECT id, name, COALESCE(email, ''), COALESCE(avatar_color, '#d1f107'), 
		       COALESCE(pin_hash, '') != '' AS has_pin, COALESCE(pin_hash, ''),
		       COALESCE(plan, 'free'), COALESCE(role, 'developer'),
		       COALESCE(profile_md, ''), COALESCE(default_provider, ''), COALESCE(default_model, ''), created_at
		FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		var pinHash string
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.AvatarColor, &u.HasPin, &pinHash, &u.Plan, &u.Role, &u.ProfileMd, &u.DefaultProvider, &u.DefaultModel, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.PinHash = pinHash
		users = append(users, u)
	}
	return users, nil
}

func GetUser(id string) (*models.User, error) {
	var u models.User
	var pinHash string
	err := DB.QueryRow(`
		SELECT id, name, COALESCE(email, ''), COALESCE(avatar_color, '#d1f107'), 
		       COALESCE(pin_hash, '') != '' AS has_pin, COALESCE(pin_hash, ''),
		       COALESCE(plan, 'free'), COALESCE(role, 'developer'),
		       COALESCE(profile_md, ''), COALESCE(default_provider, ''), COALESCE(default_model, ''), created_at
		FROM users WHERE id = ?`, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.AvatarColor, &u.HasPin, &pinHash, &u.Plan, &u.Role, &u.ProfileMd, &u.DefaultProvider, &u.DefaultModel, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.PinHash = pinHash
	return &u, nil
}

func CreateUserProfile(name, email, avatarColor, pin, role string) (*models.User, error) {
	id := uuid.NewString()
	if avatarColor == "" {
		avatarColor = "#d1f107"
	}
	if role == "" {
		role = "developer"
	}

	pinHash := ""
	if strings.TrimSpace(pin) != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(pin)), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("error hashing PIN: %w", err)
		}
		pinHash = string(hashed)
	}

	now := time.Now()
	_, err := DB.Exec(`
		INSERT INTO users (id, name, email, avatar_color, pin_hash, plan, role, created_at)
		VALUES (?, ?, ?, ?, ?, 'free', ?, ?)`,
		id, name, email, avatarColor, pinHash, role, now)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:          id,
		Name:        name,
		Email:       email,
		AvatarColor: avatarColor,
		HasPin:      pinHash != "",
		Plan:        "free",
		Role:        role,
		CreatedAt:   now,
	}, nil
}

func VerifyUserPin(id, pin string) (bool, *models.User, error) {
	u, err := GetUser(id)
	if err != nil {
		return false, nil, err
	}
	if !u.HasPin {
		return true, u, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PinHash), []byte(strings.TrimSpace(pin)))
	if err != nil {
		return false, nil, nil
	}
	return true, u, nil
}

func UpdateUserPin(id, oldPin, newPin string) error {
	u, err := GetUser(id)
	if err != nil {
		return err
	}

	if u.HasPin {
		if err := bcrypt.CompareHashAndPassword([]byte(u.PinHash), []byte(strings.TrimSpace(oldPin))); err != nil {
			return fmt.Errorf("PIN actual incorrecto")
		}
	}

	newHash := ""
	if strings.TrimSpace(newPin) != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(newPin)), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("error hashing nuevo PIN: %w", err)
		}
		newHash = string(hashed)
	}

	_, err = DB.Exec(`UPDATE users SET pin_hash = ? WHERE id = ?`, newHash, id)
	return err
}

func DeleteUser(id string) error {
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err == nil && count <= 1 {
		return fmt.Errorf("no puedes eliminar el único perfil existente")
	}
	_, err := DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
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

func UpdateChat(c *models.Chat) error {
	_, err := DB.Exec(
		`UPDATE chats SET name = ?, project_id = NULLIF(?, ''), mode = ?, provider = ?, model = ? WHERE id = ?`,
		c.Name, c.ProjectID, c.Mode, c.Provider, c.Model, c.ID)
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

	chats := make([]models.Chat, 0)
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

	projects := make([]models.Project, 0)
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

func execTx(tx *sql.Tx, query string, args ...any) error {
	_, err := tx.Exec(query, args...)
	return err
}

func DeleteProject(id string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := execTx(tx, "UPDATE chats SET project_id = NULL WHERE project_id = ?", id); err != nil {
		return err
	}
	if err := execTx(tx, "UPDATE agent_tasks SET project_id = NULL WHERE project_id = ?", id); err != nil {
		return err
	}
	if err := execTx(tx, "DELETE FROM code_graph_edges WHERE project_id = ?", id); err != nil {
		return err
	}
	if err := execTx(tx, "DELETE FROM projects WHERE id = ?", id); err != nil {
		return err
	}
	return tx.Commit()
}

func DeleteChat(chatID string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := execTx(tx, "DELETE FROM messages WHERE chat_id = ?", chatID); err != nil {
		return err
	}
	if err := execTx(tx, "DELETE FROM chats WHERE id = ?", chatID); err != nil {
		return err
	}
	return tx.Commit()
}

// CountMessagesByChat retorna la cantidad de mensajes guardados en un chat específico.
func CountMessagesByChat(chatID string) (int, error) {
	if DB == nil {
		return 0, nil
	}
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM messages WHERE chat_id = ?", chatID).Scan(&count)
	return count, err
}

// CleanupEmptyChats elimina de SQLite todos los chats que no tengan ningún mensaje asociado,
// evitando la acumulación de sesiones vacías o huérfanas. Si excludeChatID no está vacío,
// preserva dicho chat aunque esté temporalmente vacío (ej: la sesión recién creada en uso).
func CleanupEmptyChats(excludeChatID string) error {
	if DB == nil {
		return nil
	}
	query := "DELETE FROM chats WHERE id NOT IN (SELECT DISTINCT chat_id FROM messages)"
	if excludeChatID != "" {
		query += " AND id != ?"
		_, err := DB.Exec(query, excludeChatID)
		return err
	}
	_, err := DB.Exec(query)
	return err
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

	results := make([]SearchResult, 0)
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
	escaped := escapeLike(query)
	rows, err := DB.Query(
		`SELECT id, name, '' as snippet FROM chats WHERE name LIKE ? ESCAPE '\' ORDER BY created_at DESC LIMIT ?`,
		"%"+escaped+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0)
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
	escaped := escapeLike(query)
	rows, err := DB.Query(
		`SELECT id, name, COALESCE(root_path,'') FROM projects WHERE name LIKE ? ESCAPE '\' ORDER BY created_at DESC LIMIT ?`,
		"%"+escaped+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0)
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

	entries := make([]models.MemoryEntry, 0)
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

	results := make([]SearchResult, 0)
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

	skills := make([]models.Skill, 0)
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

	connectors := make([]models.Connector, 0)
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
		`INSERT INTO agent_tasks (id, user_id, project_id, title, prompt, status, total_steps, current_step, error_message, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.UserID, t.ProjectID, t.Title, t.Prompt, string(t.Status), t.TotalSteps, t.CurrentStep, t.ErrorMessage, t.CreatedAt, t.UpdatedAt)
	return err
}

func UpdateTaskStatus(id, status, summary string) error {
	var err error
	if summary != "" {
		_, err = DB.Exec(`UPDATE agent_tasks SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, summary, id)
	} else {
		_, err = DB.Exec(`UPDATE agent_tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	}
	return err
}

func GetTask(id string) (*models.AgentTask, error) {
	row := DB.QueryRow(
		`SELECT id, user_id, project_id, title, prompt, status, total_steps, current_step, error_message, created_at, updated_at
		 FROM agent_tasks WHERE id = ?`, id)
	var t models.AgentTask
	var status string
	err := row.Scan(&t.ID, &t.UserID, &t.ProjectID, &t.Title, &t.Prompt, &status, &t.TotalSteps, &t.CurrentStep, &t.ErrorMessage, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	t.Status = models.TaskStatus(status)
	return &t, nil
}

func CreateTaskStep(s *models.TaskStep) error {
	_, err := DB.Exec(
		`INSERT INTO task_steps (id, task_id, step_order, agent_assigned, action_type, payload, requires_pin, pin_authorized, verification_rule, status, output, retry_count, max_retries, created_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TaskID, s.StepOrder, s.AgentAssigned, s.ActionType, s.Payload, s.RequiresPIN, s.PINAuthorized, s.VerificationRule, string(s.Status), s.Output, s.RetryCount, s.MaxRetries, s.CreatedAt, s.CompletedAt)
	return err
}

func UpdateTaskStepStatus(stepID string, status models.StepStatus, output *string) error {
	terminal := status == models.StepStatusCompleted || status == models.StepStatusFailed
	var err error
	if terminal {
		_, err = DB.Exec(`UPDATE task_steps SET status = ?, output = ?, completed_at = CURRENT_TIMESTAMP WHERE id = ?`, string(status), output, stepID)
	} else {
		_, err = DB.Exec(`UPDATE task_steps SET status = ?, output = ? WHERE id = ?`, string(status), output, stepID)
	}
	return err
}

func GetPendingTaskSteps(taskID string) ([]models.TaskStep, error) {
	rows, err := DB.Query(
		`SELECT id, task_id, step_order, agent_assigned, action_type, payload, requires_pin, pin_authorized, verification_rule, status, output, retry_count, max_retries, created_at, completed_at
		 FROM task_steps WHERE task_id = ? AND status != 'completed' ORDER BY step_order ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []models.TaskStep
	for rows.Next() {
		var s models.TaskStep
		var st string
		if err := rows.Scan(&s.ID, &s.TaskID, &s.StepOrder, &s.AgentAssigned, &s.ActionType, &s.Payload, &s.RequiresPIN, &s.PINAuthorized, &s.VerificationRule, &st, &s.Output, &s.RetryCount, &s.MaxRetries, &s.CreatedAt, &s.CompletedAt); err != nil {
			return nil, err
		}
		s.Status = models.StepStatus(st)
		steps = append(steps, s)
	}
	return steps, nil
}

func ConfirmAction(actionID string) error {
	_, err := DB.Exec(
		`UPDATE task_steps SET pin_authorized = 1 WHERE id = ?`, actionID)
	return err
}

func CancelAllRunningTasks() error {
	query := `UPDATE agent_tasks SET status = 'cancelled', error_message = 'Cancelación forzosa por Kill-Switch', updated_at = CURRENT_TIMESTAMP WHERE status IN ('pending', 'planning', 'running', 'blocked_approval')`
	_, err := DB.Exec(query)
	return err
}

func CreateAgentAction(a *models.AgentAction) error {
	stepOrder := 1
	var maxOrder int
	_ = DB.QueryRow(`SELECT COALESCE(MAX(step_order), 0) FROM task_steps WHERE task_id = ?`, a.TaskID).Scan(&maxOrder)
	stepOrder = maxOrder + 1

	st := models.StepStatusCompleted
	if a.RequiresConfirmation && !a.ConfirmedByUser {
		st = models.StepStatusPending
	}
	s := &models.TaskStep{
		ID:            a.ID,
		TaskID:        a.TaskID,
		StepOrder:     stepOrder,
		AgentAssigned: "ozy",
		ActionType:    a.ActionType,
		Payload:       a.DetailsJSON,
		RequiresPIN:   a.RequiresConfirmation,
		PINAuthorized: a.ConfirmedByUser,
		Status:        st,
		Output:        &a.Result,
		CreatedAt:     a.CreatedAt,
	}
	return CreateTaskStep(s)
}


func GetMessages(chatID string) ([]models.Message, error) {
	rows, err := DB.Query(
		`SELECT id, chat_id, role, content, COALESCE(attachments_json,'[]'), COALESCE(tool_calls_json,'[]'), COALESCE(feedback,''), created_at
		 FROM messages WHERE chat_id = ? ORDER BY created_at ASC`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	msgs := make([]models.Message, 0)
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

	edges := make([]models.CodeGraphEdge, 0)
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

func GetAllGraphEdges(projectID string) ([]models.CodeGraphEdge, error) {
	rows, err := DB.Query(
		`SELECT id, project_id, from_symbol, to_symbol, edge_type, created_at
		 FROM code_graph_edges
		 WHERE project_id = ?
		 ORDER BY from_symbol, to_symbol`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	edges := make([]models.CodeGraphEdge, 0)
	for rows.Next() {
		var e models.CodeGraphEdge
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.FromSymbol, &e.ToSymbol, &e.EdgeType, &e.CreatedAt); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func SaveFileTree(projectID, treeJSON string) error {
	_, err := DB.Exec(`UPDATE projects SET file_tree_json = ? WHERE id = ?`, treeJSON, projectID)
	return err
}
