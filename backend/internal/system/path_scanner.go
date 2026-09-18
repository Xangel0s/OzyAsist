package system

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

type PathCategory string

const (
	CategoryProject   PathCategory = "project"
	CategoryWorkspace PathCategory = "workspace"
	CategorySubRepo   PathCategory = "subrepo"
	CategoryFolder    PathCategory = "folder"
)

type IndexedPath struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	FullPath  string       `json:"full_path"`
	Drive     string       `json:"drive"`
	Category  PathCategory `json:"category"`
	Markers   []string     `json:"markers"`
	Parent    string       `json:"parent,omitempty"`
	UpdatedAt time.Time    `json:"updated_at"`
}

var ignoredDirectories = map[string]bool{
	"node_modules":              true,
	".git":                      true,
	".svn":                      true,
	".hg":                       true,
	"appdata":                   true,
	"local settings":            true,
	"vendor":                    true,
	"dist":                      true,
	"build":                     true,
	"target":                    true,
	"bin":                       true,
	"obj":                       true,
	"windows":                   true,
	"program files":             true,
	"program files (x86)":       true,
	"programdata":               true,
	"$recycle.bin":              true,
	"system volume information": true,
	".cache":                    true,
	".vscode":                   true,
	".idea":                     true,
	"tmp":                       true,
	"temp":                      true,
}

// DiscoverHostRoots detecta dinámicamente las carpetas de anclaje de cualquier PC.
func DiscoverHostRoots() []string {
	var roots []string
	seen := make(map[string]bool)

	addRoot := func(p string) {
		p = filepath.Clean(p)
		if p == "" || seen[strings.ToLower(p)] {
			return
		}
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			seen[strings.ToLower(p)] = true
			roots = append(roots, p)
		}
	}

	// 1. Perfil del usuario actual
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		if usr, err := user.Current(); err == nil && usr.HomeDir != "" {
			userProfile = usr.HomeDir
		} else {
			userProfile = filepath.Join("C:\\Users", os.Getenv("USERNAME"))
		}
	}

	// Carpetas habituales del usuario
	addRoot(filepath.Join(userProfile, "Documents"))
	addRoot(filepath.Join(userProfile, "Desktop"))
	addRoot(filepath.Join(userProfile, "Projects"))
	addRoot(filepath.Join(userProfile, "Development"))
	addRoot(filepath.Join(userProfile, "workspace"))
	addRoot(filepath.Join(userProfile, "repos"))
	addRoot(filepath.Join(userProfile, "code"))
	addRoot(filepath.Join(userProfile, "dev"))

	// 2. Escanear particiones o discos disponibles en Windows (C:, D:, E:, F:, G:)
	drives := []string{"C:\\", "D:\\", "E:\\", "F:\\", "G:\\"}
	for _, d := range drives {
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			// Añadir carpetas de código típicas en otras unidades
			candidates := []string{
				filepath.Join(d, "Projects"),
				filepath.Join(d, "Development"),
				filepath.Join(d, "workspace"),
				filepath.Join(d, "repos"),
				filepath.Join(d, "code"),
				filepath.Join(d, "dev"),
				filepath.Join(d, "Documents"),
			}
			for _, cand := range candidates {
				addRoot(cand)
			}
		}
	}

	return roots
}

// ScanHostWorkspaces realiza un escaneo superficial y seguro (BFS depth 3) descubriendo proyectos y arquitectura.
func ScanHostWorkspaces(ctx context.Context, maxDepth int) ([]IndexedPath, error) {
	if maxDepth <= 0 {
		maxDepth = 3
	}

	roots := DiscoverHostRoots()
	var results []IndexedPath
	seenPaths := make(map[string]bool)

	for _, root := range roots {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		walkRoot(root, 0, maxDepth, "", &results, seenPaths)
	}

	return results, nil
}

func walkRoot(currentDir string, depth, maxDepth int, parentProject string, results *[]IndexedPath, seen map[string]bool) {
	if depth > maxDepth {
		return
	}

	cleanCurrent := filepath.Clean(currentDir)
	if seen[strings.ToLower(cleanCurrent)] {
		return
	}
	seen[strings.ToLower(cleanCurrent)] = true

	entries, err := os.ReadDir(cleanCurrent)
	if err != nil {
		return
	}

	var markers []string
	hasGit := false

	// Analizar archivos de firma en esta carpeta
	for _, e := range entries {
		nameLower := strings.ToLower(e.Name())
		if nameLower == ".git" {
			hasGit = true
			markers = append(markers, "git")
		} else if nameLower == "go.mod" {
			markers = append(markers, "go")
		} else if nameLower == "package.json" {
			markers = append(markers, "node")
		} else if nameLower == "requirements.txt" || nameLower == "pyproject.toml" || nameLower == "pipfile" {
			markers = append(markers, "python")
		} else if nameLower == "cargo.toml" {
			markers = append(markers, "rust")
		} else if nameLower == "pom.xml" || nameLower == "build.gradle" {
			markers = append(markers, "java")
		} else if strings.HasSuffix(nameLower, ".sln") || strings.HasSuffix(nameLower, ".csproj") {
			markers = append(markers, "dotnet")
		} else if nameLower == "composer.json" {
			markers = append(markers, "php")
		}
	}

	currentIsProject := len(markers) > 0 || hasGit
	category := CategoryFolder
	if currentIsProject {
		if parentProject != "" {
			category = CategorySubRepo
		} else {
			category = CategoryProject
		}
	}

	// Si es un proyecto o carpeta de primer nivel, registrarlo
	if currentIsProject || depth == 0 {
		h := sha256.Sum256([]byte(cleanCurrent))
		id := hex.EncodeToString(h[:8])
		drive := filepath.VolumeName(cleanCurrent)
		if drive == "" {
			drive = "C:"
		}

		name := filepath.Base(cleanCurrent)
		*results = append(*results, IndexedPath{
			ID:        id,
			Name:      name,
			FullPath:  cleanCurrent,
			Drive:     drive,
			Category:  category,
			Markers:   markers,
			Parent:    parentProject,
			UpdatedAt: time.Now(),
		})
	}

	nextParent := parentProject
	if currentIsProject && parentProject == "" {
		nextParent = filepath.Base(cleanCurrent)
	}

	// Recursión en subdirectorios válidos
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dirNameLower := strings.ToLower(e.Name())
		if ignoredDirectories[dirNameLower] || strings.HasPrefix(dirNameLower, ".") {
			continue
		}

		subPath := filepath.Join(cleanCurrent, e.Name())
		walkRoot(subPath, depth+1, maxDepth, nextParent, results, seen)
	}
}
