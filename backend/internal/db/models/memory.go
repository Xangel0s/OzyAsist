package models

type MemoryEntry struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	ProjectID string `json:"projectId,omitempty"`
	Source    string `json:"source"`
	SourceID  string `json:"sourceId,omitempty"`
	Topic     string `json:"topic"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}
