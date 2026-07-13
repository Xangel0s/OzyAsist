package models

import "time"

type Message struct {
	ID              string    `json:"id"`
	ChatID          string    `json:"chatId"`
	Role            string    `json:"role"`
	Content         string    `json:"content"`
	AttachmentsJSON string    `json:"attachments,omitempty"`
	ToolCallsJSON   string    `json:"toolCalls,omitempty"`
	Feedback        string    `json:"feedback,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
