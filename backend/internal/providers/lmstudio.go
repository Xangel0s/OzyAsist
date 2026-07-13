package providers

import "context"

type LMStudioProvider struct{}

func NewLMStudio(baseURL string) *LMStudioProvider {
	return &LMStudioProvider{}
}

func (p *LMStudioProvider) Name() string          { return "lmstudio" }
func (p *LMStudioProvider) SupportsTools() bool    { return false }
func (p *LMStudioProvider) Models() []string       { return []string{"local-model"} }

func (p *LMStudioProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}
