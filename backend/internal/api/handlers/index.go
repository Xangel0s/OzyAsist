package handlers

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
)

type FileTreeNode struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Path     string          `json:"path"`
	Size     int64           `json:"size,omitempty"`
	Children []*FileTreeNode `json:"children,omitempty"`
}

var codeExtensions = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".py": true, ".rs": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
	".css": true, ".html": true, ".json": true, ".yaml": true, ".yml": true,
	".toml": true, ".md": true, ".sql": true, ".proto": true,
}

var excludedDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true,
	"target": true, "__pycache__": true, ".next": true, "coverage": true,
	".vscode": true, ".idea": true, "vendor": true,
}

func IndexProject(c *gin.Context) {
	projectID := c.Param("id")
	project, err := db.GetProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	if project.RootPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project has no root_path"})
		return
	}

	// Build file tree
	tree := &FileTreeNode{
		Name:     filepath.Base(project.RootPath),
		Type:     "folder",
		Path:     "/",
		Children: []*FileTreeNode{},
	}

	var imports []models.CodeGraphEdge
	fileCount := 0

	filepath.WalkDir(project.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(project.RootPath, path)
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}

		// Skip excluded dirs
		for _, part := range strings.Split(relPath, "/") {
			if excludedDirs[part] {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}

		if d.IsDir() {
			tree.Children = append(tree.Children, &FileTreeNode{
				Name: d.Name(),
				Type: "folder",
				Path: relPath,
			})
			return nil
		}

		fileCount++
		info, err := d.Info()
		if err != nil {
			return nil
		}

		node := &FileTreeNode{
			Name: d.Name(),
			Type: "file",
			Path: relPath,
			Size: info.Size(),
		}
		tree.Children = append(tree.Children, node)

		// Extract imports for code files
		ext := strings.ToLower(filepath.Ext(path))
		if codeExtensions[ext] {
			fileImports := extractImports(path, ext, project.RootPath)
			for _, imp := range fileImports {
				imports = append(imports, models.CodeGraphEdge{
					ID:         uuid.NewString(),
					ProjectID:  projectID,
					FromSymbol: relPath,
					ToSymbol:   imp,
					EdgeType:   "imports",
					CreatedAt:  time.Now(),
				})
			}
		}

		return nil
	})

	// Flatten tree — append files/folders into tree structure
	treeJSON, _ := json.Marshal(tree)

	// Update project with tree
	project.FileTreeJSON = string(treeJSON)
	if err := db.UpdateProject(projectID, project); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update tree: " + err.Error()})
		return
	}

	// Clear old edges and insert new
	db.ClearGraphEdges(projectID)
	for _, edge := range imports {
		db.InsertGraphEdge(&edge)
	}

	c.JSON(http.StatusOK, gin.H{
		"tree":      tree,
		"files":     fileCount,
		"imports":   len(imports),
	})
}

func GetProjectTree(c *gin.Context) {
	projectID := c.Param("id")
	project, err := db.GetProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	if project.FileTreeJSON == "" {
		c.JSON(http.StatusOK, gin.H{"tree": nil, "indexed": false})
		return
	}
	var tree FileTreeNode
	if err := json.Unmarshal([]byte(project.FileTreeJSON), &tree); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parse tree: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tree": &tree, "indexed": true})
}

func GetProjectGraph(c *gin.Context) {
	projectID := c.Param("id")
	filePath := strings.TrimPrefix(c.Param("filepath"), "/")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "filepath is required"})
		return
	}

	edges, err := db.GetGraphNeighbors(projectID, filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type Neighbor struct {
		File     string `json:"file"`
		Relation string `json:"relation"`
	}
	var neighbors []Neighbor
	seen := map[string]bool{}

	for _, e := range edges {
		if e.FromSymbol == filePath {
			if !seen[e.ToSymbol] {
				neighbors = append(neighbors, Neighbor{File: e.ToSymbol, Relation: "imports"})
				seen[e.ToSymbol] = true
			}
		} else if e.ToSymbol == filePath {
			if !seen[e.FromSymbol] {
				neighbors = append(neighbors, Neighbor{File: e.FromSymbol, Relation: "imported_by"})
				seen[e.FromSymbol] = true
			}
		}
	}

	if neighbors == nil {
		neighbors = []Neighbor{}
	}

	c.JSON(http.StatusOK, gin.H{
		"file":      filePath,
		"neighbors": neighbors,
	})
}

func GetAllProjectGraph(c *gin.Context) {
	projectID := c.Param("id")
	edges, err := db.GetAllGraphEdges(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, edges)
}

