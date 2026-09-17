package agent

import "github.com/ozyassist/backend/internal/providers"

func init() {
	AgentTools = append(AgentTools, providers.ToolDef{
		Name:        "os_search_index",
		Description: "Busca en el índice del sistema (ChromaDB) información sobre aplicaciones instaladas o memoria pasada. Útil para encontrar rutas de programas antes de ejecutarlos.",
		InputSchema: mustJSON(`{"type":"object","properties":{"query":{"type":"string","description":"Nombre del programa o dato a buscar"}},"required":["query"]}`),
	})
	
	VoiceAgentTools = append(VoiceAgentTools, providers.ToolDef{
		Name:        "os_search_index",
		Description: "Busca en el índice del sistema (ChromaDB) información sobre aplicaciones instaladas.",
		InputSchema: mustJSON(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
	})
}
