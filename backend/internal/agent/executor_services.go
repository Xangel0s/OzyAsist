package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// execOSServiceManager gestiona e inspecciona servicios de Windows de forma nativa
func execOSServiceManager(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // list, status, start, stop, restart
		Name   string `json:"name"`
		Filter string `json:"filter"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "list"
	}

	target := strings.TrimSpace(params.Name)

	switch action {
	case "list":
		psCmd := `Get-Service | Select-Object -Property Name, DisplayName, Status, StartType | ConvertTo-Json -Compress`
		if params.Filter != "" {
			escFilter := strings.ReplaceAll(params.Filter, "'", "''")
			psCmd = fmt.Sprintf(`Get-Service | Where-Object { $_.Name -like '*%s*' -or $_.DisplayName -like '*%s*' } | Select-Object -Property Name, DisplayName, Status, StartType | ConvertTo-Json -Compress`, escFilter, escFilter)
		}

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
		if err != nil {
			return fmt.Sprintf("Error listando servicios de Windows: %v", err), false
		}

		raw := strings.TrimSpace(string(out))
		if raw == "" || raw == "null" {
			return fmt.Sprintf("No se encontraron servicios de Windows que coincidan con el filtro '%s'.", params.Filter), true
		}

		type svcItem struct {
			Name        string      `json:"Name"`
			DisplayName string      `json:"DisplayName"`
			Status      interface{} `json:"Status"`
			StartType   interface{} `json:"StartType"`
		}

		var items []svcItem
		if strings.HasPrefix(raw, "[") {
			_ = json.Unmarshal([]byte(raw), &items)
		} else {
			var single svcItem
			if err := json.Unmarshal([]byte(raw), &single); err == nil {
				items = append(items, single)
			}
		}

		if len(items) == 0 {
			return "No se detectaron servicios.", true
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== SERVICIOS DE WINDOWS (%d detectados) ===\n", len(items)))
		for i, it := range items {
			if i >= 40 {
				sb.WriteString(fmt.Sprintf("... y %d servicios más (refina con el parámetro 'filter').\n", len(items)-40))
				break
			}
			stStr := fmt.Sprintf("%v", it.Status)
			badge := "⚪"
			if stStr == "4" || strings.EqualFold(stStr, "Running") {
				badge = "🟢 EN EJECUCIÓN"
			} else if stStr == "1" || strings.EqualFold(stStr, "Stopped") {
				badge = "🔴 DETENIDO"
			} else {
				badge = "🟡 " + stStr
			}
			sb.WriteString(fmt.Sprintf("- %s [%s]: %s (Inicio: %v)\n", it.Name, badge, it.DisplayName, it.StartType))
		}
		return sb.String(), true

	case "status":
		if target == "" {
			return "Debes indicar el nombre del servicio (parámetro 'name').", false
		}
		escTarget := strings.ReplaceAll(target, "'", "''")
		psCmd := fmt.Sprintf(`Get-Service -Name '%s' -ErrorAction Stop | Select-Object -Property Name, DisplayName, Status, StartType, ServiceType | ConvertTo-Json`, escTarget)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("No se encontró el servicio '%s' o ocurrió un error: %s", target, strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("=== ESTADO DEL SERVICIO: %s ===\n%s", target, strings.TrimSpace(string(out))), true

	case "start":
		if target == "" {
			return "Debes indicar el nombre del servicio a iniciar.", false
		}
		escTarget := strings.ReplaceAll(target, "'", "''")
		psCmd := fmt.Sprintf(`Start-Service -Name '%s' -ErrorAction Stop; (Get-Service -Name '%s').Status`, escTarget, escTarget)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error iniciando el servicio '%s' (puede requerir elevación de administrador): %s", target, strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Servicio '%s' iniciado exitosamente. Estado actual: %s", target, strings.TrimSpace(string(out))), true

	case "stop":
		if target == "" {
			return "Debes indicar el nombre del servicio a detener.", false
		}
		escTarget := strings.ReplaceAll(target, "'", "''")
		psCmd := fmt.Sprintf(`Stop-Service -Name '%s' -Force -ErrorAction Stop; (Get-Service -Name '%s').Status`, escTarget, escTarget)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error deteniendo el servicio '%s' (puede requerir elevación de administrador): %s", target, strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Servicio '%s' detenido exitosamente. Estado actual: %s", target, strings.TrimSpace(string(out))), true

	case "restart":
		if target == "" {
			return "Debes indicar el nombre del servicio a reiniciar.", false
		}
		escTarget := strings.ReplaceAll(target, "'", "''")
		psCmd := fmt.Sprintf(`Restart-Service -Name '%s' -Force -ErrorAction Stop; (Get-Service -Name '%s').Status`, escTarget, escTarget)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error reiniciando el servicio '%s': %s", target, strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Servicio '%s' reiniciado exitosamente. Estado actual: %s", target, strings.TrimSpace(string(out))), true

	default:
		return fmt.Sprintf("Acción de servicio desconocida: '%s'. Usa 'list', 'status', 'start', 'stop' o 'restart'.", action), false
	}
}

// execOSDockerManager gestiona contenedores Docker presentes en el host de Windows
func execOSDockerManager(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action    string `json:"action"` // status, list, logs, start, stop, restart, stats
		Container string `json:"container"`
		Tail      int    `json:"tail"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "status"
	}

	if params.Tail <= 0 {
		params.Tail = 50
	}

	// Verificar si el ejecutable docker existe en el PATH
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return "Docker CLI no está instalado o no se encuentra en el PATH del sistema.", false
	}

	switch action {
	case "status":
		cmd := exec.CommandContext(ctx, dockerPath, "info", "--format", "{{.ServerVersion}}")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "Docker CLI está disponible pero el motor Docker (Docker Desktop / Docker Daemon) no está en ejecución.", false
		}
		version := strings.TrimSpace(string(out))
		return fmt.Sprintf("Docker Daemon activo en el host (Versión del Servidor: %s).", version), true

	case "list":
		cmd := exec.CommandContext(ctx, dockerPath, "ps", "-a", "--format", "table {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error listando contenedores: %s", strings.TrimSpace(string(out))), false
		}
		res := strings.TrimSpace(string(out))
		if res == "" || strings.Count(res, "\n") == 0 {
			return "No hay contenedores Docker en el host.", true
		}
		return fmt.Sprintf("=== CONTENEDORES DOCKER ===\n%s", res), true

	case "logs":
		if params.Container == "" {
			return "Debes indicar el nombre o ID del contenedor para ver sus logs.", false
		}
		cmd := exec.CommandContext(ctx, dockerPath, "logs", "--tail", fmt.Sprintf("%d", params.Tail), params.Container)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error obteniendo logs del contenedor '%s': %s", params.Container, strings.TrimSpace(string(out))), false
		}
		res := strings.TrimSpace(string(out))
		if res == "" {
			return fmt.Sprintf("El contenedor '%s' no tiene logs recientes.", params.Container), true
		}
		return fmt.Sprintf("=== LOGS DE %s (Últimas %d líneas) ===\n%s", params.Container, params.Tail, res), true

	case "start":
		if params.Container == "" {
			return "Debes indicar el nombre o ID del contenedor a iniciar.", false
		}
		cmd := exec.CommandContext(ctx, dockerPath, "start", params.Container)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error iniciando contenedor '%s': %s", params.Container, strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Contenedor '%s' iniciado exitosamente.", params.Container), true

	case "stop":
		if params.Container == "" {
			return "Debes indicar el nombre o ID del contenedor a detener.", false
		}
		cmd := exec.CommandContext(ctx, dockerPath, "stop", params.Container)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error deteniendo contenedor '%s': %s", params.Container, strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Contenedor '%s' detenido exitosamente.", params.Container), true

	case "restart":
		if params.Container == "" {
			return "Debes indicar el nombre o ID del contenedor a reiniciar.", false
		}
		cmd := exec.CommandContext(ctx, dockerPath, "restart", params.Container)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error reiniciando contenedor '%s': %s", params.Container, strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Contenedor '%s' reiniciado exitosamente.", params.Container), true

	case "stats":
		cmd := exec.CommandContext(ctx, dockerPath, "stats", "--no-stream", "--format", "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error obteniendo métricas de Docker: %s", strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("=== CONSUMO DE RECURSOS DOCKER ===\n%s", strings.TrimSpace(string(out))), true

	default:
		return fmt.Sprintf("Acción desconocida de Docker: '%s'. Usa 'status', 'list', 'logs', 'start', 'stop', 'restart' o 'stats'.", action), false
	}
}

