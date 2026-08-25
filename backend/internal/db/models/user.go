package models

import "time"

type User struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Email           string    `json:"email,omitempty"`
	AvatarColor     string    `json:"avatarColor,omitempty"`
	HasPin          bool      `json:"hasPin"`
	PinHash         string    `json:"-"`
	Role            string    `json:"role,omitempty"`
	Plan            string    `json:"plan,omitempty"`
	ProfileMd       string    `json:"profileMd,omitempty"`
	DefaultProvider string    `json:"defaultProvider,omitempty"`
	DefaultModel    string    `json:"defaultModel,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
