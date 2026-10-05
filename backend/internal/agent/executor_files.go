package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

func execReadFile(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	path, err := resolveSandboxPath(sandbox, params.Path)
	if err != nil {
		return err.Error(), false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error leyendo %s: %v", params.Path, err), false
	}
	return string(data), true
}

func execWriteFile(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	path, err := resolveSandboxPath(sandbox, params.Path)
	if err != nil {
		return err.Error(), false
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Sprintf("error creando directorio: %v", err), false
	}
	var backupMsg string
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		if bPath, bErr := AutoBackupFile(path); bErr == nil && bPath != "" {
			backupMsg = fmt.Sprintf(" (Backup automático: %s)", filepath.Base(bPath))
		}
	}
	if err := os.WriteFile(path, []byte(params.Content), 0644); err != nil {
		return fmt.Sprintf("error escribiendo %s: %v", params.Path, err), false
	}
	return fmt.Sprintf("✓ Archivo escrito: %s (%d bytes)%s", params.Path, len(params.Content), backupMsg), true
}

func execRunCommand(ctx context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Command string `json:"command"`
		Cwd     string `json:"cwd"`
		Dir     string `json:"dir"`
		Path    string `json:"path"`
		Project string `json:"project"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	if params.Cwd == "" {
		if params.Dir != "" {
			params.Cwd = params.Dir
		} else if params.Path != "" {
			params.Cwd = params.Path
		} else if params.Project != "" {
			params.Cwd = params.Project
		} else if sandbox != nil && sandbox.ProjectRoot != "" && sandbox.ProjectRoot != "." {
			params.Cwd = sandbox.ProjectRoot
		}
	}
	return runOSCommandInternal(ctx, params.Command, params.Cwd)
}

func execListFiles(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Pattern    string `json:"pattern"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	root := "."
	if sandbox != nil {
		root = sandbox.ProjectRoot
	}

	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, path)
		matched, _ := filepath.Match(strings.ReplaceAll(params.Pattern, "**", "*"), rel)
		if !matched {
			// También intentar match solo por extensión
			if strings.HasPrefix(params.Pattern, "**/*.") {
				ext := strings.TrimPrefix(params.Pattern, "**/*")
				matched = strings.HasSuffix(path, ext)
			}
		}
		if matched && !d.IsDir() {
			matches = append(matches, system.CleanCanonicalPath(rel))
		}
		if len(matches) >= maxResults {
			return fmt.Errorf("max")
		}
		return nil
	})
	_ = err // ignorar error de max

	if len(matches) == 0 {
		return fmt.Sprintf("No se encontraron archivos con el patrón: %s", params.Pattern), true
	}
	return strings.Join(matches, "\n") + fmt.Sprintf("\n\n[%d archivos]", len(matches)), true
}

func execSearchText(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Query         string `json:"query"`
		Include       string `json:"include"`
		CaseSensitive bool   `json:"case_sensitive"`
		MaxResults    int    `json:"max_results"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 30
	}

	root := "."
	if sandbox != nil {
		root = sandbox.ProjectRoot
	}

	flags := 0
	_ = flags
	var re *regexp.Regexp
	var err error
	if !params.CaseSensitive {
		re, err = regexp.Compile("(?i)" + params.Query)
	} else {
		re, err = regexp.Compile(params.Query)
	}
	if err != nil {
		return fmt.Sprintf("regex inválido '%s': %v", params.Query, err), false
	}

	type Match struct {
		File string
		Line int
		Text string
	}
	var matches []Match

	filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			if d != nil && d.IsDir() && (d.Name() == "node_modules" || d.Name() == ".git" || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if params.Include != "" {
			matched, _ := filepath.Match(params.Include, filepath.Base(path))
			if !matched {
				return nil
			}
		}
		// Limitar a archivos de texto razonables
		if info, err := d.Info(); err == nil && info.Size() > 1<<20 { // >1MB skip
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)

		scanner := bufio.NewScanner(bytes.NewReader(data))
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				matches = append(matches, Match{File: rel, Line: lineNum, Text: strings.TrimSpace(line)})
				if len(matches) >= maxResults {
					return fmt.Errorf("max")
				}
			}
		}
		return nil
	})

	if len(matches) == 0 {
		return fmt.Sprintf("No se encontraron coincidencias para: %s", params.Query), true
	}

	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(fmt.Sprintf("%s:%d: %s\n", system.CleanCanonicalPath(m.File), m.Line, m.Text))
	}
	sb.WriteString(fmt.Sprintf("\n[%d coincidencias]", len(matches)))
	return sb.String(), true
}

func execApplyDiff(_ context.Context, tc providers.ToolCall, sandbox *Sandbox) (string, bool) {
	var params struct {
		Path string `json:"path"`
		Diff string `json:"diff"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	path, err := resolveSandboxPath(sandbox, params.Path)
	if err != nil {
		return err.Error(), false
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("error leyendo %s: %v", params.Path, err), false
	}
	patched, err := applyUnifiedDiff(string(original), params.Diff)
	if err != nil {
		return fmt.Sprintf("error aplicando diff: %v", err), false
	}
	if err := os.WriteFile(path, []byte(patched), 0644); err != nil {
		return fmt.Sprintf("error escribiendo %s: %v", params.Path, err), false
	}
	return fmt.Sprintf("✓ Diff aplicado a %s", params.Path), true
}

