package agent

import (
	"encoding/json"

	"github.com/ozyassist/backend/internal/providers"
)

// AgentTools son las herramientas nativas del agente disponibles para el LLM en Modo Code.
// El InputSchema sigue el formato JSON Schema estándar (agnóstico de provider).
// La capa de providers se encarga de traducir al formato de cada API (Anthropic, OpenAI, etc.)
var AgentTools = []providers.ToolDef{
	{
		Name:        "read_file",
		Description: "Lee el contenido completo de un archivo del proyecto. Úsalo para entender el código antes de modificarlo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo desde la raíz del proyecto (ej: src/main.go)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "write_file",
		Description: "Crea o sobreescribe completamente un archivo del proyecto. Incluye el contenido completo del archivo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo desde la raíz del proyecto"
				},
				"content": {
					"type": "string",
					"description": "Contenido completo del archivo"
				}
			},
			"required": ["path", "content"]
		}`),
	},
	{
		Name:        "run_command",
		Description: "Ejecuta un comando shell en el directorio raíz del proyecto. Usa esto para compilar, correr tests, instalar dependencias, etc.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "Comando a ejecutar (ej: go build ./..., npm install, git status)"
				}
			},
			"required": ["command"]
		}`),
	},
	{
		Name:        "list_files",
		Description: "Lista archivos del proyecto con un patrón glob. Útil para explorar la estructura del proyecto.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"pattern": {
					"type": "string",
					"description": "Patrón glob (ej: **/*.go, src/**/*.ts, *.json)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo número de resultados a devolver (default: 50)"
				}
			},
			"required": ["pattern"]
		}`),
	},
	{
		Name:        "search_text",
		Description: "Busca texto o expresiones regulares dentro de los archivos del proyecto (similar a ripgrep). Devuelve archivo, línea y contenido.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Texto o regex a buscar"
				},
				"include": {
					"type": "string",
					"description": "Patrón glob para filtrar archivos (ej: *.go, *.ts)"
				},
				"case_sensitive": {
					"type": "boolean",
					"description": "Si la búsqueda es sensible a mayúsculas (default: false)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo número de coincidencias (default: 30)"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "apply_diff",
		Description: "Aplica un diff unificado (formato unified diff) a un archivo. Úsalo para modificaciones quirúrgicas sin reescribir el archivo completo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo a parchear"
				},
				"diff": {
					"type": "string",
					"description": "Diff en formato unified diff estándar (--- a/ +++ b/ @@ ...)"
				}
			},
			"required": ["path", "diff"]
		}`),
	},
}

func mustJSON(s string) json.RawMessage {
	// Comprimir el JSON eliminando espacios innecesarios
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic("tools.go: invalid JSON schema: " + err.Error())
	}
	raw, _ := json.Marshal(v)
	return json.RawMessage(raw)
}
