package agent

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/providers"
)

type mockLLMProvider struct {
	response string
}

func (m *mockLLMProvider) StreamCompletion(ctx context.Context, messages []providers.Message, opts providers.CompletionOptions) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk, 2)
	go func() {
		defer close(ch)
		ch <- providers.StreamChunk{Type: "text", Content: m.response}
		ch <- providers.StreamChunk{Type: "done"}
	}()
	return ch, nil
}

func (m *mockLLMProvider) Name() string          { return "mock" }
func (m *mockLLMProvider) SupportsTools() bool   { return false }
func (m *mockLLMProvider) Models() []string      { return []string{"mock-model"} }

func TestHealer_AttemptRecoverySuccess(t *testing.T) {
	mockProv := &mockLLMProvider{response: "npm install && npm run build"}
	verifier := NewVerifier()
	healer := NewHealer(mockProv, verifier)

	task := models.AgentTask{
		ID:     uuid.NewString(),
		Title:  "Build Frontend",
		Prompt: "Compile the React frontend",
	}

	step := &models.TaskStep{
		ID:         uuid.NewString(),
		StepOrder:  1,
		ActionType: "shell_exec",
		Payload:    "npm run build",
		RetryCount: 0,
		MaxRetries: 3,
	}

	result, err := healer.AttemptRecovery(context.Background(), task, step, "vite: not found", "Error: command failed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected recovery success, got failure: %s", result.FailureReason)
	}

	if result.PatchApplied != "npm install && npm run build" {
		t.Errorf("unexpected patch applied: %s", result.PatchApplied)
	}

	if step.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", step.RetryCount)
	}
}

func TestHealer_MaxRetriesExceeded(t *testing.T) {
	mockProv := &mockLLMProvider{response: "fixed command"}
	verifier := NewVerifier()
	healer := NewHealer(mockProv, verifier)

	task := models.AgentTask{ID: uuid.NewString()}
	step := &models.TaskStep{
		ID:         uuid.NewString(),
		RetryCount: 3,
		MaxRetries: 3,
	}

	result, err := healer.AttemptRecovery(context.Background(), task, step, "persistent error", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Fatal("expected failure due to max retries exceeded")
	}
}
