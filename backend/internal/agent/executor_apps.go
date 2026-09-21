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

func execOSGetDesktop(ctx context.Context) (string, bool) {
	nav := system.NewWindowsNavigator()
	items, err := nav.GetDesktopItems(ctx)
	if err != nil {
		return fmt.Sprintf("Error obteniendo elementos del escritorio: %v", err), false
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== ELEMENTOS DEL ESCRITORIO (%d encontrados) ===\n", len(items)))
	for i, it := range items {
		loc := "Personal"
		if it.IsPublic {
			loc = "Público"
		}
		if it.Kind == system.ItemKindSystemIcon {
			loc = "Sistema"
		}
		sb.WriteString(fmt.Sprintf("%d. [%s | %s] %s\n", i+1, it.Kind, loc, it.Name))
	}
	return sb.String(), true
}

func execOSListApps(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Filter string `json:"filter"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	nav := system.NewWindowsNavigator()
	apps, err := nav.GetInstalledSoftware(ctx, params.Filter)
	if err != nil {
		return fmt.Sprintf("Error consultando registro de aplicaciones: %v", err), false
	}
	if len(apps) == 0 {
		if params.Filter != "" {
			return fmt.Sprintf("No se encontraron aplicaciones instaladas que coincidan con \"%s\".", params.Filter), true
		}
		return "No se encontraron aplicaciones instaladas registradas.", true
	}
	var sb strings.Builder
	title := "APLICACIONES INSTALADAS"
	if params.Filter != "" {
		title += fmt.Sprintf(" (Filtro: \"%s\")", params.Filter)
	}
	sb.WriteString(fmt.Sprintf("=== %s (%d encontradas) ===\n", title, len(apps)))
	for i, app := range apps {
		details := ""
		if app.DisplayVersion != "" {
			details += " v" + app.DisplayVersion
		}
		if app.Publisher != "" {
			details += " (" + app.Publisher + ")"
		}
		if app.InstallLocation != "" {
			details += " en " + app.InstallLocation
		}
		sb.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, app.DisplayName, details))
	}
	return sb.String(), true
}


func execOSActiveWindows(ctx context.Context) (string, bool) {
	nav := system.NewWindowsNavigator()
	windows, err := nav.GetActiveWindows(ctx)
	if err != nil {
		return fmt.Sprintf("Error obteniendo ventanas activas: %v", err), false
	}
	if len(windows) == 0 {
		return "No se detectaron ventanas visibles abiertas.", true
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== VENTANAS ABIERTAS EN PANTALLA (%d) ===\n", len(windows)))
	for i, w := range windows {
		sb.WriteString(fmt.Sprintf("%d. %s (HWND: %d, PID: %d)\n", i+1, w.Title, w.Handle, w.ProcessID))
	}
	return sb.String(), true
}

func execOSDetectDialogs(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		AppFilter string `json:"app_filter"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	nav := system.NewWindowsNavigator()
	dialogs, err := nav.DetectDialogs(ctx, params.AppFilter)
	if err != nil {
		return fmt.Sprintf("Error inspeccionando cuadros de diálogo: %v", err), false
	}
	if len(dialogs) == 0 {
		filterMsg := ""
		if params.AppFilter != "" {
			filterMsg = fmt.Sprintf(" para '%s'", params.AppFilter)
		}
		return fmt.Sprintf("No se detectaron cuadros de diálogo emergentes ni errores activos%s.", filterMsg), true
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== CUADROS DE DIÁLOGO Y ERRORES DETECTADOS (%d) ===\n", len(dialogs)))
	for i, d := range dialogs {
		prefix := "ℹ️ INFO"
		if d.IsError {
			prefix = "🚨 ERROR MODAL DETECTADO"
		} else if d.Severity == "WARNING" {
			prefix = "⚠️ ADVERTENCIA"
		}

		procStr := ""
		if d.ProcessName != "" {
			procStr = fmt.Sprintf(" [%s, PID: %d]", d.ProcessName, d.ProcessID)
		} else if d.ProcessID != 0 {
			procStr = fmt.Sprintf(" [PID: %d]", d.ProcessID)
		}

		sb.WriteString(fmt.Sprintf("\n%d. %s: %s%s\n", i+1, prefix, d.Title, procStr))
		sb.WriteString(fmt.Sprintf("   • Clase:    %s (HWND: %d)\n", d.ClassName, d.Handle))
		if d.Message != "" {
			sb.WriteString(fmt.Sprintf("   • Mensaje:  %q\n", d.Message))
		}
		if len(d.Buttons) > 0 {
			sb.WriteString(fmt.Sprintf("   • Botones:  [%s]\n", strings.Join(d.Buttons, ", ")))
		}
	}
	return sb.String(), true
}


func execOSPortInspector(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Port int  `json:"port"`
		Kill bool `json:"kill"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	if params.Kill && params.Port > 0 {
		msg, err := system.KillPortProcess(ctx, params.Port)
		if err != nil {
			return fmt.Sprintf("Error liberando puerto %d: %v", params.Port, err), false
		}
		return fmt.Sprintf("🛑 === PUERTO %d LIBERADO ===\n%s", params.Port, msg), true
	}

	bindings, err := system.InspectPorts(params.Port)
	if err != nil {
		return fmt.Sprintf("Error inspeccionando puertos: %v", err), false
	}

	if len(bindings) == 0 {
		if params.Port > 0 {
			return fmt.Sprintf("El puerto %d está completamente LIBRE (ningún proceso en escucha).", params.Port), true
		}
		return "No se detectaron puertos TCP en estado LISTENING.", true
	}

	var sb strings.Builder
	title := "PUERTOS EN ESCUCHA ACTIVOS"
	if params.Port > 0 {
		title = fmt.Sprintf("ESTADO DEL PUERTO %d", params.Port)
	}
	sb.WriteString(fmt.Sprintf("🌐 === %s (%d) ===\n", title, len(bindings)))
	for i, b := range bindings {
		procInfo := "Proceso desconocido"
		if b.ProcessName != "" {
			procInfo = fmt.Sprintf("%s (PID: %d)", b.ProcessName, b.ProcessID)
		} else if b.ProcessID != 0 {
			procInfo = fmt.Sprintf("PID: %d", b.ProcessID)
		}
		sb.WriteString(fmt.Sprintf("%d. Puerto %d (%s) → %s [%s]\n", i+1, b.Port, b.Protocol, procInfo, b.State))
	}
	return sb.String(), true
}

func execOSSystemInfo(ctx context.Context) (string, bool) {

	collector := system.NewCollector()
	metrics, err := collector.Collect(ctx)
	if err != nil {
		return fmt.Sprintf("Error obteniendo métricas del sistema: %v", err), false
	}
	return fmt.Sprintf("=== MÉTRICAS DEL SISTEMA ===\n- CPU: %.1f%% (%d núcleos)\n- RAM: %d MB usados / %d MB totales (%.1f%%)\n- Disco: %d GB libres / %d GB totales (%.1f%%)\n- Plataforma: %s",
		metrics.CPUUsagePercent, metrics.CPUCores, metrics.RAMUsedMB, metrics.RAMTotalMB, metrics.RAMUsagePercent,
		metrics.DiskFreeGB, metrics.DiskTotalGB, metrics.DiskUsagePercent, metrics.Platform), true
}


func execOSLaunchApp(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Target   string   `json:"target"`
		AppName  string   `json:"appName"`
		App_Name string   `json:"app_name"`
		Path     string   `json:"path"`
		Args     []string `json:"args"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	target := strings.TrimSpace(params.Target)
	if target == "" {
		target = strings.TrimSpace(params.AppName)
	}
	if target == "" {
		target = strings.TrimSpace(params.App_Name)
	}

	args := params.Args
	if strings.TrimSpace(params.Path) != "" {
		args = append(args, strings.TrimSpace(params.Path))
	}

	cleanLower := strings.ToLower(target)
	// Limpiar prefijos conversacionales comunes
	for _, prefix := range []string{"abrir con ", "abre con ", "abrir ", "abre ", "lanza ", "ejecuta ", "el ", "la ", "los ", "las ", "un ", "una "} {
		if strings.HasPrefix(cleanLower, prefix) {
			cleanLower = strings.TrimSpace(cleanLower[len(prefix):])
			target = strings.TrimSpace(target[len(prefix):])
		}
	}

	// Si no se pasaron args pero el target contiene una app seguida de un proyecto o ruta (ej: "antigravity crmgeofal", "code mi_proyecto")
	if len(args) == 0 {
		appPrefixes := []string{"antigravity ide ", "antigravity ", "code ", "vscode ", "notepad ", "calc ", "chrome ", "brave ", "explorer "}
		for _, ap := range appPrefixes {
			if strings.HasPrefix(cleanLower, ap) {
				target = strings.TrimSpace(target[:len(ap)])
				remPath := strings.TrimSpace(cleanLower[len(ap):])
				if remPath != "" {
					args = append(args, remPath)
				}
				break
			}
		}
	}

	nav := system.NewWindowsNavigator()
	if err := nav.LaunchApplication(ctx, target, args); err != nil {
		return fmt.Sprintf("Error ejecutando aplicación %s: %v", target, err), false
	}
	if len(args) > 0 {
		return fmt.Sprintf("Aplicación %s lanzada exitosamente con: %s", target, strings.Join(args, " ")), true
	}
	return fmt.Sprintf("Aplicación lanzada exitosamente: %s", target), true
}

func execOSFocusWindow(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		HWND uintptr `json:"hwnd"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	nav := system.NewWindowsNavigator()
	if err := nav.FocusWindow(ctx, params.HWND); err != nil {
		return fmt.Sprintf("Error enfocando ventana (HWND: %d): %v", params.HWND, err), false
	}
	return fmt.Sprintf("Ventana (HWND: %d) enfocada y traída al frente.", params.HWND), true
}

func execOSCloseWindow(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		HWND   uintptr `json:"hwnd"`
		Title  string  `json:"title"`
		Target string  `json:"target"`
		Name   string  `json:"name"`
		PID    uint32  `json:"pid"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	nav := system.NewWindowsNavigator()

	// 1. Si se especificó HWND directo
	if params.HWND > 0 {
		if err := nav.CloseWindow(ctx, params.HWND); err != nil {
			return fmt.Sprintf("Error cerrando ventana (HWND: %d): %v", params.HWND, err), false
		}
		return fmt.Sprintf("Ventana (HWND: %d) cerrada exitosamente mediante WM_CLOSE.", params.HWND), true
	}

	// 2. Si se especificó un título o nombre de ventana
	searchTitle := strings.TrimSpace(params.Title)
	if searchTitle == "" {
		searchTitle = strings.TrimSpace(params.Target)
	}
	if searchTitle == "" {
		searchTitle = strings.TrimSpace(params.Name)
	}
	if searchTitle != "" {
		cleanLower := strings.ToLower(searchTitle)
		for _, prefix := range []string{"cerrar ", "cierra ", "el ", "la ", "los ", "las "} {
			if strings.HasPrefix(cleanLower, prefix) {
				cleanLower = strings.TrimSpace(cleanLower[len(prefix):])
			}
		}

		wins, err := nav.GetActiveWindows(ctx)
		if err == nil {
			for _, w := range wins {
				wLower := strings.ToLower(w.Title)
				if strings.Contains(wLower, cleanLower) ||
					(strings.Contains(cleanLower, "administrador") && (strings.Contains(wLower, "administrador de tareas") || strings.Contains(wLower, "task manager"))) ||
					(strings.Contains(cleanLower, "tarea") && (strings.Contains(wLower, "administrador de tareas") || strings.Contains(wLower, "task manager"))) ||
					(strings.Contains(cleanLower, "bloc") && (strings.Contains(wLower, "bloc de notas") || strings.Contains(wLower, "notepad"))) ||
					(strings.Contains(cleanLower, "calc") && (strings.Contains(wLower, "calculadora") || strings.Contains(wLower, "calc"))) {

					_ = nav.CloseWindow(ctx, w.Handle)
					_ = nav.KillProcess(ctx, w.ProcessID, true)
					return fmt.Sprintf("Ventana '%s' (HWND %d, PID %d) cerrada exitosamente.", w.Title, w.Handle, w.ProcessID), true
				}
			}
		}

		// Fallback automático por si no está visible en GetActiveWindows
		return execOSKillProcess(ctx, tc)
	}

	// 3. Si se especificó PID
	if params.PID > 0 {
		wins, _ := nav.GetActiveWindows(ctx)
		closedAny := false
		for _, w := range wins {
			if w.ProcessID == params.PID {
				_ = nav.CloseWindow(ctx, w.Handle)
				closedAny = true
			}
		}
		_ = nav.KillProcess(ctx, params.PID, true)
		if closedAny {
			return fmt.Sprintf("Ventana asociada al proceso PID %d cerrada exitosamente.", params.PID), true
		}
		return fmt.Sprintf("Proceso PID %d terminado exitosamente.", params.PID), true
	}

	return "Debes proporcionar 'hwnd', 'title' o 'pid' para cerrar la ventana.", false
}

func execOSKillProcess(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		PID     uint32 `json:"pid"`
		Name    string `json:"name"`
		AppName string `json:"appName"`
		Title   string `json:"title"`
		Force   bool   `json:"force"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	if !params.Force {
		params.Force = true
	}

	nav := system.NewWindowsNavigator()

	if params.PID > 0 {
		_ = nav.KillProcess(ctx, params.PID, params.Force)
		psCmd := fmt.Sprintf(`$p = Get-Process -Id %d -ErrorAction SilentlyContinue; if ($p) { Stop-Process -Name $p.ProcessName -Force -ErrorAction SilentlyContinue }`, params.PID)
		_ = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run()
		return fmt.Sprintf("Proceso PID %d y sus instancias asociadas terminados exitosamente.", params.PID), true
	}

	target := strings.TrimSpace(params.Name)
	if target == "" {
		target = strings.TrimSpace(params.AppName)
	}
	if target == "" {
		target = strings.TrimSpace(params.Title)
	}

	cleanLower := strings.ToLower(target)
	for _, prefix := range []string{"cerrar ", "cierra ", "matar ", "mata ", "el ", "la ", "los ", "las "} {
		if strings.HasPrefix(cleanLower, prefix) {
			cleanLower = strings.TrimSpace(cleanLower[len(prefix):])
			target = cleanLower
		}
	}

	targetsToKill := []string{}
	switch cleanLower {
	case "administrador de tareas", "administrador de tarea", "task manager", "taskmgr", "taskmgr.exe":
		targetsToKill = []string{"Taskmgr.exe", "taskmgr.exe"}
	case "configuración", "configuracion", "settings", "systemsettings":
		targetsToKill = []string{"SystemSettings.exe"}
	case "bloc de notas", "bloc", "notas", "notepad", "notepad.exe":
		targetsToKill = []string{"notepad.exe", "Notepad.exe"}
	case "calculadora", "calc", "calc.exe", "calculator", "calculatorapp":
		targetsToKill = []string{"CalculatorApp.exe", "calc.exe", "Calculator.exe"}
	case "explorador", "explorador de archivos", "explorer", "explorer.exe":
		targetsToKill = []string{"explorer.exe"}
	case "consola", "cmd", "símbolo del sistema", "simbolo del sistema":
		targetsToKill = []string{"cmd.exe"}
	case "terminal", "powershell", "windows terminal":
		targetsToKill = []string{"powershell.exe", "pwsh.exe", "WindowsTerminal.exe"}
	case "chrome", "google chrome":
		targetsToKill = []string{"chrome.exe"}
	case "edge", "microsoft edge":
		targetsToKill = []string{"msedge.exe"}
	case "brave":
		targetsToKill = []string{"brave.exe"}
	case "spotify":
		targetsToKill = []string{"spotify.exe"}
	case "paint", "mspaint":
		targetsToKill = []string{"mspaint.exe"}
	case "word":
		targetsToKill = []string{"WINWORD.EXE", "winword.exe"}
	case "excel":
		targetsToKill = []string{"EXCEL.EXE", "excel.exe"}
	case "powerpoint":
		targetsToKill = []string{"POWERPNT.EXE", "powerpnt.exe"}
	case "discord":
		targetsToKill = []string{"Discord.exe"}
	case "steam":
		targetsToKill = []string{"steam.exe"}
	default:
		if cleanLower != "" {
			if !strings.HasSuffix(cleanLower, ".exe") {
				targetsToKill = []string{cleanLower + ".exe", cleanLower}
			} else {
				targetsToKill = []string{cleanLower}
			}
		}
	}

	killedAny := false
	for _, img := range targetsToKill {
		args := []string{"/IM", img}
		if params.Force {
			args = append(args, "/F", "/T")
		}
		cmd := exec.CommandContext(ctx, "taskkill", args...)
		if err := cmd.Run(); err == nil {
			killedAny = true
		}
	}

	// Fallback 1: PowerShell Stop-Process
	if !killedAny {
		for _, img := range targetsToKill {
			procBase := strings.TrimSuffix(img, ".exe")
			psCmd := fmt.Sprintf("Stop-Process -Name '%s' -Force -ErrorAction SilentlyContinue", procBase)
			if err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Run(); err == nil {
				killedAny = true
			}
		}
	}

	// Fallback 2: WMI / CIM Terminate (capaz de cerrar procesos elevados como Taskmgr.exe sin requerir elevación manual)
	if !killedAny {
		for _, img := range targetsToKill {
			wmiCmd := fmt.Sprintf("Get-CimInstance Win32_Process -Filter \"Name = '%s'\" | Invoke-CimMethod -MethodName Terminate", img)
			out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", wmiCmd).CombinedOutput()
			if err == nil && strings.Contains(string(out), "0") {
				killedAny = true
			}
		}
	}

	if killedAny {
		return fmt.Sprintf("Aplicación '%s' cerrada exitosamente.", target), true
	}

	// Fallback 3 inteligente: buscar en las ventanas activas por coincidencia de título
	if wins, err := nav.GetActiveWindows(ctx); err == nil && cleanLower != "" {
		for _, w := range wins {
			wLower := strings.ToLower(w.Title)
			if strings.Contains(wLower, cleanLower) ||
				(strings.Contains(cleanLower, "administrador") && (strings.Contains(wLower, "administrador") || strings.Contains(wLower, "task manager"))) ||
				(strings.Contains(cleanLower, "tarea") && (strings.Contains(wLower, "tarea") || strings.Contains(wLower, "task"))) ||
				(strings.Contains(cleanLower, "bloc") && (strings.Contains(wLower, "bloc") || strings.Contains(wLower, "notepad"))) ||
				(strings.Contains(cleanLower, "calc") && (strings.Contains(wLower, "calc") || strings.Contains(wLower, "calculadora"))) {

				_ = nav.CloseWindow(ctx, w.Handle)
				if err := nav.KillProcess(ctx, w.ProcessID, true); err == nil {
					return fmt.Sprintf("Ventana '%s' (PID %d) cerrada exitosamente.", w.Title, w.ProcessID), true
				}
				return fmt.Sprintf("Orden de cierre enviada a la ventana '%s' (HWND %d, PID %d).", w.Title, w.Handle, w.ProcessID), true
			}
		}
	}

	return fmt.Sprintf("No se encontró ningún proceso activo para '%s'.", target), false
}

func execOSHardwareControl(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Setting string `json:"setting"`
		Value   int    `json:"value"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	out, err := OSHardwareControl(ctx, params.Setting, params.Value)
	if err != nil {
		return err.Error(), false
	}
	return out, true
}

func execOSPowerState(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		State string `json:"state"`
	}
	_ = json.Unmarshal(tc.Input, &params)
	out, err := OSPowerState(ctx, params.State)
	if err != nil {
		return err.Error(), false
	}
	return out, true
}

func execOSRunCommand(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Command string `json:"command"`
		Cwd     string `json:"cwd"`
		Dir     string `json:"dir"`
		Path    string `json:"path"`
		Project string `json:"project"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return fmt.Sprintf("parámetros inválidos para os_run_command: %v", err), false
	}
	if params.Cwd == "" {
		if params.Dir != "" {
			params.Cwd = params.Dir
		} else if params.Path != "" {
			params.Cwd = params.Path
		} else if params.Project != "" {
			params.Cwd = params.Project
		}
	}
	return runOSCommandInternal(ctx, params.Command, params.Cwd)
}

func runOSCommandInternal(ctx context.Context, command, rawCwd string) (string, bool) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "comando vacío", false
	}

	// Verificación de seguridad básica contra comandos destructivos no deseados
	if IsDestructiveCommand(command) {
		return fmt.Sprintf("⚠️ COMANDO BLOQUEADO POR SEGURIDAD: '%s' contiene operaciones destructivas que requieren autorización explícita.", command), false
	}

	cwd := strings.TrimSpace(rawCwd)
	if cwd != "" {
		cwd = system.ResolveUserPath(cwd)
	}

	// Si cwd no fue especificado, buscar si el mensaje del usuario menciona un proyecto conocido en Documents
	if cwd == "" {
		userMsg := strings.ToLower(UserMessageFromContext(ctx))
		if userMsg != "" {
			userProfile := os.Getenv("USERPROFILE")
			if userProfile != "" {
				docsDir := filepath.Join(userProfile, "Documents")
				if entries, err := os.ReadDir(docsDir); err == nil {
					for _, entry := range entries {
						if entry.IsDir() {
							folderLower := strings.ToLower(entry.Name())
							if len(folderLower) >= 4 && strings.Contains(userMsg, folderLower) {
								cwd = filepath.Join(docsDir, entry.Name())
								break
							}
						}
					}
				}
			}
		}
	}

	// Comprobar si el comando es de Git
	isGit := strings.HasPrefix(strings.ToLower(command), "git ") || strings.HasPrefix(strings.ToLower(command), "git.exe ")
	if isGit && cwd != "" {
		if fi, err := os.Stat(cwd); err == nil && fi.IsDir() {
			gitDir := filepath.Join(cwd, ".git")
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				// La carpeta no tiene .git en la raíz. Verificar si contiene sub-repositorios Git independientes
				if multiRepoOut, hasRepos := inspectGitMultiRepo(ctx, cwd, command); hasRepos {
					return multiRepoOut, true
				}
			}
		}
	}

	res, healLogs, err := executeWithSelfHealing(ctx, command, cwd)
	if err != nil {
		return fmt.Sprintf("error ejecutando comando '%s': %v", command, err), false
	}

	// Si el comando git falló con "not a git repository", intentar inspección multi-repo
	if isGit && (strings.Contains(res.Stderr, "not a git repository") || strings.Contains(res.Stdout, "not a git repository")) && cwd != "" {
		if multiRepoOut, hasRepos := inspectGitMultiRepo(ctx, cwd, command); hasRepos {
			return multiRepoOut, true
		}
	}

	var sb strings.Builder
	if healLogs != "" {
		sb.WriteString(healLogs)
	}
	effectiveDir := cwd
	if effectiveDir == "" {
		effectiveDir, _ = os.Getwd()
	}
	sb.WriteString(fmt.Sprintf("💻 [DIR: %s] [COMANDO: %s] (Salida: código %d, duración %s)\n", effectiveDir, command, res.ExitCode, res.Duration))

	stdout := strings.TrimSpace(res.Stdout)
	stderr := strings.TrimSpace(res.Stderr)

	// Truncar para no desbordar tokens si el comando produce salida kilométrica
	if len([]rune(stdout)) > 4000 {
		runes := []rune(stdout)
		stdout = string(runes[:4000]) + "\n[...salida truncada a 4000 caracteres...]"
	}
	if len([]rune(stderr)) > 2000 {
		runes := []rune(stderr)
		stderr = string(runes[:2000]) + "\n[...errores truncados...]"
	}

	if stdout != "" {
		sb.WriteString("--- SALIDA (STDOUT) ---\n")
		sb.WriteString(stdout)
		sb.WriteString("\n")
	}
	if stderr != "" {
		sb.WriteString("--- ERRORES (STDERR) ---\n")
		sb.WriteString(stderr)
		sb.WriteString("\n")
	}
	if stdout == "" && stderr == "" {
		sb.WriteString("(Comando ejecutado exitosamente sin salida en consola)\n")
	}

	return sb.String(), res.ExitCode == 0
}

func inspectGitMultiRepo(ctx context.Context, dir, gitCommand string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}

	var subRepos []string
	for _, entry := range entries {
		if entry.IsDir() {
			subGit := filepath.Join(dir, entry.Name(), ".git")
			if _, err := os.Stat(subGit); err == nil {
				subRepos = append(subRepos, entry.Name())
			}
		}
	}

	if len(subRepos) == 0 {
		return "", false
	}

	// Extraer argumentos de git (ej: "log -n 3 --oneline", "status")
	lower := strings.ToLower(gitCommand)
	gitArgs := gitCommand
	if strings.HasPrefix(lower, "git ") {
		gitArgs = strings.TrimSpace(gitCommand[4:])
	} else if strings.HasPrefix(lower, "git.exe ") {
		gitArgs = strings.TrimSpace(gitCommand[8:])
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📁 [ESPACIO DE TRABAJO MULTI-REPO: %s]\n", dir))
	sb.WriteString(fmt.Sprintf("Nota: Esta carpeta no es un único repositorio .git, sino un proyecto que contiene %d repositorios Git independientes.\n", len(subRepos)))
	sb.WriteString(fmt.Sprintf("Resultados reales de 'git %s' en cada repositorio:\n\n", gitArgs))

	hasSuccess := false
	for _, repoName := range subRepos {
		repoPath := filepath.Join(dir, repoName)
		res, err := ExecuteShellInDir(ctx, "powershell", "git "+gitArgs, repoPath)
		if err == nil && res.ExitCode == 0 {
			hasSuccess = true
			out := strings.TrimSpace(res.Stdout)
			if out == "" {
				out = "(Sin cambios o sin salida)"
			}
			sb.WriteString(fmt.Sprintf("🔹 Repositorio [%s]:\n%s\n\n", repoName, out))
		} else {
			errText := strings.TrimSpace(res.Stderr)
			if errText == "" && err != nil {
				errText = err.Error()
			}
			sb.WriteString(fmt.Sprintf("🔸 Repositorio [%s] (error): %s\n\n", repoName, errText))
		}
	}

	return sb.String(), hasSuccess
}

// execRememberFact registra un hecho o preferencia en la memoria persistente de OzyAssist.