// resolveSandboxPath resuelve una ruta relativa al sandbox y previene path traversal.
// Si relPath ya es absoluta, la limpia y la retorna directamente.
func resolveSandboxPath(sandbox *Sandbox, relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return filepath.Clean(relPath), nil
	}
	if sandbox == nil {
		return relPath, nil
	}
	clean := filepath.Clean(relPath)
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("ruta inválida (path traversal): %s", relPath)
	}
	return filepath.Join(sandbox.ProjectRoot, clean), nil
}

// applyUnifiedDiff aplica un diff unificado simple línea a línea.
// Para diffs complejos con múltiples hunks.
func applyUnifiedDiff(original, diff string) (string, error) {
	origLines := strings.Split(original, "\n")
	diffLines := strings.Split(diff, "\n")

	result := make([]string, 0, len(origLines))
	origIdx := 0

	i := 0
	for i < len(diffLines) {
		line := diffLines[i]

		// Saltar headers del diff (--- +++)
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			i++
			continue
		}

		// Hunk header: @@ -a,b +c,d @@
		if strings.HasPrefix(line, "@@") {
			var startOrig, countOrig int
			fmt.Sscanf(line, "@@ -%d,%d", &startOrig, &countOrig)
			startOrig-- // 0-indexed

			// Copiar líneas originales hasta el hunk
			for origIdx < startOrig && origIdx < len(origLines) {
				result = append(result, origLines[origIdx])
				origIdx++
			}
			i++
			continue
		}

		if strings.HasPrefix(line, "+") {
			result = append(result, strings.TrimPrefix(line, "+"))
			i++
		} else if strings.HasPrefix(line, "-") {
			origIdx++ // saltar línea eliminada
			i++
		} else if strings.HasPrefix(line, " ") {
			// Contexto — usar línea original
			if origIdx < len(origLines) {
				result = append(result, origLines[origIdx])
				origIdx++
			}
			i++
		} else {
			i++
		}
	}

	// Copiar el resto de líneas originales
	for origIdx < len(origLines) {
		result = append(result, origLines[origIdx])
		origIdx++
	}

	return strings.Join(result, "\n"), nil
}


