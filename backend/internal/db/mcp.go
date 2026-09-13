package db

import (
	"time"

	"github.com/ozyassist/backend/internal/db/models"
)

// CreateMCPConnector creates a new MCP connector configuration
func CreateMCPConnector(connector *models.MCPConnector) error {
	_, err := DB.Exec(`
		INSERT INTO mcp_connectors (id, user_id, name, command, args, env, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, connector.ID, connector.UserID, connector.Name, connector.Command, connector.Args, connector.Env, connector.Status, connector.CreatedAt, connector.UpdatedAt)
	return err
}

// ListMCPConnectors returns all MCP connectors for a user
func ListMCPConnectors(userID string) ([]models.MCPConnector, error) {
	rows, err := DB.Query(`
		SELECT id, user_id, name, command, args, env, status, created_at, updated_at
		FROM mcp_connectors WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []models.MCPConnector
	for rows.Next() {
		var c models.MCPConnector
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Command, &c.Args, &c.Env, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		connectors = append(connectors, c)
	}
	return connectors, nil
}

// GetMCPConnector returns a specific connector
func GetMCPConnector(id string) (*models.MCPConnector, error) {
	var c models.MCPConnector
	err := DB.QueryRow(`
		SELECT id, user_id, name, command, args, env, status, created_at, updated_at
		FROM mcp_connectors WHERE id = ?
	`, id).Scan(&c.ID, &c.UserID, &c.Name, &c.Command, &c.Args, &c.Env, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateMCPConnectorStatus updates the connection status of a connector
func UpdateMCPConnectorStatus(id string, status string) error {
	_, err := DB.Exec(`
		UPDATE mcp_connectors SET status = ?, updated_at = ? WHERE id = ?
	`, status, time.Now(), id)
	return err
}

// DeleteMCPConnector deletes a connector
func DeleteMCPConnector(id string) error {
	_, err := DB.Exec("DELETE FROM mcp_connectors WHERE id = ?", id)
	return err
}

// GetConnectedMCPConnectors returns all connectors that should be active
func GetConnectedMCPConnectors() ([]models.MCPConnector, error) {
	rows, err := DB.Query(`
		SELECT id, user_id, name, command, args, env, status, created_at, updated_at
		FROM mcp_connectors
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []models.MCPConnector
	for rows.Next() {
		var c models.MCPConnector
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Command, &c.Args, &c.Env, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		connectors = append(connectors, c)
	}
	return connectors, nil
}
