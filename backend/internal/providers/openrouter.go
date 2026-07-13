package providers

import "context"

type OpenRouterProvider struct{}

func NewOpenRouter(apiKey string) *OpenRouterProvider {
	return &OpenRouterProvider{}
}

func (p *OpenRouterProvider) Name() string          { return "openrouter" }
func (p *OpenRouterProvider) SupportsTools() bool    { return true }
func (p *OpenRouterProvider) Models() []string       { return []string{"openrouter/auto"} }

func (p *OpenRouterProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}
