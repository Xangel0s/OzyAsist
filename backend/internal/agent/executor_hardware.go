package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

// execOSTileWindows organiza y posiciona ventanas visibles en la pantalla
func execOSTileWindows(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		HWND   uintptr `json:"hwnd"`
		Title  string  `json:"title"`
		Layout string  `json:"layout"` // left, right, maximize, minimize, restore, center, show_desktop
	}
	_ = json.Unmarshal(tc.Input, &params)

	layout := strings.ToLower(strings.TrimSpace(params.Layout))
	if layout == "" {
		layout = "maximize"
	}

	nav := system.NewWindowsNavigator()

	if layout == "show_desktop" || layout == "minimize_all" {
		if err := nav.TileWindow(ctx, 0, layout); err != nil {
			return fmt.Sprintf("Error minimizando ventanas: %v", err), false
		}
		return "Escritorio mostrado: todas las ventanas han sido minimizadas.", true
	}

	targetHWND := params.HWND
	targetTitle := strings.TrimSpace(params.Title)

	if targetHWND == 0 && targetTitle != "" {
		cleanLower := strings.ToLower(targetTitle)
		for _, prefix := range []string{"la ventana ", "ventana ", "el ", "la ", "los ", "las "} {
			if strings.HasPrefix(cleanLower, prefix) {
				cleanLower = strings.TrimSpace(cleanLower[len(prefix):])
			}
		}

		wins, err := nav.GetActiveWindows(ctx)
		if err == nil {
			for _, w := range wins {
				wLower := strings.ToLower(w.Title)
				if strings.Contains(wLower, cleanLower) ||
					(strings.Contains(cleanLower, "administrador") && strings.Contains(wLower, "administrador")) ||
					(strings.Contains(cleanLower, "tarea") && strings.Contains(wLower, "tarea")) ||
					(strings.Contains(cleanLower, "chrome") && strings.Contains(wLower, "chrome")) ||
					(strings.Contains(cleanLower, "code") && strings.Contains(wLower, "code")) {
					targetHWND = w.Handle
					targetTitle = w.Title
					break
				}
			}
		}
	}

	if targetHWND == 0 {
		return fmt.Sprintf("No se encontró ninguna ventana activa que coincida con '%s' para acomodar.", params.Title), false
	}

	if err := nav.TileWindow(ctx, targetHWND, layout); err != nil {
		return fmt.Sprintf("Error acomodando ventana: %v", err), false
	}

	layoutNames := map[string]string{
		"left":         "mitad izquierda",
		"right":        "mitad derecha",
		"maximize":     "maximizada",
		"minimize":     "minimizada",
		"restore":      "tamaño estándar restaurado",
		"center":       "centrada",
		"show_desktop": "escritorio",
	}
	desc := layoutNames[layout]
	if desc == "" {
		desc = layout
	}

	if targetTitle != "" {
		return fmt.Sprintf("Ventana '%s' acomodada en: %s.", targetTitle, desc), true
	}
	return fmt.Sprintf("Ventana (HWND %d) acomodada en: %s.", targetHWND, desc), true
}

