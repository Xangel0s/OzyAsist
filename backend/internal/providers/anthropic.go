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
			client:  &http.Client{},
		},
	}
}

func (p *AnthropicProvider) Name() string           { return "anthropic" }
func (p *AnthropicProvider) SupportsTools() bool     { return true }
func (p *AnthropicProvider) Models() []string {
	return []string{"claude-sonnet-4-20250514", "claude-haiku-4-20250514", "claude-opus-4-20250514"}
}

func (p *AnthropicProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)

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
		resp.Body.Close()
		close(ch)
		return nil, fmt.Errorf("Anthropic API status %d", resp.StatusCode)
	}

	go p.readStream(ctx, ch, resp.Body)
	return ch, nil
}

func (p *AnthropicProvider) readStream(ctx context.Context, ch chan<- StreamChunk, body io.ReadCloser) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	var currentBlockID string
	var currentBlockType string

	for scanner.Scan() {
		line := scanner.Text()

		if ctx.Err() != nil {
			ch <- StreamChunk{Type: "error", Content: "cancelado"}
			return
		}

		if strings.HasPrefix(line, "event: ") {
			eventType := strings.TrimPrefix(line, "event: ")
			if eventType == "content_block_start" || eventType == "content_block_delta" || eventType == "message_stop" || eventType == "message_start" {
				continue
			}
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type string `json:"type"`
			Delta *struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
			} `json:"delta,omitempty"`
			ContentBlock *struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
				Input *json.RawMessage `json:"input,omitempty"`
			} `json:"content_block,omitempty"`
			Message *struct {
				ID string `json:"id"`
				StopReason string `json:"stop_reason"`
			} `json:"message,omitempty"`
			ContentBlockStop *struct {
				Type string `json:"type"`
			} `json:"content_block_stop,omitempty"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock != nil {
				currentBlockID = event.ContentBlock.ID
				currentBlockType = event.ContentBlock.Type
				if event.ContentBlock.Type == "tool_use" {
					ch <- StreamChunk{
						Type:     "tool_call",
						ToolID:   event.ContentBlock.ID,
						ToolName: event.ContentBlock.Name,
						ToolInput: "",
					}
				}
			}
		case "content_block_delta":
			if event.Delta != nil {
				if event.Delta.Type == "text_delta" {
					ch <- StreamChunk{Type: "text", Content: event.Delta.Text}
				}
				if currentBlockType == "tool_use" && event.Delta.Type == "input_json_delta" {
					ch <- StreamChunk{
						Type:      "tool_call",
						ToolID:    currentBlockID,
						ToolInput: event.Delta.PartialJSON,
					}
				}
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

func toAnthropicMessages(msgs []Message) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		obj := map[string]any{"role": m.Role, "content": m.Content}
		out = append(out, obj)
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
	// Ensure the type implements Provider
	var _ Provider = (*AnthropicProvider)(nil)
}
