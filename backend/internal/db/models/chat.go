package models

import "time"

type Chat struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	ProjectID string    `json:"projectId,omitempty"`
	Name      string    `json:"name"`
	Mode      string    `json:"mode"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"createdAt"`
}
