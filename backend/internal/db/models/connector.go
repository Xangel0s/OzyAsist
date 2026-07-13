package models

import "time"

type Connector struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Endpoint      string    `json:"endpoint"`
	AuthConfigJSON string   `json:"authConfig,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}