// execOSAnalyzeLogs analiza archivos de registro o eventos del visor de Windows
func execOSAnalyzeLogs(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Source   string `json:"source"`   // Ruta de archivo de log o "application", "system", "windows-events"
		Lines    int    `json:"lines"`    // Cantidad de líneas a inspeccionar (default: 60)
		Filter   string `json:"filter"`   // Cadena o regex de búsqueda
		Severity string `json:"severity"` // "ERROR", "WARNING", "ALL"
	}
	_ = json.Unmarshal(tc.Input, &params)

	if params.Lines <= 0 {
		params.Lines = 60
	}
	if params.Lines > 300 {
		params.Lines = 300
	}

	source := strings.TrimSpace(params.Source)
	if source == "" {
		source = "windows-events"
	}

	// Caso 1: Visor de eventos de Windows (Application o System)
	if strings.EqualFold(source, "windows-events") || strings.EqualFold(source, "application") || strings.EqualFold(source, "system") {
		logName := "Application"
		if strings.EqualFold(source, "system") {
			logName = "System"
		}

		levelFilter := "1,2,3" // 1: Critical, 2: Error, 3: Warning
		if strings.EqualFold(params.Severity, "ERROR") {
			levelFilter = "1,2"
		} else if strings.EqualFold(params.Severity, "WARNING") {
			levelFilter = "3"
		}

		psCmd := fmt.Sprintf(`Get-WinEvent -FilterHashtable @{LogName='%s'; Level=%s} -MaxEvents %d -ErrorAction SilentlyContinue | Select-Object -Property TimeCreated, LevelDisplayName, Id, ProviderName, Message | ConvertTo-Json -Compress`, logName, levelFilter, params.Lines)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
		if err != nil || len(strings.TrimSpace(string(out))) == 0 || strings.TrimSpace(string(out)) == "null" {
			return fmt.Sprintf("No se registraron eventos recientes de severidad %s en el registro '%s'.", params.Severity, logName), true
		}

		type winEvent struct {
			TimeCreated      string `json:"TimeCreated"`
			LevelDisplayName string `json:"LevelDisplayName"`
			ID               int    `json:"Id"`
			ProviderName     string `json:"ProviderName"`
			Message          string `json:"Message"`
		}

		var events []winEvent
		raw := strings.TrimSpace(string(out))
		if strings.HasPrefix(raw, "[") {
			_ = json.Unmarshal([]byte(raw), &events)
		} else {
			var single winEvent
			if err := json.Unmarshal([]byte(raw), &single); err == nil {
				events = append(events, single)
			}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== EVENTOS DE WINDOWS: %s (%d encontrados) ===\n", logName, len(events)))
		for i, ev := range events {
			badge := "[ERROR]"
			if strings.EqualFold(ev.LevelDisplayName, "Warning") || strings.EqualFold(ev.LevelDisplayName, "Advertencia") {
				badge = "[ADVERTENCIA]"
			} else if strings.EqualFold(ev.LevelDisplayName, "Critical") {
				badge = "[CRITICO]"
			}
			msgClean := strings.ReplaceAll(ev.Message, "\r\n", " ")
			if len(msgClean) > 200 {
				msgClean = msgClean[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] [%s] %s (ID %d): %s\n", i+1, badge, ev.TimeCreated, ev.ProviderName, ev.ID, msgClean))
		}
		return sb.String(), true
	}

	// Caso 2: Archivo de registro en disco
	resolvedPath := system.ResolveUserPath(source)
	if !filepath.IsAbs(resolvedPath) {
		wd, _ := os.Getwd()
		resolvedPath = filepath.Join(wd, resolvedPath)
	}

	f, err := os.Open(resolvedPath)
	if err != nil {
		return fmt.Sprintf("No se pudo abrir el archivo de log '%s': %v", source, err), false
	}
	defer f.Close()

	// Leer líneas del archivo
	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if len(allLines) == 0 {
		return fmt.Sprintf("El archivo de log '%s' está vacío.", filepath.Base(resolvedPath)), true
	}

	startIdx := 0
	if len(allLines) > params.Lines {
		startIdx = len(allLines) - params.Lines
	}
	selectedLines := allLines[startIdx:]

	var filterRe *regexp.Regexp
	if params.Filter != "" {
		if re, err := regexp.Compile("(?i)" + params.Filter); err == nil {
			filterRe = re
		}
	} else if strings.EqualFold(params.Severity, "ERROR") {
		filterRe = regexp.MustCompile(`(?i)(error|panic|fatal|exception|traceback|fail)`)
	} else if strings.EqualFold(params.Severity, "WARNING") {
		filterRe = regexp.MustCompile(`(?i)(warn|warning|alert)`)
	}

	var matchedLines []string
	for i, l := range selectedLines {
		lineNum := startIdx + i + 1
		if filterRe == nil || filterRe.MatchString(l) {
			matchedLines = append(matchedLines, fmt.Sprintf("L%d: %s", lineNum, l))
		}
	}

	if len(matchedLines) == 0 {
		return fmt.Sprintf("Se analizaron las últimas %d líneas de '%s' y no se encontraron coincidencias para el filtro.", len(selectedLines), filepath.Base(resolvedPath)), true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== ANÁLISIS DE LOG: %s (%d coincidencias de %d líneas revisadas) ===\n", filepath.Base(resolvedPath), len(matchedLines), len(selectedLines)))
	for _, ml := range matchedLines {
		sb.WriteString(ml + "\n")
	}
	return sb.String(), true
}