func execOSExplore(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path  string `json:"path"`
		Depth int    `json:"depth"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	if params.Depth <= 0 {
		params.Depth = 1
	}
	nav := system.NewWindowsNavigator()
	node, err := nav.ExplorePath(ctx, params.Path, params.Depth)
	if err != nil {
		return fmt.Sprintf("Error explorando ruta: %v", err), false
	}
	var sb strings.Builder
	cleanRoot := system.CleanCanonicalPath(node.Path)
	sb.WriteString(fmt.Sprintf("📂 %s (%d elementos):\n", cleanRoot, len(node.Children)))
	for _, c := range node.Children {
		cleanChildPath := system.CleanCanonicalPath(c.Path)
		if c.IsDir {
			sb.WriteString(fmt.Sprintf("• [carpeta] %s/\n", cleanChildPath))
		} else {
			sb.WriteString(fmt.Sprintf("• [archivo] %s (%s | %s)\n", cleanChildPath, formatFileSize(c.Size), c.ModifiedAt.Format("2006-01-02")))
		}
	}
	return strings.TrimSpace(sb.String()), true
}

func execOSFindFiles(ctx context.Context, tc providers.ToolCall) (string, bool) {
	searchStart := time.Now()
	var params struct {
		Root       string `json:"root"`
		Pattern    string `json:"pattern"`
		MaxResults int    `json:"max_results"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	nav := system.NewWindowsNavigator()

	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}

	root := strings.TrimSpace(params.Root)
	if root == "" || root == "." {
		root = filepath.Join(os.Getenv("USERPROFILE"), "Documents")
	}

	rawPattern := strings.TrimSpace(params.Pattern)
	searchPattern := rawPattern
	if searchPattern == "" {
		searchPattern = "*.pdf"
	}

	matches, err := nav.FindFiles(ctx, root, searchPattern, maxResults)

	// Fallback 1: Si no hubo coincidencias y el patrón contiene palabras compuestas o ruido,
	// extraer palabras clave y reintentar con búsqueda difusa
	if len(matches) == 0 {
		cleanedKw := extractFileSearchPattern(searchPattern)
		if cleanedKw != "" {
			fuzzyPattern := "*" + strings.ReplaceAll(cleanedKw, " ", "*") + "*"
			if m2, err2 := nav.FindFiles(ctx, root, fuzzyPattern, maxResults); err2 == nil && len(m2) > 0 {
				matches = m2
				searchPattern = fuzzyPattern
			}
		}
	}

	// Fallback 2: Si aún no hay coincidencias y buscamos en Documents, buscar también en Desktop
	if len(matches) == 0 && strings.Contains(strings.ToLower(root), "documents") {
		desktopPath := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
		if mDesktop, errD := nav.FindFiles(ctx, desktopPath, searchPattern, maxResults); errD == nil && len(mDesktop) > 0 {
			matches = mDesktop
			root = desktopPath
		}
	}

	// Fallback 3: Si aún no hay coincidencias y el patrón buscaba un tema específico,
	// listar todos los PDFs recientes y filtrar por coincidencia de subcadena en el nombre
	if len(matches) == 0 {
		cleanedKw := extractFileSearchPattern(rawPattern)
		if cleanedKw != "" {
			kwLower := strings.ToLower(cleanedKw)
			allPDFs, _ := nav.FindFiles(ctx, filepath.Join(os.Getenv("USERPROFILE"), "Documents"), "*.pdf", 50)
			for _, p := range allPDFs {
				pName := strings.ToLower(p.Name)
				if strings.Contains(pName, kwLower) || strings.Contains(pName, strings.ReplaceAll(kwLower, " ", "_")) {
					matches = append(matches, p)
					if len(matches) >= maxResults {
						break
					}
				}
			}
		}
	}

	// Fallback 4: Si se busca un ejecutable o app y no se encontró en Documents/Desktop, consultar AppResolver
	if len(matches) == 0 {
		cleanAppQuery := strings.Trim(rawPattern, "*.")
		if appInfo, err := system.ResolveAppExecutable(cleanAppQuery); err == nil && appInfo != nil && appInfo.Command != "" && !appInfo.IsAUMID {
			if fi, errStat := os.Stat(appInfo.Command); errStat == nil && !fi.IsDir() {
				matches = append(matches, system.PathNode{
					Name:       filepath.Base(appInfo.Command),
					Path:       appInfo.Command,
					IsDir:      false,
					Size:       fi.Size(),
					ModifiedAt: fi.ModTime(),
					Extension:  filepath.Ext(appInfo.Command),
				})
				root = filepath.Dir(appInfo.Command)
			}
		}
	}

	elapsed := time.Since(searchStart).Milliseconds()
	if err != nil && len(matches) == 0 {
		return fmt.Sprintf("Error buscando archivos (%d ms): %v", elapsed, err), false
	}
	cleanRoot := system.CleanCanonicalPath(root)
	if len(matches) == 0 {
		return fmt.Sprintf("No se encontraron archivos en %s", cleanRoot), true
	}

	// Ordenar coincidencias por fecha de modificación descendente (los más recientes primero)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ModifiedAt.After(matches[j].ModifiedAt)
	})

	if len(matches) > 0 {
		primaryFile := matches[0].Path
		if strings.HasSuffix(strings.ToLower(primaryFile), ".lnk") {
			if deref, errD := system.ResolveLnkTarget(primaryFile); errD == nil && deref != "" {
				primaryFile = deref
			}
		}
		SetFocusedFile(primaryFile, rawPattern)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 Archivos encontrados (%d):\n", len(matches)))
	for i, m := range matches {
		cleanP := system.CleanCanonicalPath(m.Path)
		resolvedTarget := ""
		if strings.HasSuffix(strings.ToLower(cleanP), ".lnk") {
			if tgt, errT := system.ResolveLnkTarget(m.Path); errT == nil && tgt != "" {
				resolvedTarget = fmt.Sprintf(" -> %s", tgt)
			}
		}
		sizeStr := formatFileSize(m.Size)
		sb.WriteString(fmt.Sprintf("%d. %s%s (%s)\n", i+1, cleanP, resolvedTarget, sizeStr))
	}
	return strings.TrimSpace(sb.String()), true
}


