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

type openaiCfg struct {
	providerName string
	apiKey       string
	baseURL      string
	model        string
	client       *http.Client
}

type OpenAIProvider struct {
	cfg *openaiCfg
}

func NewOpenAI(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		cfg: &openaiCfg{
			providerName: "openai",
			apiKey:       apiKey,
			baseURL:      "https://api.openai.com/v1",
			model:        "gpt-4o",
			client:       &http.Client{},
		},
	}
}

func (p *OpenAIProvider) Name() string {
	if p.cfg != nil && p.cfg.providerName != "" {
		return p.cfg.providerName
	}
	return "openai"
}
func (p *OpenAIProvider) SupportsTools() bool { return true }
func (p *OpenAIProvider) Models() []string    { return []string{"gpt-4o", "gpt-4o-mini", "o3", "o4-mini"} }

func (p *OpenAIProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 32)

	model := opts.Model
	if strings.Contains(p.cfg.baseURL, "openrouter.ai") {
		// OpenRouter requiere formato "proveedor/modelo" (ej: deepseek/deepseek-chat)
		if model == "" || strings.Contains(model, "ozyassist") || strings.Contains(model, "local-model") || !strings.Contains(model, "/") {
			model = p.cfg.model
		}
		if strings.HasPrefix(model, "openrouter/") && strings.Count(model, "/") > 1 {
			model = strings.TrimPrefix(model, "openrouter/")
		}
	} else {
		if model == "" {
			model = p.cfg.model
		}
		model = strings.TrimPrefix(model, "openrouter/")
		model = strings.TrimPrefix(model, "openai/")
		model = strings.TrimPrefix(model, "deepseek/")
		model = strings.TrimPrefix(model, "lmstudio/")
		model = strings.TrimPrefix(model, "ollama/")
	}

	body := map[string]any{
		"model":    model,
		"messages": toOpenAIMessages(messages),
		"stream":   true,
	}
	if strings.Contains(p.cfg.baseURL, "api.openai.com") {
		body["stream_options"] = map[string]bool{"include_usage": false}
	}
	if opts.Temperature != 0 {
		body["temperature"] = opts.Temperature
	}
	if opts.MaxTokens > 0 {
		body["max_tokens"] = opts.MaxTokens
	} else if strings.Contains(p.cfg.baseURL, "openrouter.ai") {
		body["max_tokens"] = 1024
	} else {
		body["max_tokens"] = 2048
	}

	// Tool calling — OpenAI usa "tools" con "function.parameters" (JSON Schema)
	if len(opts.Tools) > 0 {
		body["tools"] = toOpenAITools(opts.Tools)
		body["tool_choice"] = "auto"
	}

	raw, _ := json.Marshal(body)

	var resp *http.Response
	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.baseURL+"/chat/completions", bytes.NewReader(raw))
		if err != nil {
			close(ch)
			return nil, fmt.Errorf("crear request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+p.cfg.apiKey)
		if strings.Contains(p.cfg.baseURL, "openrouter.ai") {
			req.Header.Set("HTTP-Referer", "http://localhost:1420")
			req.Header.Set("X-Title", "OzyAssist")
		}

		resp, err = p.cfg.client.Do(req)
		if err != nil {
			close(ch)
			return nil, fmt.Errorf("request fallido: %w", err)
		}

		if resp.StatusCode == 429 && attempt < maxRetries {
			resp.Body.Close()
			select {
			case <-ctx.Done():
				close(ch)
				return nil, ctx.Err()
			case <-time.After(time.Duration(1500*(attempt+1)) * time.Millisecond):
			}
			continue
		}
		break
	}

	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		close(ch)
		errStr := strings.TrimSpace(string(errBody))

		provName := p.Name()
		if provName == "" {
			provName = "LLM"
		}
		if resp.StatusCode == 402 {
			return nil, fmt.Errorf("%s (Créditos insuficientes - 402): %s", provName, errStr)
		}
		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("%s (No autorizado - 401): La API Key ingresada no es válida para %s. Revisa tu clave.", provName, provName)
		}
		if resp.StatusCode == 429 {
			return nil, fmt.Errorf("%s (Límite de peticiones - 429): %s", provName, errStr)
		}

		if len(errStr) > 300 {
			errStr = errStr[:300] + "..."
		}
		return nil, fmt.Errorf("API %s (status %d): %s", p.Name(), resp.StatusCode, errStr)
	}

	go p.readStream(ctx, ch, resp.Body)
	return ch, nil
}

// toOpenAITools convierte ToolDef agnóstico al formato nativo de OpenAI.
// OpenAI espera: {"type":"function","function":{"name","description","parameters":<JSON Schema>}}
func toOpenAITools(tools []ToolDef) []any {
	out := make([]any, len(tools))
	for i, t := range tools {
		schema := t.InputSchema
		if schema == nil {
			schema = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  schema,
			},
		}
	}
	return out
}

