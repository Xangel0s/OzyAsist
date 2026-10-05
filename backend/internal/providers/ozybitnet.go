package providers

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"
)

// OzyBitNetProvider integrates Ozy-BitCoconut (port 8765) and OzyTalk (port 8766)
// into OzyAssist's native cognitive provider architecture.
type OzyBitNetProvider struct {
	name            string
	talkProvider    *OpenAIProvider
	coconutProvider *OpenAIProvider
	talkURL         string
	coconutURL      string
	models          []string
}

// GetOzyBitNetURL returns the default URL for OzyBitNet / OzyTalk
func GetOzyBitNetURL() string {
	if url := strings.TrimSpace(os.Getenv("OZYTALK_URL")); url != "" {
		return url
	}
	if url := strings.TrimSpace(os.Getenv("OZYBITNET_URL")); url != "" {
		return url
	}
	return "http://localhost:8766/v1"
}

// GetOzyCoconutURL returns the continuous latent Coconut URL
func GetOzyCoconutURL() string {
	if url := strings.TrimSpace(os.Getenv("OZYCOCONUT_URL")); url != "" {
		return url
	}
	return "http://localhost:8765/v1"
}

// NewOzyBitNet initializes the native OzyBitNet provider with default name "ozytalk"
func NewOzyBitNet(baseURL string) *OzyBitNetProvider {
	return NewOzyBitNetWithName("ozytalk", baseURL)
}

// NewOzyBitNetWithName initializes with a custom provider display name ("ozytalk" or "ozybitnet")
func NewOzyBitNetWithName(name string, baseURL string) *OzyBitNetProvider {
	talkURL := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(talkURL, "/v1") {
		talkURL = talkURL + "/v1"
	}

	coconutURL := strings.TrimRight(GetOzyCoconutURL(), "/")
	if !strings.HasSuffix(coconutURL, "/v1") {
		coconutURL = coconutURL + "/v1"
	}

	pTalk := NewOpenAI("not-needed")
	pTalk.cfg.providerName = name
	pTalk.cfg.baseURL = talkURL
	pTalk.cfg.model = "ozytalk-1.58b"

	pCoconut := NewOpenAI("not-needed")
	pCoconut.cfg.providerName = "ozy-bitcoconut"
	pCoconut.cfg.baseURL = coconutURL
	pCoconut.cfg.model = "ozy-bitcoconut-1.58b"

	ob := &OzyBitNetProvider{
		name:            name,
		talkProvider:    pTalk,
		coconutProvider: pCoconut,
		talkURL:         talkURL,
		coconutURL:      coconutURL,
		models:          []string{"ozytalk-1.58b", "ozy-bitcoconut-1.58b"},
	}

	return ob
}

func (p *OzyBitNetProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "ozytalk"
}

func (p *OzyBitNetProvider) SupportsTools() bool {
	return true
}

func (p *OzyBitNetProvider) Models() []string {
	return p.models
}

// StreamCompletion dynamically routes requests:
// - "ozy-bitcoconut-1.58b" or latent healing queries route to port 8765
// - "ozytalk-1.58b" or conversational queries route to port 8766
func (p *OzyBitNetProvider) StreamCompletion(ctx context.Context, messages []Message, opts CompletionOptions) (<-chan StreamChunk, error) {
	targetModel := strings.ToLower(strings.TrimSpace(opts.Model))

	// Check if Coconut continuous latent reasoning is explicitly requested
	if strings.Contains(targetModel, "coconut") || strings.Contains(targetModel, "latent") {
		opts.Model = "ozy-bitcoconut-1.58b"
		return p.coconutProvider.StreamCompletion(ctx, messages, opts)
	}

	// Default to OzyTalk conversational model
	if opts.Model == "" {
		opts.Model = "ozytalk-1.58b"
	}
	return p.talkProvider.StreamCompletion(ctx, messages, opts)
}

// HealthCheck verifies whether either or both local servers are reachable
func (p *OzyBitNetProvider) HealthCheck() (talkAlive bool, coconutAlive bool) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}

	// Check OzyTalk (port 8766)
	cleanTalk := strings.TrimSuffix(p.talkURL, "/v1")
	if resp, err := client.Get(cleanTalk + "/health"); err == nil {
		talkAlive = (resp.StatusCode == http.StatusOK)
		_ = resp.Body.Close()
	}

	// Check Ozy-BitCoconut (port 8765)
	cleanCoconut := strings.TrimSuffix(p.coconutURL, "/v1")
	if resp, err := client.Get(cleanCoconut + "/health"); err == nil {
		coconutAlive = (resp.StatusCode == http.StatusOK)
		_ = resp.Body.Close()
	}

	return talkAlive, coconutAlive
}