func execOSCompressZip(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Src     string   `json:"src"`
		Srcs    []string `json:"src_paths"`
		DestZip string   `json:"dest_zip"`
		Path    string   `json:"path"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_compress_zip: %v", err), false
	}

	var sources []string
	if len(params.Srcs) > 0 {
		sources = params.Srcs
	} else if params.Src != "" {
		sources = []string{params.Src}
	} else if params.Path != "" {
		sources = []string{params.Path}
	} else {
		return "Se requiere especificar al menos un archivo o carpeta en 'src_paths' o 'src'", false
	}

	dest := params.DestZip
	if dest == "" {
		dest = sources[0] + ".zip"
	}

	err := system.CompressZip(sources, dest)
	if err != nil {
		return fmt.Sprintf("Error comprimiendo archivo ZIP: %v", err), false
	}

	cleanDest := system.CleanCanonicalPath(dest)
	var sizeInfo string
	if fi, err := os.Stat(system.ResolveUserPath(dest)); err == nil {
		sizeInfo = fmt.Sprintf(" (%s)", formatFileSize(fi.Size()))
	}
	return fmt.Sprintf("✅ ZIP Creado: %s%s | Elementos: %d", cleanDest, sizeInfo, len(sources)), true
}

func execOSExtractZip(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		ZipPath string `json:"zip_path"`
		Path    string `json:"path"`
		DestDir string `json:"dest_dir"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_extract_zip: %v", err), false
	}

	zipPath := params.ZipPath
	if zipPath == "" {
		zipPath = params.Path
	}
	if zipPath == "" {
		return "Se requiere especificar la ruta del archivo zip en 'zip_path'", false
	}

	files, err := system.ExtractZip(zipPath, params.DestDir)
	if err != nil {
		return fmt.Sprintf("Error descomprimiendo archivo ZIP: %v", err), false
	}

	cleanZip := system.CleanCanonicalPath(zipPath)
	cleanDest := system.CleanCanonicalPath(params.DestDir)
	return fmt.Sprintf("✅ ZIP Extraído: %s | Archivos: %d | Destino: %s", cleanZip, len(files), cleanDest), true
}

