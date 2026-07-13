package models

import "time"

type User struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	ProfileMd       string    `json:"profileMd,omitempty"`
	DefaultProvider string    `json:"defaultProvider,omitempty"`
	DefaultModel    string    `json:"defaultModel,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
