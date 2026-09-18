package system

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestPathRegistry_LookupExactAndToken(t *testing.T) {
	reg := &PathRegistry{
		paths:    make(map[string]IndexedPath),
		nameMap:  make(map[string]string),
		tokenMap: make(map[string][]string),
	}

	p1 := IndexedPath{
		ID:        "p1",
		Name:      "crmgeofal",
		FullPath:  `C:\Users\Dev\Documents\crmgeofal`,
		Drive:     "C:",
		Category:  CategoryProject,
		Markers:   []string{"git", "go"},
		UpdatedAt: time.Now(),
	}

	p2 := IndexedPath{
		ID:        "p2",
		Name:      "api-geofal-crm",
		FullPath:  `C:\Users\Dev\Documents\crmgeofal\api-geofal-crm`,
		Drive:     "C:",
		Category:  CategorySubRepo,
		Markers:   []string{"git", "go"},
		Parent:    "crmgeofal",
		UpdatedAt: time.Now(),
	}

	reg.Register(p1, false)
	reg.Register(p2, false)

	// 1. Búsqueda exacta por nombre
	found, ok := reg.Lookup("crmgeofal")
	if !ok || found.FullPath != p1.FullPath {
		t.Fatalf("expected to find crmgeofal, got: %+v, ok: %v", found, ok)
	}

	// 2. Búsqueda por tokens combinados (intersección)
	foundSub, okSub := reg.Lookup("api geofal")
	if !okSub || foundSub.FullPath != p2.FullPath {
		t.Fatalf("expected to find api-geofal-crm via tokens 'api geofal', got: %+v, ok: %v", foundSub, okSub)
	}

	// 3. Consulta no existente
	_, okNone := reg.Lookup("inexistente_proyecto_xyz")
	if okNone {
		t.Fatalf("expected not to find nonexistent project")
	}
}

func TestPathRegistry_GetSystemArchitectureSummary(t *testing.T) {
	reg := &PathRegistry{
		paths:    make(map[string]IndexedPath),
		nameMap:  make(map[string]string),
		tokenMap: make(map[string][]string),
	}

	reg.Register(IndexedPath{
		Name:     "crmgeofal",
		FullPath: `C:\Work\crmgeofal`,
		Category: CategoryProject,
		Markers:  []string{"git", "go"},
	}, false)

	reg.Register(IndexedPath{
		Name:     "api-geofal",
		FullPath: `C:\Work\crmgeofal\api-geofal`,
		Category: CategorySubRepo,
		Parent:   "crmgeofal",
		Markers:  []string{"git"},
	}, false)

	summary := reg.GetSystemArchitectureSummary()
	if !strings.Contains(summary, "ARQUITECTURA DE PROYECTOS Y RUTAS DETECTADAS") {
		t.Errorf("expected summary header, got: %s", summary)
	}
	if !strings.Contains(summary, "crmgeofal") || !strings.Contains(summary, "api-geofal") {
		t.Errorf("expected project names in summary, got: %s", summary)
	}
}

func TestPathScanner_DetectsMarkers(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ozy-scanner-test-*")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Crear estructura de proyecto ficticia
	projDir := filepath.Join(tempDir, "sample-project")
	_ = os.MkdirAll(filepath.Join(projDir, ".git"), 0755)
	_ = os.WriteFile(filepath.Join(projDir, "go.mod"), []byte("module sample\n"), 0644)
	_ = os.WriteFile(filepath.Join(projDir, "package.json"), []byte("{}\n"), 0644)

	var results []IndexedPath
	seen := make(map[string]bool)
	walkRoot(tempDir, 0, 2, "", &results, seen)

	found := false
	for _, p := range results {
		if p.Name == "sample-project" {
			found = true
			hasGit, hasGo, hasNode := false, false, false
			for _, m := range p.Markers {
				if m == "git" {
					hasGit = true
				}
				if m == "go" {
					hasGo = true
				}
				if m == "node" {
					hasNode = true
				}
			}
			if !hasGit || !hasGo || !hasNode {
				t.Errorf("expected git, go and node markers, got: %v", p.Markers)
			}
			if p.Category != CategoryProject {
				t.Errorf("expected CategoryProject, got: %s", p.Category)
			}
		}
	}

	if !found {
		t.Fatalf("sample-project was not detected by scanner")
	}
}

func TestPathRegistry_SQLiteSync(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ozy-db-sync-*")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	dbConn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("error opening test db: %v", err)
	}
	defer dbConn.Close()

	reg := InitPathRegistry(dbConn)
	p := IndexedPath{
		ID:        "test-id",
		Name:      "my-tool",
		FullPath:  `C:\Dev\my-tool`,
		Drive:     "C:",
		Category:  CategoryProject,
		Markers:   []string{"go"},
		UpdatedAt: time.Now(),
	}

	reg.Register(p, true)

	// Esperar que la goroutine de persistencia termine
	time.Sleep(100 * time.Millisecond)

	// Crear nuevo registry y recargar desde SQLite
	newReg := &PathRegistry{
		paths:    make(map[string]IndexedPath),
		nameMap:  make(map[string]string),
		tokenMap: make(map[string][]string),
		db:       dbConn,
	}
	if err := newReg.loadFromSQLite(); err != nil {
		t.Fatalf("error loading from sqlite: %v", err)
	}

	loaded, ok := newReg.Lookup("my-tool")
	if !ok || loaded.FullPath != `C:\Dev\my-tool` {
		t.Fatalf("expected to reload my-tool from sqlite, got: %+v, ok: %v", loaded, ok)
	}
}
