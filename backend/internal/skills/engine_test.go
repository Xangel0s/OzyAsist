package skills

import (
	"testing"

	"github.com/ozyassist/backend/internal/db/models"
)

func TestPromptTemplateSkillExecution(t *testing.T) {
	executor := NewExecutor()
	skill := &models.Skill{
		Name:          "summarize",
		ExecutionType: "prompt_template",
		ConfigJSON:    "Resumen de: {input}",
	}

	res, err := executor.Execute(skill, "texto largo de ejemplo")
	if err != nil {
		t.Fatalf("Unexpected error executing prompt template skill: %v", err)
	}

	if !res.Success {
		t.Errorf("Expected success, got error: %s", res.Error)
	}

	expected := "Resumen de: texto largo de ejemplo"
	if res.Output != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, res.Output)
	}
}

func TestScriptSkillExecution(t *testing.T) {
	executor := NewExecutor()
	skill := &models.Skill{
		Name:          "echo-script",
		ExecutionType: "script",
		ConfigJSON:    `{"command": "Write-Output 'Skill Output'", "shell": "powershell"}`,
	}

	res, err := executor.Execute(skill, "")
	if err != nil {
		t.Fatalf("Unexpected error executing script skill: %v", err)
	}

	if !res.Success {
		t.Errorf("Expected success script execution, got error: %s", res.Error)
	}
}

func TestInvalidExecutionTypeFallback(t *testing.T) {
	executor := NewExecutor()
	skill := &models.Skill{
		Name:          "invalid",
		ExecutionType: "unknown_type",
	}

	_, err := executor.Execute(skill, "test")
	if err == nil {
		t.Errorf("Expected error for unknown execution type")
	}
}
