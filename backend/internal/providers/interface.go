package providers

import "context"

type StreamChunk struct {
	Type    string `json:"type"` // text | tool_call | done | error
	Content string `json:"content,omitempty"`
	ToolID  string `json:"tool_id,omitempty"`
	ToolName string `json:"tool_name,omitempty"`
	ToolInput string `json:"tool_input,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionOptions struct {
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Stream      bool    `json:"stream"`
}

type Provider interface {
	StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error)
	Name() string
	SupportsTools() bool
	Models() []string
}