func (p *OpenAIProvider) readStream(ctx context.Context, ch chan<- StreamChunk, body io.ReadCloser) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	// Acumulador de tool calls indexado por index delta
	type tcAccum struct {
		id   string
		name string
		args string
	}
	toolAccum := make(map[int]*tcAccum)
	doneEmitted := false

	for scanner.Scan() {
		if ctx.Err() != nil {
			ch <- StreamChunk{Type: "error", Content: "cancelado"}
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			for _, acc := range toolAccum {
				var inputRaw json.RawMessage
				if acc.args != "" {
					inputRaw = json.RawMessage(acc.args)
				} else {
					inputRaw = json.RawMessage(`{}`)
				}
				ch <- StreamChunk{
					Type: "tool_call",
					ToolCall: &ToolCall{
						ID:    acc.id,
						Name:  acc.name,
						Input: inputRaw,
					},
				}
			}
			ch <- StreamChunk{Type: "done"}
			doneEmitted = true
			return
		}

		var sse struct {
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Role             string `json:"role"`
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
					Reasoning        string `json:"reasoning"`
					Thought          string `json:"thought"`
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
			// Pensamiento / Razonamiento (DeepSeek R1, OpenAI o1, Cohere, etc.)
			thought := choice.Delta.ReasoningContent
			if thought == "" {
				thought = choice.Delta.Reasoning
			}
			if thought == "" {
				thought = choice.Delta.Thought
			}
			if thought != "" {
				ch <- StreamChunk{Type: "thinking", Content: thought}
			}

			// Texto en streaming
			if choice.Delta.Content != "" {
				ch <- StreamChunk{Type: "text", Content: choice.Delta.Content}
			}

			// Acumular tool calls (llegan fragmentados en múltiples deltas)
			for _, tc := range choice.Delta.ToolCalls {
				acc, ok := toolAccum[tc.Index]
				if !ok {
					acc = &tcAccum{}
					toolAccum[tc.Index] = acc
				}
				if tc.ID != "" {
					acc.id = tc.ID
				}
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
				}
				acc.args += tc.Function.Arguments
			}

			// Si finish_reason viene seteado (sea 'tool_calls', 'stop', 'COMPLETE', etc.)
			if choice.FinishReason != "" {
				if len(toolAccum) > 0 {
					for _, acc := range toolAccum {
						var inputRaw json.RawMessage
						if acc.args != "" {
							inputRaw = json.RawMessage(acc.args)
						} else {
							inputRaw = json.RawMessage(`{}`)
						}
						ch <- StreamChunk{
							Type: "tool_call",
							ToolCall: &ToolCall{
								ID:    acc.id,
								Name:  acc.name,
								Input: inputRaw,
							},
						}
					}
					toolAccum = make(map[int]*tcAccum)
				}
				ch <- StreamChunk{Type: "done"}
				doneEmitted = true
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Type: "error", Content: err.Error()}
		return
	}

	if !doneEmitted {
		for _, acc := range toolAccum {
			var inputRaw json.RawMessage
			if acc.args != "" {
				inputRaw = json.RawMessage(acc.args)
			} else {
				inputRaw = json.RawMessage(`{}`)
			}
			ch <- StreamChunk{
				Type: "tool_call",
				ToolCall: &ToolCall{
					ID:    acc.id,
					Name:  acc.name,
					Input: inputRaw,
				},
			}
		}
		ch <- StreamChunk{Type: "done"}
	}
}

// toOpenAIMessages convierte el historial extendido al formato de OpenAI.
// Los mensajes role="tool" se envían con tool_call_id.
// Los mensajes role="assistant" con ToolCalls se envían con tool_calls[].
func toOpenAIMessages(msgs []Message) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "system":
			if strings.TrimSpace(m.Content) == "" {
				continue
			}
			out = append(out, map[string]any{
				"role":    "system",
				"content": m.Content,
			})
		case "user":
			if strings.TrimSpace(m.Content) == "" {
				continue
			}
			out = append(out, map[string]any{
				"role":    "user",
				"content": m.Content,
			})
		case "assistant":
			if len(m.ToolCalls) == 0 {
				if strings.TrimSpace(m.Content) == "" {
					continue
				}
				out = append(out, map[string]any{
					"role":    "assistant",
					"content": m.Content,
				})
			} else {
				tcs := make([]any, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					argStr := "{}"
					if tc.Input != nil {
						argStr = string(tc.Input)
					}
					tcs[i] = map[string]any{
						"id":   tc.ID,
						"type": "function",
						"function": map[string]any{
							"name":      tc.Name,
							"arguments": argStr,
						},
					}
				}
				obj := map[string]any{
					"role":       "assistant",
					"tool_calls": tcs,
				}
				if strings.TrimSpace(m.Content) != "" {
					obj["content"] = m.Content
				}
				out = append(out, obj)
			}
		case "tool":
			if m.ToolResult != nil {
				contentStr := m.ToolResult.Content
				if strings.TrimSpace(contentStr) == "" {
					contentStr = "ok"
				}
				out = append(out, map[string]any{
					"role":         "tool",
					"tool_call_id": m.ToolResult.ToolCallID,
					"content":      contentStr,
				})
			}
		}
	}
	return out
}
