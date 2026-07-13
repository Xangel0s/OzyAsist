package models

import "time"

type CodeGraphEdge struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"projectId"`
	FromSymbol string    `json:"fromSymbol"`
	ToSymbol   string    `json:"toSymbol"`
	EdgeType   string    `json:"edgeType"`
	CreatedAt  time.Time `json:"createdAt"`
}
