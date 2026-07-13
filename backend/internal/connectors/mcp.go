package connectors

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ozyassist/backend/internal/db/models"
)

// MCPClient implementa el Model Context Protocol con soporte dual:
// - stdio: lanza un subproceso (npx, python, etc.) y habla por pipes
// - http:  POST a un endpoint HTTP JSON-RPC
type MCPClient struct {
	transport string // "stdio" | "http"

	// stdio
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	stderr  io.ReadCloser
	mu      sync.Mutex
	nextID  int

	// http
	httpClient *http.Client
	endpoint   string
	headers    map[string]string
}

// MCPRequest es un mensaje JSON-RPC 2.0 request/notification
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"` // nil para notificaciones
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse es un mensaje JSON-RPC 2.0 response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// InitializeParams describe las capacidades del cliente en initialize
type InitializeParams struct {
	ProtocolVersion string        `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ImplementationInfo `json:"clientInfo"`
}

type ClientCapabilities struct {
	Roots     *map[string]interface{} `json:"roots,omitempty"`
	Sampling  *map[string]interface{} `json:"sampling,omitempty"`
	Tools     *map[string]interface{} `json:"tools,omitempty"`
}

type ImplementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewMCPClient(connector *models.Connector) *MCPClient {
	c := &MCPClient{
		nextID: 1,
	}

	if strings.HasPrefix(connector.Endpoint, "http://") || strings.HasPrefix(connector.Endpoint, "https://") {
		c.transport = "http"
		c.httpClient = &http.Client{Timeout: 30 * time.Second}
		c.endpoint = connector.Endpoint
		c.headers = map[string]string{"Content-Type": "application/json"}
		if connector.AuthConfigJSON != "" && connector.AuthConfigJSON != "{}" {
			var auth struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal([]byte(connector.AuthConfigJSON), &auth); err == nil && auth.Token != "" {
				c.headers["Authorization"] = "Bearer " + auth.Token
			}
		}
	} else {
		c.transport = "stdio"
	}

	return c
}

// Connect inicia la conexión y ejecuta el handshake initialize.
// Para stdio: lanza el subproceso, configura pipes, envía initialize.
// Para http: solo envía initialize.
func (m *MCPClient) Connect(ctx context.Context, command string) error {
	switch m.transport {
	case "stdio":
		return m.connectStdio(ctx, command)
	case "http":
		return m.connectHTTP(ctx)
	default:
		return fmt.Errorf("unknown transport: %s", m.transport)
	}
}

func (m *MCPClient) connectStdio(ctx context.Context, command string) error {
	parts := splitCommand(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	m.cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)

	stdin, err := m.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	m.stdin = stdin

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	m.scanner = bufio.NewScanner(stdout)
	m.scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	m.stderr = stderr

	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 0, 4096), 64*1024)
		for sc.Scan() {
			log.Printf("[mcp-stderr] %s", sc.Text())
		}
	}()

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	// Handshake: initialize
	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	return m.handshake(initCtx)
}

func (m *MCPClient) connectHTTP(ctx context.Context) error {
	return m.handshake(ctx)
}

func (m *MCPClient) handshake(ctx context.Context) error {
	initParams := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities: ClientCapabilities{},
		ClientInfo: ImplementationInfo{
			Name:    "ozyassist",
			Version: "0.1.0",
		},
	}

	result, err := m.Call(ctx, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("initialize handshake: %w", err)
	}
	log.Printf("[mcp] initialize OK: %s", string(result))

	// Send initialized notification (no id = notification)
	_, err = m.sendNotification(ctx, "notifications/initialized", nil)
	if err != nil {
		// Non-fatal: some servers accept this as a notification even on error
		log.Printf("[mcp] initialized notification (non-fatal): %v", err)
	}

	return nil
}

// sendNotification envía una notificación JSON-RPC (sin id).
// Para stdio escribe al pipe; para http envía POST (ignorando respuesta).
func (m *MCPClient) sendNotification(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return m.send(ctx, req)
}

// Call envía un request JSON-RPC y espera la respuesta con id correspondiente.
func (m *MCPClient) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	m.mu.Lock()
	id := m.nextID
	m.nextID++
	m.mu.Unlock()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}
	return m.send(ctx, req)
}

func (m *MCPClient) send(ctx context.Context, req MCPRequest) (json.RawMessage, error) {
	switch m.transport {
	case "stdio":
		return m.sendStdio(ctx, req)
	case "http":
		return m.sendHTTP(ctx, req)
	default:
		return nil, fmt.Errorf("unknown transport: %s", m.transport)
	}
}

func (m *MCPClient) sendStdio(ctx context.Context, req MCPRequest) (json.RawMessage, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	m.mu.Lock()
	_, err = fmt.Fprintln(m.stdin, string(data))
	m.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("write stdin: %w", err)
	}

	if req.ID == nil {
		// Notification — no esperamos respuesta
		return nil, nil
	}

	// Leer líneas hasta encontrar la respuesta con el id que coincide
	for m.scanner.Scan() {
		line := m.scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var resp MCPResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			log.Printf("[mcp] skip unparseable line: %s", string(line))
			continue
		}

		if resp.ID != nil && *resp.ID == *req.ID {
			if resp.Error != nil {
				return nil, fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
			}
			return resp.Result, nil
		}

		// Notification (sin id o id no coincide) — log y seguimos
		if resp.ID == nil {
			log.Printf("[mcp] notification: method=%s", extractMethod(line))
		}
	}

	if err := m.scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan stdout: %w", err)
	}
	return nil, fmt.Errorf("connection closed before response for id=%d", *req.ID)
}

func (m *MCPClient) sendHTTP(ctx context.Context, req MCPRequest) (json.RawMessage, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", m.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	for k, v := range m.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var mcpResp MCPResponse
	if err := json.Unmarshal(body, &mcpResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}
	return mcpResp.Result, nil
}

// Tools lista las herramientas disponibles.
func (m *MCPClient) Tools(ctx context.Context) (json.RawMessage, error) {
	return m.Call(ctx, "tools/list", nil)
}

// CallTool invoca una herramienta específica.
func (m *MCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (json.RawMessage, error) {
	return m.Call(ctx, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
}

// Close termina la conexión.
// Para stdio: envía shutdown, mata el proceso.
// Para http: no-op.
func (m *MCPClient) Close() error {
	if m.transport == "http" {
		return nil
	}

	if m.cmd != nil && m.cmd.Process != nil {
		// Intentar shutdown graceful, después matar
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := m.Call(ctx, "shutdown", nil); err != nil {
			log.Printf("[mcp] shutdown error (killing anyway): %v", err)
		}
		if err := m.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill process: %w", err)
		}
		_ = m.cmd.Wait()
	}
	return nil
}

// splitCommand separa un comando respetando comillas.
func splitCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == ' ' && !inSingle && !inDouble:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// extractMethod extrae el método de una línea JSON (para logging).
func extractMethod(line []byte) string {
	var msg struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(line, &msg); err == nil {
		return msg.Method
	}
	return "unknown"
}
