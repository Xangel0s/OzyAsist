package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

func execOSSearchIndex(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(tc.Input, &args); err != nil {
		return "Error: " + err.Error(), false
	}

	results, err := memory.GetCaps2().Search(args.Query, 3, "", "")
	if err != nil {
		return "Error consultando ChromaDB: " + err.Error(), false
	}

	if len(results) == 0 {
		return "No se encontraron resultados en el índice.", true
	}

	return "Resultados del índice:\n" + strings.Join(results, "\n"), true
}
