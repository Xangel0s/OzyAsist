package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Caps1 almacena memoria explícita del proyecto (archivos .md en ProjectRoot).
// Se usa como contexto directo para el LLM en cada chat asociado al proyecto.

type Caps1Entry struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updatedAt"`
}

type Caps1Store struct {
	mu     sync.RWMutex
	roots  map[string]string // projectID -> rootPath
	cache  map[string]Caps1Entry
}

var caps1 *Caps1Store

func init() {
	caps1 = &Caps1Store{
		roots: make(map[string]string),
		cache: make(map[string]Caps1Entry),
	}
}

func GetCaps1() *Caps1Store {
	return caps1
}

func (c *Caps1Store) RegisterProject(projectID, rootPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roots[projectID] = rootPath
}

func (c *Caps1Store) UnregisterProject(projectID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.roots, projectID)
}

// GetContext obtiene el contenido de todos los .md en ProjectRoot para usar como contexto.
func (c *Caps1Store) GetContext(projectID string) ([]Caps1Entry, error) {
	c.mu.RLock()
	rootPath, ok := c.roots[projectID]
	c.mu.RUnlock()

	if !ok || rootPath == "" {
		return nil, nil
	}

	var entries []Caps1Entry
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(info.Name()) != ".md" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		rel, _ := filepath.Rel(rootPath, path)
		entries = append(entries, Caps1Entry{
			Path:      rel,
			Content:   string(content),
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		})
		return nil
	})
	return entries, err
}

// WriteMd escribe o actualiza un archivo .md dentro del ProjectRoot.
func (c *Caps1Store) WriteMd(projectID, relPath, content string) error {
	c.mu.RLock()
	rootPath, ok := c.roots[projectID]
	c.mu.RUnlock()

	if !ok || rootPath == "" {
		return fmt.Errorf("project %s not registered", projectID)
	}
	if filepath.Ext(relPath) != ".md" {
		return fmt.Errorf("only .md files are allowed")
	}

	fullPath := filepath.Join(rootPath, relPath)
	// Security: ensure the resulting path is still within rootPath
	absPath, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(rootPath)
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || rel[0:1] == "." {
		return fmt.Errorf("path traversal detected")
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}
