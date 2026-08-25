package providers

import (
	"context"
	"encoding/json"
)

// ToolDef define una herramienta que el LLM puede invocar.
// El InputSchema es un JSON Schema estándar (agnóstico de provider).
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolCall representa una invocación de herramienta solicitada por el LLM.
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"` // JSON parseado del input
}

// ToolResult es el resultado de ejecutar una herramienta, para enviarlo de vuelta al LLM.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

// Message representa un mensaje en el historial de conversación.
// Soporta todos los roles del protocolo de tool calling.
type Message struct {
	Role string `json:"role"` // system | user | assistant | tool

	// Para role=system, user, assistant (texto plano)
	Content string `json:"content,omitempty"`

	// Para role=assistant cuando el LLM solicitó tool calls
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Para role=tool (resultado de ejecución de herramienta)
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// StreamChunk es un fragmento del stream de respuesta del LLM.
type StreamChunk struct {
	Type string `json:"type"` // text | tool_call | done | error

	// Para type=text
	Content string `json:"content,omitempty"`

	// Para type=tool_call (tool call completamente acumulado)
	ToolCall *ToolCall `json:"tool_call,omitempty"`
}

// CompletionOptions configura la llamada al LLM.
type CompletionOptions struct {
	Model       string    `json:"model,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
	Tools       []ToolDef `json:"tools,omitempty"` // nil = sin tool calling
}

// Provider es la interfaz unificada para todos los proveedores LLM.
type Provider interface {
	// StreamCompletion hace una llamada al LLM con soporte opcional de tool calling.
	// Devuelve un canal de StreamChunk cerrado al terminar.
	StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error)
	Name() string
	SupportsTools() bool
	Models() []string
}
