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

type openaiCfg struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type OpenAIProvider struct {
	cfg *openaiCfg
}

func NewOpenAI(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		cfg: &openaiCfg{
			apiKey:  apiKey,
			baseURL: "https://api.openai.com/v1",
			model:   "gpt-4o",
			client:  &http.Client{},
		},
	}
}

func (p *OpenAIProvider) Name() string       { return "openai" }
func (p *OpenAIProvider) SupportsTools() bool { return true }
func (p *OpenAIProvider) Models() []string    { return []string{"gpt-4o", "gpt-4o-mini", "o3", "o4-mini"} }

func (p *OpenAIProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)

	model := opts.Model
	if model == "" {
		model = p.cfg.model
	}
	body := map[string]any{
		"model":      model,
		"messages":   toOpenAIMessages(messages),
		"stream":     true,
		"stream_options": map[string]bool{"include_usage": false},
	}
	if opts.Temperature != 0 {
		body["temperature"] = opts.Temperature
	}
	if opts.MaxTokens != 0 {
		body["max_tokens"] = opts.MaxTokens
	}

	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("crear request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.apiKey)

	resp, err := p.cfg.client.Do(req)
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("request fallido: %w", err)
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		close(ch)
		return nil, fmt.Errorf("OpenAI API status %d", resp.StatusCode)
	}

	go p.readStream(ctx, ch, resp.Body)
	return ch, nil
}

func (p *OpenAIProvider) readStream(ctx context.Context, ch chan<- StreamChunk, body io.ReadCloser) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	var toolAccum map[int]struct {
		id     string
		name   string
		args   string
	}

	for scanner.Scan() {
		line := scanner.Text()

		if ctx.Err() != nil {
			ch <- StreamChunk{Type: "error", Content: "cancelado"}
			return
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var sse struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Choices []struct {
				Index int `json:"index"`
			Delta struct {
				Role             string `json:"role"`
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &sse); err != nil {
			continue
		}

		for _, choice := range sse.Choices {
			if choice.Delta.Content != "" {
				ch <- StreamChunk{Type: "text", Content: choice.Delta.Content}
			}

			for _, tc := range choice.Delta.ToolCalls {
				if tc.Function.Name != "" || tc.ID != "" {
					if toolAccum == nil {
						toolAccum = make(map[int]struct{ id string; name string; args string })
					}
					acc := toolAccum[tc.Index]
					if tc.ID != "" {
						acc.id = tc.ID
					}
					if tc.Function.Name != "" {
						acc.name = tc.Function.Name
					}
					acc.args += tc.Function.Arguments
					toolAccum[tc.Index] = acc
				}
			}

			if choice.FinishReason == "tool_calls" {
				for _, acc := range toolAccum {
					ch <- StreamChunk{
						Type:     "tool_call",
						ToolID:   acc.id,
						ToolName: acc.name,
						ToolInput: acc.args,
					}
				}
				toolAccum = nil
			}

			if choice.FinishReason == "stop" {
				ch <- StreamChunk{Type: "done"}
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Type: "error", Content: err.Error()}
	}
}

func toOpenAIMessages(msgs []Message) []any {
	out := make([]any, len(msgs))
	for i, m := range msgs {
		obj := map[string]any{"role": m.Role, "content": m.Content}
		out[i] = obj
	}
	return out
}