// execOSDiskCleaner analiza o limpia archivos temporales y la papelera de reciclaje
func execOSDiskCleaner(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action          string `json:"action"` // analyze, clean
		TargetDir       string `json:"target_dir"`
		EmptyRecycleBin bool   `json:"empty_recycle_bin"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "analyze"
	}

	tempDirs := []string{}
	if strings.TrimSpace(params.TargetDir) != "" {
		tempDirs = append(tempDirs, system.ResolveUserPath(params.TargetDir))
	} else {
		tempDirs = append(tempDirs, os.TempDir())
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			localTemp := filepath.Join(userProfile, "AppData", "Local", "Temp")
			if _, err := os.Stat(localTemp); err == nil && localTemp != os.TempDir() {
				tempDirs = append(tempDirs, localTemp)
			}
		}
	}

	var totalBytes int64
	var totalFiles int
	var deletedBytes int64
	var deletedFiles int

	for _, d := range tempDirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			nameLower := strings.ToLower(e.Name())
			// Proteger directorios de compilación activa y procesos en ejecución
			if strings.HasPrefix(nameLower, "go-build") ||
				strings.HasPrefix(nameLower, "antigravity") ||
				strings.HasPrefix(nameLower, "gemini") ||
				strings.HasPrefix(nameLower, "cortex") ||
				strings.HasPrefix(nameLower, "scoped_dir") ||
				strings.HasPrefix(nameLower, "~") {
				continue
			}

			fullPath := filepath.Join(d, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			size := info.Size()
			totalBytes += size
			totalFiles++

			if action == "clean" {
				// Intentar eliminar, omitiendo errores de archivos bloqueados o en uso
				if err := os.RemoveAll(fullPath); err == nil {
					deletedBytes += size
					deletedFiles++
				}
			}
		}
	}

	if action == "clean" && params.EmptyRecycleBin {
		psCmd := `Clear-RecycleBin -Force -ErrorAction SilentlyContinue`
		_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run()
	}

	toMB := func(b int64) float64 {
		return float64(b) / (1024 * 1024)
	}

	if action == "clean" {
		binMsg := ""
		if params.EmptyRecycleBin {
			binMsg = " Papelera de reciclaje vaciada."
		}
		return fmt.Sprintf("Limpieza completada: %.2f MB liberados (%d de %d archivos temporales eliminados).%s",
			toMB(deletedBytes), deletedFiles, totalFiles, binMsg), true
	}

	return fmt.Sprintf("=== ANÁLISIS DE ESPACIO PURGABLE ===\n- Archivos temporales detectados: %d\n- Espacio recuperable aproximado: %.2f MB en carpetas temporales.\nUsa action: 'clean' para proceder con la purga segura.",
		totalFiles, toMB(totalBytes)), true
}

// execOSAudioDevice lista o conmuta dispositivos de audio en Windows
func execOSAudioDevice(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // list, status, set
		Name   string `json:"name"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "list", "status":
		psCmd := `Get-PnpDevice -Class AudioEndpoint -Status OK -ErrorAction SilentlyContinue | Select-Object -Property FriendlyName, Status, InstanceId | ConvertTo-Json -Compress`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
		if err != nil || len(strings.TrimSpace(string(out))) == 0 || strings.TrimSpace(string(out)) == "null" {
			// Fallback a Win32_SoundDevice
			psCmd2 := `Get-CimInstance Win32_SoundDevice | Select-Object -Property Name, Status | ConvertTo-Json -Compress`
			out2, _ := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd2).Output()
			out = out2
		}

		type audioItem struct {
			FriendlyName string `json:"FriendlyName"`
			Name         string `json:"Name"`
			Status       string `json:"Status"`
		}

		var items []audioItem
		raw := strings.TrimSpace(string(out))
		if strings.HasPrefix(raw, "[") {
			_ = json.Unmarshal([]byte(raw), &items)
		} else if strings.HasPrefix(raw, "{") {
			var single audioItem
			if err := json.Unmarshal([]byte(raw), &single); err == nil {
				items = append(items, single)
			}
		}

		if len(items) == 0 {
			return "No se detectaron dispositivos de audio activos.", true
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== DISPOSITIVOS DE AUDIO DETECTADOS (%d) ===\n", len(items)))
		for i, dev := range items {
			name := dev.FriendlyName
			if name == "" {
				name = dev.Name
			}
			badge := "🔊"
			lower := strings.ToLower(name)
			if strings.Contains(lower, "auricular") || strings.Contains(lower, "headphone") || strings.Contains(lower, "buds") {
				badge = "🎧"
			} else if strings.Contains(lower, "fxsound") {
				badge = "⚡"
			}
			sb.WriteString(fmt.Sprintf("%d. %s %s (Estado: %s)\n", i+1, badge, name, dev.Status))
		}
		return sb.String(), true

	case "set":
		if strings.TrimSpace(params.Name) == "" {
			return "Debes indicar el nombre del dispositivo de audio a activar.", false
		}
		target := strings.TrimSpace(params.Name)
		escTarget := strings.ReplaceAll(target, "'", "''")

		// Intentar con AudioDeviceCmdlets si está disponible, o advertir amablemente
		psCmd := fmt.Sprintf(`
			if (Get-Command Set-AudioDevice -ErrorAction SilentlyContinue) {
				$dev = Get-AudioDevice -List | Where-Object { $_.Name -like '*%s*' } | Select-Object -First 1
				if ($dev) {
					Set-AudioDevice -Index $dev.Index
					"OK: Dispositivo cambiado a $($dev.Name)"
				} else {
					"No se encontró un dispositivo que coincida con '%s'."
				}
			} else {
				$dev = Get-PnpDevice -Class AudioEndpoint -Status OK | Where-Object { $_.FriendlyName -like '*%s*' } | Select-Object -First 1
				if ($dev) {
					"Dispositivo detectado: '$($dev.FriendlyName)'. Para conmutación automática de hardware sin clics, instala el módulo PowerShell 'AudioDeviceCmdlets' ('Install-Module -Name AudioDeviceCmdlets')."
				} else {
					"No se encontró ningún endpoint de audio con el nombre '%s'."
				}
			}
		`, escTarget, escTarget, escTarget, escTarget)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error configurando dispositivo de audio: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'list' o 'set'.", action), false
	}
}