func GetFileContent(c *gin.Context) {
	projectID := c.Param("id")
	relPath := c.Query("path")
	if relPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path query param required"})
		return
	}

	project, err := db.GetProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found: " + err.Error()})
		return
	}
	if project.RootPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project has no root_path"})
		return
	}

	cleanPath := filepath.Clean(relPath)
	fullPath := filepath.Join(project.RootPath, cleanPath)

	if !strings.HasPrefix(fullPath, filepath.Clean(project.RootPath)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "path traversal not allowed"})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found: " + fullPath + " (" + err.Error() + ")"})
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is a directory"})
		return
	}

	if info.Size() > 1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (>1MB)"})
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read file: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":    cleanPath,
		"content": string(content),
		"size":    info.Size(),
	})
}

type FileUpload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func UploadProjectFiles(c *gin.Context) {
	projectID := c.Param("id")
	project, err := db.GetProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	var files []FileUpload
	if err := c.ShouldBindJSON(&files); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	db.ClearGraphEdges(projectID)

	root := &FileTreeNode{Name: project.Name, Type: "folder", Path: "."}
	importCount := 0

	for _, f := range files {
		parts := strings.Split(f.Path, "/")
		current := root
		for i, part := range parts {
			if i == len(parts)-1 {
				current.Children = append(current.Children, &FileTreeNode{
					Name: part,
					Type: "file",
					Path: f.Path,
				})
			} else {
				found := false
				for _, child := range current.Children {
					if child.Name == part && child.Type == "folder" {
						current = child
						found = true
						break
					}
				}
				if !found {
					newDir := &FileTreeNode{Name: part, Type: "folder", Path: strings.Join(parts[:i+1], "/")}
					current.Children = append(current.Children, newDir)
					current = newDir
				}
			}
		}

		ext := filepath.Ext(f.Path)
		if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".py" || ext == ".rs" {
			imports := extractImportsFromContent(f.Path, ext, f.Content)
			for _, imp := range imports {
				edge := &models.CodeGraphEdge{
					ID:         uuid.NewString(),
					ProjectID:  projectID,
					FromSymbol: f.Path,
					ToSymbol:   imp,
					EdgeType:   "import",
					CreatedAt:  time.Now(),
				}
				db.InsertGraphEdge(edge)
				importCount++
			}
		}
	}

	treeJSON, _ := json.Marshal(root)
	db.SaveFileTree(projectID, string(treeJSON))

	c.JSON(http.StatusOK, gin.H{
		"files":   len(files),
		"imports": importCount,
	})
}

func extractImportsFromContent(filePath, ext, content string) []string {
	var imports []string
	lines := strings.Split(content, "\n")
	relPath := filePath

	switch ext {
	case ".go":
		inImportBlock := false
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import (") {
				inImportBlock = true
				continue
			}
			if inImportBlock {
				if line == ")" {
					inImportBlock = false
					continue
				}
				if m := goImportRe.FindStringSubmatch(line); len(m) > 1 {
					if resolved := resolveImportFromContent(m[1], relPath); resolved != "" {
						imports = append(imports, resolved)
					}
				}
			} else if strings.HasPrefix(line, "import ") {
				if m := goImportRe.FindStringSubmatch(line); len(m) > 1 {
					if resolved := resolveImportFromContent(m[1], relPath); resolved != "" {
						imports = append(imports, resolved)
					}
				}
			}
		}
	case ".ts", ".tsx", ".js", ".jsx":
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "import ") && !strings.Contains(line, "require(") {
				continue
			}
			if m := tsImportRe.FindStringSubmatch(line); len(m) > 1 {
				if resolved := resolveImportFromContent(m[1], relPath); resolved != "" {
					imports = append(imports, resolved)
				}
			}
		}
	case ".py":
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "from ") && !strings.HasPrefix(line, "import ") {
				continue
			}
			matches := pyImportRe.FindStringSubmatch(line)
			if len(matches) > 2 {
				mod := matches[1]
				if mod == "" {
					mod = matches[2]
				}
				if resolved := resolveImportFromContent(strings.ReplaceAll(mod, ".", "/"), relPath); resolved != "" {
					imports = append(imports, resolved)
				}
			}
		}
	case ".rs":
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "use ") {
				continue
			}
			m := rustImportRe.FindStringSubmatch(line)
			if len(m) > 1 {
				mod := strings.TrimSuffix(m[1], "::*")
				mod = strings.ReplaceAll(mod, "::", "/")
				if resolved := resolveImportFromContent(mod, relPath); resolved != "" {
					imports = append(imports, resolved)
				}
			}
		}
	}
	return imports
}

func resolveImportFromContent(importPath, fromFile string) string {
	importPath = strings.TrimSuffix(importPath, `"`)
	importPath = strings.TrimSuffix(importPath, `'`)
	if importPath == "" || strings.Contains(importPath, "://") {
		return ""
	}
	if strings.HasPrefix(importPath, ".") {
		fromDir := filepath.Dir(fromFile)
		resolved := filepath.Clean(filepath.Join(fromDir, importPath))
		return filepath.ToSlash(resolved)
	}
	return ""
}

