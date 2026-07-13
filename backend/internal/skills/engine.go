package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/ozyassist/backend/internal/db/models"
)

type Executor struct {
	HTTPClient *http.Client
}

func NewExecutor() *Executor {
	return &Executor{HTTPClient: &http.Client{Timeout: 15 * time.Second}}
}

type Result struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

func (e *Executor) Execute(skill *models.Skill, input string) (*Result, error) {
	switch skill.ExecutionType {
	case "script":
		return e.execScript(skill, input)
	case "prompt_template":
		return e.execPromptTemplate(skill, input)
	case "api_call":
		return e.execAPICall(skill, input)
	default:
		return nil, fmt.Errorf("unknown execution type: %s", skill.ExecutionType)
	}
}

func (e *Executor) execScript(skill *models.Skill, input string) (*Result, error) {
	var config struct {
		Command string `json:"command"`
		Shell   string `json:"shell"`
		Timeout int    `json:"timeout"`
	}
	json.Unmarshal([]byte(skill.ConfigJSON), &config)

	if config.Command == "" {
		return nil, fmt.Errorf("no command configured for script skill")
	}

	shell := config.Shell
	if shell == "" {
		shell = "powershell"
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, "-c", config.Command)
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.String() != "" {
		output += "\nSTDERR: " + stderr.String()
	}

	if err != nil {
		return &Result{Success: false, Output: output, Error: err.Error()}, nil
	}
	return &Result{Success: true, Output: output}, nil
}

// normalizeTemplate convierte {input}/{name} a sintaxis {{.input}}/{{.name}}
// para compatibilidad con el formato simple de skills.
func normalizeTemplate(text string) string {
	text = strings.ReplaceAll(text, "{input}", "{{.input}}")
	text = strings.ReplaceAll(text, "{name}", "{{.name}}")
	return text
}

func (e *Executor) execPromptTemplate(skill *models.Skill, input string) (*Result, error) {
	tmpl, err := template.New("skill").Parse(normalizeTemplate(skill.ConfigJSON))
	if err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("invalid template: %v", err)}, nil
	}

	data := map[string]string{"input": input, "name": skill.Name}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("template execution: %v", err)}, nil
	}

	return &Result{Success: true, Output: buf.String()}, nil
}

func (e *Executor) execAPICall(skill *models.Skill, input string) (*Result, error) {
	var config struct {
		URL    string            `json:"url"`
		Method string            `json:"method"`
		Headers map[string]string `json:"headers"`
		Body   string            `json:"bodyTemplate"`
	}
	json.Unmarshal([]byte(skill.ConfigJSON), &config)

	method := config.Method
	if method == "" {
		method = "POST"
	}

	bodyStr := strings.ReplaceAll(config.Body, "{{input}}", input)
	body := bytes.NewReader([]byte(bodyStr))

	req, err := http.NewRequest(method, config.URL, body)
	if err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("create request: %v", err)}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("api call: %v", err)}, nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return &Result{Success: resp.StatusCode < 300, Output: string(respBody)}, nil
}
