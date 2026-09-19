package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type anthropicCfg struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type AnthropicProvider struct {
	cfg *anthropicCfg
}

func NewAnthropic(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		cfg: &anthropicCfg{
			apiKey:  apiKey,
			baseURL: "https://api.anthropic.com/v1",
			model:   "claude-sonnet-4-20250514",
			client: &http.Client{
				Transport: &http.Transport{
					ResponseHeaderTimeout: 60 * time.Second,
					IdleConnTimeout:       90 * time.Second,
				},
			},
		},
	}
}

func (p *AnthropicProvider) Name() string       { return "anthropic" }
func (p *AnthropicProvider) SupportsTools() bool { return true }
func (p *AnthropicProvider) Models() []string {
	return []string{"claude-sonnet-4-20250514", "claude-haiku-4-20250514", "claude-opus-4-20250514"}
}

func (p *AnthropicProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 32)

	model := opts.Model
	if model == "" {
		model = p.cfg.model
	}

	body := map[string]any{
		"model":      model,
		"messages":   toAnthropicMessages(messages),
		"max_tokens": 8192,
		"stream":     true,
	}

	system := extractSystem(messages)
	if system != "" {
		body["system"] = system
	}
	if opts.MaxTokens != 0 {
		body["max_tokens"] = opts.MaxTokens
	}
	if opts.Temperature != 0 {
		body["temperature"] = opts.Temperature
	}

	// Tool calling — Anthropic usa "tools" con "input_schema"
	if len(opts.Tools) > 0 {
		body["tools"] = toAnthropicTools(opts.Tools)
	}

	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.baseURL+"/messages", bytes.NewReader(raw))
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("crear request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.cfg.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.cfg.client.Do(req)
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("request fallido: %w", err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		close(ch)
		return nil, fmt.Errorf("Anthropic API status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	go p.readStream(ctx, ch, resp.Body)
	return ch, nil
}

// toAnthropicTools convierte ToolDef agnóstico al formato nativo de Anthropic.
// Anthropic espera: {"name","description","input_schema": <JSON Schema>}
func toAnthropicTools(tools []ToolDef) []any {
	out := make([]any, len(tools))
	for i, t := range tools {
		schema := t.InputSchema
		if schema == nil {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out[i] = map[string]any{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": schema,
		}
	}
	return out
}

func (p *AnthropicProvider) readStream(ctx context.Context, ch chan<- StreamChunk, body io.ReadCloser) {
	defer body.Close()
	defer close(ch)

	stopCancelWatcher := make(chan struct{})
	defer close(stopCancelWatcher)
	go func() {
		select {
		case <-ctx.Done():
			_ = body.Close()
		case <-stopCancelWatcher:
		}
	}()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	// Acumuladores por block_index
	type toolBlock struct {
		id        string
		name      string
		inputJSON string // JSON parcial acumulado
	}
	blocks := make(map[int]*toolBlock)

	for scanner.Scan() {
		if ctx.Err() != nil {
			ch <- StreamChunk{Type: "error", Content: "cancelado"}
			return
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Index int    `json:"index"`

			// content_block_start
			ContentBlock *struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block,omitempty"`

			// content_block_delta
			Delta *struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta,omitempty"`

			// message_delta (stop_reason)
			Message *struct {
				StopReason string `json:"stop_reason"`
			} `json:"message,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				blocks[event.Index] = &toolBlock{
					id:   event.ContentBlock.ID,
					name: event.ContentBlock.Name,
				}
			}

		case "content_block_delta":
			if event.Delta == nil {
				continue
			}
			if event.Delta.Type == "text_delta" {
				ch <- StreamChunk{Type: "text", Content: event.Delta.Text}
			}
			if event.Delta.Type == "input_json_delta" {
				if b, ok := blocks[event.Index]; ok {
					b.inputJSON += event.Delta.PartialJSON
				}
			}

		case "content_block_stop":
			// Tool block completado — emitir ToolCall completo con input parseado
			if b, ok := blocks[event.Index]; ok {
				var inputRaw json.RawMessage
				if b.inputJSON != "" {
					inputRaw = json.RawMessage(b.inputJSON)
				} else {
					inputRaw = json.RawMessage(`{}`)
				}
				ch <- StreamChunk{
					Type: "tool_call",
					ToolCall: &ToolCall{
						ID:    b.id,
						Name:  b.name,
						Input: inputRaw,
					},
				}
				delete(blocks, event.Index)
			}

		case "message_stop":
			ch <- StreamChunk{Type: "done"}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Type: "error", Content: err.Error()}
	}
}

// toAnthropicMessages convierte el historial a formato Anthropic.
// Los mensajes con role="tool" se convierten en bloques tool_result dentro de user.
// Los mensajes con role="assistant" y ToolCalls se convierten al formato de Anthropic.
func toAnthropicMessages(msgs []Message) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}

		switch m.Role {
		case "user":
			out = append(out, map[string]any{
				"role":    "user",
				"content": m.Content,
			})

		case "assistant":
			if len(m.ToolCalls) == 0 {
				out = append(out, map[string]any{
					"role":    "assistant",
					"content": m.Content,
				})
			} else {
				// Anthropic: content es array de bloques (text + tool_use)
				var content []any
				if m.Content != "" {
					content = append(content, map[string]any{
						"type": "text",
						"text": m.Content,
					})
				}
				for _, tc := range m.ToolCalls {
					inputRaw := tc.Input
					if inputRaw == nil {
						inputRaw = json.RawMessage(`{}`)
					}
					var inputObj any
					json.Unmarshal(inputRaw, &inputObj)
					content = append(content, map[string]any{
						"type":  "tool_use",
						"id":    tc.ID,
						"name":  tc.Name,
						"input": inputObj,
					})
				}
				out = append(out, map[string]any{
					"role":    "assistant",
					"content": content,
				})
			}

		case "tool":
			if m.ToolResult != nil {
				// Anthropic: el tool result va en un mensaje user con content=[tool_result block]
				out = append(out, map[string]any{
					"role": "user",
					"content": []any{
						map[string]any{
							"type":        "tool_result",
							"tool_use_id": m.ToolResult.ToolCallID,
							"content":     m.ToolResult.Content,
						},
					},
				})
			}
		}
	}
	return out
}

func extractSystem(msgs []Message) string {
	for _, m := range msgs {
		if m.Role == "system" {
			return m.Content
		}
	}
	return ""
}

func init() {
	var _ Provider = (*AnthropicProvider)(nil)
}
