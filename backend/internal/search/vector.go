package search

import (
	"fmt"

	"github.com/ozyassist/backend/internal/memory"
)

// VectorSearch busca por similitud semántica vía Caps2 (Qdrant).
// Si Qdrant no está disponible, devuelve error que el llamador puede ignorar.
func VectorSearch(query string, limit int, projectID, userID string) ([]string, error) {
	caps2 := memory.GetCaps2()
	if caps2 == nil {
		return nil, fmt.Errorf("caps2 not available")
	}

	results, err := caps2.Search(query, limit, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	return results, nil
}