// execOSWifiManager obtiene la telemetría y estado de conexiones inalámbricas Wi-Fi
func execOSWifiManager(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // status, networks
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "status"
	}

	switch action {
	case "status":
		out, err := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando interfaz Wi-Fi: %s", strings.TrimSpace(string(out))), false
		}
		res := strings.TrimSpace(string(out))
		if strings.Contains(res, "El Servicio de directivas de diagnóstico") || strings.Contains(res, "no hay ninguna interfaz") {
			return "No hay adaptador Wi-Fi inalámbrico activo o habilitado en este equipo.", true
		}

		lines := strings.Split(res, "\n")
		var sb strings.Builder
		sb.WriteString("=== ESTADO DE CONEXIÓN WI-FI ===\n")
		for _, l := range lines {
			lTrim := strings.TrimSpace(l)
			if strings.HasPrefix(lTrim, "Nombre") ||
				strings.HasPrefix(lTrim, "Descripción") ||
				strings.HasPrefix(lTrim, "Estado") ||
				strings.HasPrefix(lTrim, "SSID") ||
				strings.HasPrefix(lTrim, "Tipo de radio") ||
				strings.HasPrefix(lTrim, "Autenticación") ||
				strings.HasPrefix(lTrim, "Señal") ||
				strings.HasPrefix(lTrim, "Velocidad") ||
				strings.HasPrefix(lTrim, "Canal") {
				sb.WriteString("- " + lTrim + "\n")
			}
		}
		return sb.String(), true

	case "networks":
		out, err := exec.CommandContext(ctx, "netsh", "wlan", "show", "networks").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error escaneando redes Wi-Fi: %s", strings.TrimSpace(string(out))), false
		}
		lines := strings.Split(string(out), "\n")
		var sb strings.Builder
		sb.WriteString("=== REDES WI-FI DISPONIBLES EN EL ENTORNO ===\n")
		currentSSID := ""
		for _, l := range lines {
			lTrim := strings.TrimSpace(l)
			if strings.HasPrefix(lTrim, "SSID") {
				currentSSID = lTrim
				sb.WriteString("\n📶 " + currentSSID + "\n")
			} else if strings.HasPrefix(lTrim, "Tipo de red") || strings.HasPrefix(lTrim, "Autenticación") || strings.HasPrefix(lTrim, "Cifrado") {
				sb.WriteString("   - " + lTrim + "\n")
			}
		}
		return sb.String(), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'status' o 'networks'.", action), false
	}
}

// execOSScheduleTask gestiona tareas programadas persistentes en Windows (schtasks.exe)
func execOSScheduleTask(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action   string `json:"action"`   // list, create, delete
		Name     string `json:"name"`     // Nombre de la tarea
		Command  string `json:"command"`  // Comando a ejecutar
		Schedule string `json:"schedule"` // DAILY, HOURLY, ONLOGON, WEEKLY
		Time     string `json:"time"`     // HH:mm (ej: 20:00)
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "list":
		// Listar tareas creadas por el usuario o filtradas por Ozy
		out, err := exec.CommandContext(ctx, "schtasks", "/query", "/fo", "TABLE", "/nh").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando tareas programadas: %s", strings.TrimSpace(string(out))), false
		}
		lines := strings.Split(string(out), "\n")
		var userTasks []string
		for _, l := range lines {
			lTrim := strings.TrimSpace(l)
			if lTrim == "" {
				continue
			}
			// Filtrar tareas de usuario o que contengan Ozy
			if strings.Contains(strings.ToLower(lTrim), "ozy") || strings.HasPrefix(lTrim, "\\") {
				if len(userTasks) < 25 {
					userTasks = append(userTasks, lTrim)
				}
			}
		}

		if len(userTasks) == 0 {
			return "No hay tareas programadas personalizadas de Ozy registradas actualmente.", true
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== TAREAS PROGRAMADAS ACTIVAS (%d) ===\n", len(userTasks)))
		for i, t := range userTasks {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
		}
		return sb.String(), true

	case "create":
		if strings.TrimSpace(params.Name) == "" || strings.TrimSpace(params.Command) == "" {
			return "Debes proporcionar 'name' y 'command' para crear la tarea programada.", false
		}
		sc := strings.ToUpper(strings.TrimSpace(params.Schedule))
		if sc == "" {
			sc = "DAILY"
		}
		tTime := strings.TrimSpace(params.Time)
		if tTime == "" {
			tTime = "09:00"
		}

		taskName := "Ozy_" + strings.TrimSpace(params.Name)
		args := []string{"/create", "/tn", taskName, "/tr", params.Command, "/sc", sc}
		if sc == "DAILY" || sc == "WEEKLY" || sc == "HOURLY" {
			args = append(args, "/st", tTime)
		}
		args = append(args, "/f")

		out, err := exec.CommandContext(ctx, "schtasks", args...).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error creando tarea programada: %s", strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Tarea programada '%s' creada exitosamente con frecuencia %s a las %s.", taskName, sc, tTime), true

	case "delete":
		if strings.TrimSpace(params.Name) == "" {
			return "Debes proporcionar el nombre de la tarea a eliminar.", false
		}
		taskName := strings.TrimSpace(params.Name)
		if !strings.HasPrefix(taskName, "Ozy_") {
			taskName = "Ozy_" + taskName
		}
		out, err := exec.CommandContext(ctx, "schtasks", "/delete", "/tn", taskName, "/f").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error eliminando tarea programada: %s", strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Tarea programada '%s' eliminada exitosamente.", taskName), true

	default:
		return fmt.Sprintf("Acción de tarea desconocida: '%s'. Usa 'list', 'create' o 'delete'.", action), false
	}
}
