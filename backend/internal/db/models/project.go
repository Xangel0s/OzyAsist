package models

import "time"

type Project struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	Name            string    `json:"name"`
	RootPath        string    `json:"rootPath,omitempty"`
	InstructionsMd  string    `json:"instructionsMd,omitempty"`
	FileTreeJSON    string    `json:"fileTreeJson,omitempty"`
	PermissionLevel string    `json:"permissionLevel,omitempty"`
	AgentConsent    string    `json:"agentConsent,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