func extractImports(filePath, ext, projectRoot string) []string {
	var imports []string
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	relPath, _ := filepath.Rel(projectRoot, filePath)

	switch ext {
	case ".go":
		imports = extractGoImports(f, relPath, projectRoot)
	case ".ts", ".tsx", ".js", ".jsx":
		imports = extractTSImports(f, relPath, projectRoot)
	case ".py":
		imports = extractPyImports(f, relPath, projectRoot)
	case ".rs":
		imports = extractRustImports(f, relPath, projectRoot)
	}

	return imports
}

var goImportRe = regexp.MustCompile(`"([^"]+)"`)

func extractGoImports(f *os.File, relPath, projectRoot string) []string {
	var imports []string
	scanner := bufio.NewScanner(f)
	inImportBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "import (") {
			inImportBlock = true
			continue
		}
		if inImportBlock {
			if line == ")" {
				inImportBlock = false
				continue
			}
			if m := goImportRe.FindStringSubmatch(line); len(m) > 1 {
				if resolved := resolveImport(m[1], relPath, projectRoot); resolved != "" {
					imports = append(imports, resolved)
				}
			}
		} else if strings.HasPrefix(line, "import ") {
			if m := goImportRe.FindStringSubmatch(line); len(m) > 1 {
				if resolved := resolveImport(m[1], relPath, projectRoot); resolved != "" {
					imports = append(imports, resolved)
				}
			}
		}
	}
	return imports
}

var tsImportRe = regexp.MustCompile(`from\s+['"]([^'"]+)['"]`)

func extractTSImports(f *os.File, relPath, projectRoot string) []string {
	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "import ") && !strings.Contains(line, "require(") {
			continue
		}
		if m := tsImportRe.FindStringSubmatch(line); len(m) > 1 {
			if resolved := resolveImport(m[1], relPath, projectRoot); resolved != "" {
				imports = append(imports, resolved)
			}
		}
	}
	return imports
}

var pyImportRe = regexp.MustCompile(`(?:from\s+(\S+)\s+import|import\s+(\S+))`)

func extractPyImports(f *os.File, relPath, projectRoot string) []string {
	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "from ") && !strings.HasPrefix(line, "import ") {
			continue
		}
		matches := pyImportRe.FindStringSubmatch(line)
		if len(matches) > 2 {
			mod := matches[1]
			if mod == "" {
				mod = matches[2]
			}
			if resolved := resolveImport(strings.ReplaceAll(mod, ".", "/"), relPath, projectRoot); resolved != "" {
				imports = append(imports, resolved)
			}
		}
	}
	return imports
}

var rustImportRe = regexp.MustCompile(`use\s+([^;]+)`)

func extractRustImports(f *os.File, relPath, projectRoot string) []string {
	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "use ") {
			continue
		}
		m := rustImportRe.FindStringSubmatch(line)
		if len(m) > 1 {
			mod := strings.TrimSuffix(m[1], "::*")
			mod = strings.ReplaceAll(mod, "::", "/")
			if resolved := resolveImport(mod, relPath, projectRoot); resolved != "" {
				imports = append(imports, resolved)
			}
		}
	}
	return imports
}

func resolveImport(importPath, fromFile, projectRoot string) string {
	importPath = strings.TrimSuffix(importPath, `"`)
	importPath = strings.TrimSuffix(importPath, `'`)

	// Skip external libs
	if importPath == "" || strings.Contains(importPath, "://") {
		return ""
	}

	// Absolute within project: "ozyassist/backend/internal/agent"
	// Convert to relative: "internal/agent/..."
	if !strings.HasPrefix(importPath, ".") && !strings.Contains(importPath, "://") {
		// Try to resolve as Go module path:
		// Strip known module prefix and check if the directory exists
		parts := strings.Split(importPath, "/")
		// Look for the segment after the module root in projectRoot
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := filepath.Join(projectRoot, filepath.Join(parts[i:]...))
			for _, ext := range []string{".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs"} {
				check := candidate + ext
				if _, err := os.Stat(check); err == nil {
					rel, _ := filepath.Rel(projectRoot, check)
					return filepath.ToSlash(rel)
				}
			}
			// Also check if it's a directory (Go package)
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				rel, _ := filepath.Rel(projectRoot, candidate)
				return filepath.ToSlash(rel)
			}
		}
		return ""
	}

	// Relative import: "./auth", "../models"
	if strings.HasPrefix(importPath, ".") {
		fromDir := filepath.Dir(fromFile)
		resolved := filepath.Clean(filepath.Join(fromDir, importPath))
		// Check if file exists with any code extension
		for ext := range codeExtensions {
			check := filepath.Join(projectRoot, resolved+ext)
			if _, err := os.Stat(check); err == nil {
				return filepath.ToSlash(resolved + ext)
			}
		}
		// Check if it's a directory with index file
		for ext := range codeExtensions {
			check := filepath.Join(projectRoot, resolved, "index"+ext)
			if _, err := os.Stat(check); err == nil {
				return filepath.ToSlash(resolved + "/index" + ext)
			}
		}
		return resolved
	}

	return ""
}