func execOSSearchContent(_ context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Root       string   `json:"root"`
		Path       string   `json:"path"`
		Query      string   `json:"query"`
		IsRegex    bool     `json:"is_regex"`
		Extensions []string `json:"extensions"`
		MaxResults int      `json:"max_results"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_search_content: %v", err), false
	}

	root := params.Root
	if root == "" {
		root = params.Path
	}

	matches, err := system.SearchContent(system.SearchContentParams{
		RootDir:    root,
		Query:      params.Query,
		IsRegex:    params.IsRegex,
		Extensions: params.Extensions,
		MaxResults: params.MaxResults,
	})
	if err != nil {
		return fmt.Sprintf("Error buscando contenido: %v", err), false
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No se encontraron coincidencias para '%s' en %s", params.Query, root), true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔍 Coincidencias encontradas (%d):\n", len(matches)))
	for _, m := range matches {
		cleanP := system.CleanCanonicalPath(m.Path)
		sb.WriteString(fmt.Sprintf("• %s:%d: %s\n", cleanP, m.LineNumber, strings.TrimSpace(m.LineContent)))
	}
	return strings.TrimSpace(sb.String()), true
}


func execOSCreateDir(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	ext := strings.ToLower(filepath.Ext(params.Path))
	if ext == ".xlsx" || ext == ".xls" {
		return "Error: 'os_create_dir' es exclusivamente para carpetas. Para crear un archivo de Excel debes usar la herramienta 'os_create_excel' especificando 'path', 'headers' y 'rows'.", false
	}
	if ext == ".txt" || ext == ".md" || ext == ".csv" || ext == ".json" {
		return "Error: 'os_create_dir' es para carpetas. Para crear archivos de texto usa 'write_file'.", false
	}

	targetPath := system.ResolveUserPath(params.Path)
	nav := system.NewWindowsNavigator()
	if err := nav.CreateDirectory(ctx, targetPath); err != nil {
		return fmt.Sprintf("Error creando carpeta %s: %v", targetPath, err), false
	}
	return fmt.Sprintf("Carpeta creada exitosamente: %s", targetPath), true
}

func execOSMoveItem(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	src := system.ResolveUserPath(params.Src)
	dst := system.ResolveUserPath(params.Dst)
	nav := system.NewWindowsNavigator()
	if err := nav.MoveItem(ctx, src, dst); err != nil {
		return fmt.Sprintf("Error moviendo %s a %s: %v", src, dst, err), false
	}
	return fmt.Sprintf("Elemento movido exitosamente de %s a %s", src, dst), true
}

func execOSCopyItem(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	src := system.ResolveUserPath(params.Src)
	dst := system.ResolveUserPath(params.Dst)
	nav := system.NewWindowsNavigator()
	if err := nav.CopyItem(ctx, src, dst); err != nil {
		return fmt.Sprintf("Error copiando %s a %s: %v", src, dst, err), false
	}
	return fmt.Sprintf("Elemento copiado exitosamente a %s", dst), true
}

func execOSDeleteItem(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path      string `json:"path"`
		Permanent bool   `json:"permanent"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	targetPath := system.ResolveUserPath(params.Path)
	nav := system.NewWindowsNavigator()
	useRecycle := !params.Permanent
	if err := nav.DeleteItem(ctx, targetPath, useRecycle); err != nil {
		return fmt.Sprintf("Error eliminando %s: %v", targetPath, err), false
	}
	targetLoc := "la Papelera de reciclaje"
	if params.Permanent {
		targetLoc = "forma permanente"
	}
	return fmt.Sprintf("Elemento %s eliminado correctamente (%s).", targetPath, targetLoc), true
}

func execOSOrganizeFolder(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	if strings.TrimSpace(params.Path) == "" {
		params.Path = "mis descargas"
	}
	nav := system.NewWindowsNavigator()
	res, err := nav.OrganizeFolder(ctx, params.Path, "extension")
	if err != nil {
		return fmt.Sprintf("Error organizando carpeta %s: %v", params.Path, err), false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== CARPETA ORGANIZADA CON ÉXITO (%s) ===\n", res.SourcePath))
	sb.WriteString(fmt.Sprintf("- Total archivos procesados: %d\n", res.TotalFiles))
	sb.WriteString(fmt.Sprintf("- Total archivos clasificados y movidos: %d\n", res.MovedFiles))
	sb.WriteString("- Desglose por categorías:\n")
	for cat, count := range res.CategorizedMap {
		sb.WriteString(fmt.Sprintf("  • %s: %d archivos\n", cat, count))
	}
	if len(res.Errors) > 0 {
		sb.WriteString("\nAdvertencias:\n")
		for _, e := range res.Errors {
			sb.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}
	return sb.String(), true
}

