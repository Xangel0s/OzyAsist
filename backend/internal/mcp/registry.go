package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ServerConfig defines how to start an MCP server
type ServerConfig struct {
	ID      string
	Name    string
	Command string
	Args    []string
	Env     []string
}

// ServerStatus describes the live state of an MCP server
type ServerStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	Status    string `json:"status"` // "online", "error", "stopped"
	ToolCount int    `json:"tool_count"`
	Error     string `json:"error,omitempty"`
}

// Registry manages the lifecycle of multiple MCP clients
type Registry struct {
	clients map[string]*Client
	tools   map[string][]Tool // Cached tools per server ID
	configs map[string]ServerConfig
	errors  map[string]string
	mu      sync.RWMutex
}

// DefaultRegistry is the singleton instance used by the agent
var DefaultRegistry = &Registry{
	clients: make(map[string]*Client),
	tools:   make(map[string][]Tool),
	configs: make(map[string]ServerConfig),
	errors:  make(map[string]string),
}

// Init loads all configured MCP connectors from the database (stub for legacy API)
func Init(ctx context.Context) error {
	return nil
}

// InitFromConfigFile busca y carga el archivo de configuración mcp_servers.json
func (r *Registry) InitFromConfigFile(ctx context.Context, path string) error {
	if path == "" {
		path = FindDefaultConfigFile()
	}
	if path == "" {
		return nil
	}

	configs, err := LoadConfigFile(path)
	if err != nil {
		return err
	}

	r.InitFromDB(ctx, configs)
	return nil
}

// InitFromDB loads servers given a list of configs (called by main/server)
func (r *Registry) InitFromDB(ctx context.Context, configs []ServerConfig) {
	for _, cfg := range configs {
		if err := r.AddServer(ctx, cfg); err != nil {
			log.Printf("[MCP] Error iniciando servidor %s: %v", cfg.Name, err)
		} else {
			log.Printf("[MCP] Servidor conectado: %s", cfg.Name)
		}
	}
}

// AddServer starts a new MCP server and registers its tools
func (r *Registry) AddServer(ctx context.Context, config ServerConfig) error {
	r.mu.Lock()
	if _, exists := r.clients[config.ID]; exists {
		r.mu.Unlock()
		return fmt.Errorf("el servidor MCP %s ya está en ejecución", config.ID)
	}
	r.configs[config.ID] = config
	delete(r.errors, config.ID)
	r.mu.Unlock()

	transport := NewStdioTransport(config.Command, config.Args, config.Env)
	client := NewClient(config.Name, transport)

	if err := client.Start(ctx); err != nil {
		r.mu.Lock()
		r.errors[config.ID] = err.Error()
		r.mu.Unlock()
		return fmt.Errorf("fallo al arrancar servidor %s: %w", config.Name, err)
	}

	// Breve pausa para dar tiempo al proceso stdio a quedar listo
	time.Sleep(500 * time.Millisecond)

	_, err := client.Initialize(ctx)
	if err != nil {
		client.Stop()
		r.mu.Lock()
		r.errors[config.ID] = err.Error()
		r.mu.Unlock()
		return fmt.Errorf("fallo al inicializar protocolo MCP en %s: %w", config.Name, err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		tools = []Tool{}
	}

	r.mu.Lock()
	r.clients[config.ID] = client
	r.tools[config.ID] = tools
	delete(r.errors, config.ID)
	r.mu.Unlock()

	return nil
}

// RemoveServer stops a server and removes it from the registry
func (r *Registry) RemoveServer(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, exists := r.clients[id]
	if !exists {
		delete(r.configs, id)
		delete(r.errors, id)
		return fmt.Errorf("servidor MCP %s no encontrado", id)
	}

	client.Stop()
	delete(r.clients, id)
	delete(r.tools, id)
	delete(r.configs, id)
	delete(r.errors, id)
	return nil
}

// Reload reinicia y recarga en caliente los servidores MCP a partir de nuevas configuraciones
func (r *Registry) Reload(ctx context.Context, configs []ServerConfig) error {
	r.mu.RLock()
	currentIDs := make(map[string]bool)
	for id := range r.configs {
		currentIDs[id] = true
	}
	r.mu.RUnlock()

	newIDs := make(map[string]bool)
	for _, cfg := range configs {
		newIDs[cfg.ID] = true
	}

	// Detener los servidores que ya no existan en la nueva configuración
	for id := range currentIDs {
		if !newIDs[id] {
			_ = r.RemoveServer(id)
		}
	}

	// Reiniciar o agregar los configurados
	for _, cfg := range configs {
		_ = r.RemoveServer(cfg.ID)
		if err := r.AddServer(ctx, cfg); err != nil {
			log.Printf("[MCP] Error en recarga de %s: %v", cfg.Name, err)
		}
	}

	return nil
}

// Shutdown cierra de forma segura todos los servidores MCP en ejecución
func (r *Registry) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, client := range r.clients {
		if client != nil {
			_ = client.Stop()
		}
		delete(r.clients, id)
		delete(r.tools, id)
	}
	r.errors = make(map[string]string)
}

// GetServerStatus retorna el estado actual de todos los servidores MCP configurados
func (r *Registry) GetServerStatus() []ServerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var statuses []ServerStatus
	for id, cfg := range r.configs {
		st := ServerStatus{
			ID:      id,
			Name:    cfg.Name,
			Command: cfg.Command,
		}

		if _, active := r.clients[id]; active {
			st.Status = "online"
			st.ToolCount = len(r.tools[id])
		} else if errMsg, hasErr := r.errors[id]; hasErr {
			st.Status = "error"
			st.Error = errMsg
		} else {
			st.Status = "stopped"
		}

		statuses = append(statuses, st)
	}

	return statuses
}

// GetAllTools returns a flattened list of all tools from all registered servers.
// Returns a map of (serverID_toolName) -> Tool
func (r *Registry) GetAllTools() map[string]Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allTools := make(map[string]Tool)
	for serverID, tools := range r.tools {
		for _, t := range tools {
			prefixedName := fmt.Sprintf("mcp_%s_%s", serverID, t.Name)
			tCopy := t
			tCopy.Name = prefixedName
			allTools[prefixedName] = tCopy
		}
	}
	return allTools
}

// CallTool routes a tool call to the appropriate server
func (r *Registry) CallTool(ctx context.Context, prefixedName string, arguments interface{}) (*CallToolResult, error) {
	var serverID, toolName string
	parts := splitPrefix(prefixedName, "mcp_")
	if parts == nil {
		return nil, fmt.Errorf("formato inválido de herramienta MCP: %s", prefixedName)
	}
	serverID, toolName = parts[0], parts[1]

	r.mu.RLock()
	client, exists := r.clients[serverID]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("servidor MCP '%s' no está conectado para la herramienta '%s'", serverID, toolName)
	}

	return client.CallTool(ctx, toolName, arguments)
}

func splitPrefix(s, prefix string) []string {
	if len(s) <= len(prefix) {
		return nil
	}
	s = s[len(prefix):] // strip 'mcp_'
	
	// find the first underscore after
	for i, c := range s {
		if c == '_' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
