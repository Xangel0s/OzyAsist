package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
)

func init() {
	AgentTools = append(AgentTools, providers.ToolDef{
		Name: "query_system_knowledge",
		Description: "Consulta el Grafo Ontológico de Conocimiento (GraphRAG en RAM/SQLite) para conocer contratos de herramientas, capacidades de Windows, reglas de ejecución, subsistemas de hardware/audio/cámara/python y relaciones ontológicas del sistema.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Concepto, herramienta, subsistema o duda sobre cómo funciona algo en OzyAssist (ej. 'camara nokhwa', 'audio wasapi', 'recuperacion mcts', 'python workspace', 'servicios de windows')"
				},
				"topK": {
					"type": "integer",
					"description": "Número de nodos ontológicos relevantes a recuperar (por defecto 3)"
				}
			},
			"required": ["query"]
		}`),
	})
}

func execQuerySystemKnowledge(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Query string `json:"query"`
		TopK  int    `json:"topK"`
	}

	if err := json.Unmarshal(tc.Input, &params); err != nil || strings.TrimSpace(params.Query) == "" {
		return "❌ Parámetros inválidos para query_system_knowledge: se requiere 'query'.", false
	}

	if params.TopK <= 0 {
		params.TopK = 3
	}

	engine := memory.GetGraphRAGEngine()
	if engine == nil {
		return "⚠️ El motor GraphRAG no está inicializado en memoria.", false
	}

	result := engine.QueryGraphRAG(params.Query, params.TopK)
	return result, true
}
