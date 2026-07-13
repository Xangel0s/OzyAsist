package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// Sandbox valida que todas las operaciones de archivo estén dentro de ProjectRoot.
type Sandbox struct {
	ProjectRoot string
}

func NewSandbox(projectRoot string) (*Sandbox, error) {
	absRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("project root not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project root is not a directory")
	}
	return &Sandbox{ProjectRoot: absRoot}, nil
}

func (s *Sandbox) Resolve(path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.ProjectRoot, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	rel, err := filepath.Rel(s.ProjectRoot, abs)
	if err != nil {
		return "", fmt.Errorf("path outside sandbox: %w", err)
	}
	if rel[0] == '.' || rel[0] == os.PathSeparator {
		return "", fmt.Errorf("path traversal detected: %s", rel)
	}
	return abs, nil
}

func (s *Sandbox) ReadFile(path string) (string, error) {
	full, err := s.Resolve(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(data), nil
}

func (s *Sandbox) WriteFile(path, content string) error {
	full, err := s.Resolve(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	return os.WriteFile(full, []byte(content), 0644)
}

func (s *Sandbox) ListFiles(dir string) ([]string, error) {
	full, err := s.Resolve(dir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	var paths []string
	for _, e := range entries {
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	return paths, nil
}
