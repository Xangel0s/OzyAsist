package providers

import "context"

type AnthropicProvider struct{}

func NewAnthropic(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{}
}

func (p *AnthropicProvider) Name() string           { return "anthropic" }
func (p *AnthropicProvider) SupportsTools() bool     { return true }
func (p *AnthropicProvider) Models() []string        { return []string{"claude-sonnet-5", "claude-haiku-5"} }

func (p *AnthropicProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}
