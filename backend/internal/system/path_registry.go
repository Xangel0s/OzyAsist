package system

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type PathRegistry struct {
	mu       sync.RWMutex
	paths    map[string]IndexedPath   // fullPath -> IndexedPath
	nameMap  map[string]string        // lowercase name -> fullPath
	tokenMap map[string][]string      // token -> list of fullPaths
	db       *sql.DB
}

var (
	defaultPathRegistry     *PathRegistry
	defaultPathRegistryOnce sync.Once
)

func DefaultPathRegistry() *PathRegistry {
	defaultPathRegistryOnce.Do(func() {
		defaultPathRegistry = &PathRegistry{
			paths:    make(map[string]IndexedPath),
			nameMap:  make(map[string]string),
			tokenMap: make(map[string][]string),
		}
	})
	return defaultPathRegistry
}

// InitPathRegistry inicializa el registro en RAM y sincroniza con SQLite si está disponible.
func InitPathRegistry(dbConn *sql.DB) *PathRegistry {
	r := DefaultPathRegistry()
	r.db = dbConn
	if dbConn != nil {
		_ = r.ensureTable()
		_ = r.loadFromSQLite()
	}
	return r
}

// InitDefaultPathRegistry es un alias para inicializar el registro global por defecto con SQLite.
func InitDefaultPathRegistry(dbConn *sql.DB) *PathRegistry {
	return InitPathRegistry(dbConn)
}

func (r *PathRegistry) ensureTable() error {
	if r.db == nil {
		return nil
	}
	schema := `
	CREATE TABLE IF NOT EXISTS system_paths (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		full_path TEXT NOT NULL UNIQUE,
		drive TEXT,
		category TEXT,
		markers TEXT,
		parent TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_system_paths_name ON system_paths(name);
	`
	_, err := r.db.Exec(schema)
	return err
}

func (r *PathRegistry) loadFromSQLite() error {
	if r.db == nil {
		return nil
	}
	rows, err := r.db.Query("SELECT id, name, full_path, COALESCE(drive,''), COALESCE(category,''), COALESCE(markers,''), COALESCE(parent,''), updated_at FROM system_paths")
	if err != nil {
		return err
	}
	defer rows.Close()

	var loaded []IndexedPath
	for rows.Next() {
		var p IndexedPath
		var markersStr string
		var catStr string
		if err := rows.Scan(&p.ID, &p.Name, &p.FullPath, &p.Drive, &catStr, &markersStr, &p.Parent, &p.UpdatedAt); err == nil {
			p.Category = PathCategory(catStr)
			if markersStr != "" {
				p.Markers = strings.Split(markersStr, ",")
			}
			loaded = append(loaded, p)
		}
	}

	for _, p := range loaded {
		r.Register(p, false)
	}
	return nil
}

// Register inserta una ruta en el mapa en memoria RAM y opcionalmente en SQLite.
func (r *PathRegistry) Register(p IndexedPath, persist bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	normPath := strings.ToLower(p.FullPath)
	r.paths[normPath] = p
	r.nameMap[strings.ToLower(p.Name)] = normPath

	// Tokenizar para el índice invertido en RAM
	tokens := tokenizePath(p.Name + " " + p.FullPath)
	for _, tok := range tokens {
		list := r.tokenMap[tok]
		found := false
		for _, existing := range list {
			if existing == normPath {
				found = true
				break
			}
		}
		if !found {
			r.tokenMap[tok] = append(list, normPath)
		}
	}

	if persist && r.db != nil {
		go func(item IndexedPath) {
			markersStr := strings.Join(item.Markers, ",")
			query := `
			INSERT INTO system_paths (id, name, full_path, drive, category, markers, parent, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(full_path) DO UPDATE SET
				name = excluded.name,
				drive = excluded.drive,
				category = excluded.category,
				markers = excluded.markers,
				parent = excluded.parent,
				updated_at = CURRENT_TIMESTAMP;`
			_, _ = r.db.Exec(query, item.ID, item.Name, item.FullPath, item.Drive, string(item.Category), markersStr, item.Parent)
		}(p)
	}
}

