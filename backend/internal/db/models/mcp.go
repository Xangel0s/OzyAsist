package models

import (
	"encoding/json"
	"time"
)

type MCPConnector struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Command   string    `json:"command"`
	Args      string    `json:"args"` // Stored as JSON array
	Env       string    `json:"env"`  // Stored as JSON array
	Status    string    `json:"status"` // 'disconnected', 'connected', 'error'
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (m *MCPConnector) GetArgs() []string {
	var args []string
	if m.Args != "" {
		json.Unmarshal([]byte(m.Args), &args)
	}
	return args
}

func (m *MCPConnector) GetEnv() []string {
	var env []string
	if m.Env != "" {
		json.Unmarshal([]byte(m.Env), &env)
	}
	return env
}

func (m *MCPConnector) SetArgs(args []string) {
	b, _ := json.Marshal(args)
	m.Args = string(b)
}

func (m *MCPConnector) SetEnv(env []string) {
	b, _ := json.Marshal(env)
	m.Env = string(b)
}