// Lookup busca en el mapa de tokens en RAM en microsegundos sin acceder a disco.
func (r *PathRegistry) Lookup(query string) (*IndexedPath, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cleanQuery := strings.TrimSpace(strings.ToLower(query))
	if cleanQuery == "" {
		return nil, false
	}

	// 1. Coincidencia exacta por nombre
	if fullPath, ok := r.nameMap[cleanQuery]; ok {
		if p, found := r.paths[fullPath]; found {
			return &p, true
		}
	}

	// 2. Coincidencia exacta por ruta completa
	if p, found := r.paths[cleanQuery]; found {
		return &p, true
	}

	// 3. Coincidencia por intersección de tokens (ej: "api", "geofal")
	queryTokens := tokenizePath(cleanQuery)
	if len(queryTokens) == 0 {
		return nil, false
	}

	scoreMap := make(map[string]int)
	for _, tok := range queryTokens {
		for _, fullPath := range r.tokenMap[tok] {
			scoreMap[fullPath] += 2
		}
		// Búsqueda por prefijo de token
		for indexedTok, fullPaths := range r.tokenMap {
			if len(tok) >= 3 && strings.HasPrefix(indexedTok, tok) && indexedTok != tok {
				for _, fp := range fullPaths {
					scoreMap[fp] += 1
				}
			}
		}
	}

	var bestPath string
	highestScore := 0
	for fp, score := range scoreMap {
		if score > highestScore {
			highestScore = score
			bestPath = fp
		}
	}

	if bestPath != "" && highestScore >= len(queryTokens)*2 {
		p := r.paths[bestPath]
		return &p, true
	}

	return nil, false
}

// ListProjects devuelve todos los proyectos identificados en el host.
func (r *PathRegistry) ListProjects() []IndexedPath {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []IndexedPath
	for _, p := range r.paths {
		if p.Category == CategoryProject || p.Category == CategorySubRepo {
			list = append(list, p)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})

	return list
}

// ListAll devuelve todas las rutas indexadas en memoria RAM.
func (r *PathRegistry) ListAll() []IndexedPath {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []IndexedPath
	for _, p := range r.paths {
		list = append(list, p)
	}

	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})

	return list
}

// GetSystemArchitectureSummary genera el bloque estructurado de proyectos para el System Prompt.
func (r *PathRegistry) GetSystemArchitectureSummary() string {
	projects := r.ListProjects()
	if len(projects) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("ARQUITECTURA DE PROYECTOS Y RUTAS DETECTADAS EN ESTE HOST:\n")

	// Agrupar proyectos principales y sus sub-repositorios
	subReposByParent := make(map[string][]IndexedPath)
	var rootProjects []IndexedPath

	for _, p := range projects {
		if p.Parent != "" && p.Category == CategorySubRepo {
			subReposByParent[p.Parent] = append(subReposByParent[p.Parent], p)
		} else {
			rootProjects = append(rootProjects, p)
		}
	}

	for _, rp := range rootProjects {
		markersDesc := ""
		if len(rp.Markers) > 0 {
			markersDesc = fmt.Sprintf(" [%s]", strings.Join(rp.Markers, ", "))
		}
		sb.WriteString(fmt.Sprintf("- %s: %s%s\n", rp.Name, rp.FullPath, markersDesc))

		if subs, ok := subReposByParent[rp.Name]; ok {
			for _, sp := range subs {
				spMarkers := ""
				if len(sp.Markers) > 0 {
					spMarkers = fmt.Sprintf(" [%s]", strings.Join(sp.Markers, ", "))
				}
				sb.WriteString(fmt.Sprintf("  └── Sub-repo: %s (%s)%s\n", sp.Name, sp.FullPath, spMarkers))
			}
		}
	}

	return sb.String()
}

// RefreshScan ejecuta el escaneo de host en segundo plano y actualiza el registro en RAM y SQLite.
func (r *PathRegistry) RefreshScan(ctx context.Context) (int, error) {
	scanned, err := ScanHostWorkspaces(ctx, 3)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, p := range scanned {
		r.Register(p, true)
		count++
	}
	return count, nil
}

func tokenizePath(raw string) []string {
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return ' '
	}, strings.ToLower(raw))

	fields := strings.Fields(clean)
	var tokens []string
	seen := make(map[string]bool)
	for _, f := range fields {
		if len(f) >= 2 && !seen[f] {
			seen[f] = true
			tokens = append(tokens, f)
		}
	}
	return tokens
}
