package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
)

// execOSKeyboardLayout inspecciona o cambia la distribución de teclado y cultura de entrada en Windows
func execOSKeyboardLayout(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // status, list, set
		Layout string `json:"layout"` // latam, spain, us, o KLID hex ej. 0000080A
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		if strings.TrimSpace(params.Layout) != "" {
			action = "set"
		} else {
			action = "status"
		}
	}

	switch action {
	case "status", "current", "get":
		psCmd := `
			[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null
			$curr = [System.Windows.Forms.InputLanguage]::CurrentInputLanguage
			$cul = Get-Culture
			$override = (Get-WinDefaultInputMethodOverride -ErrorAction SilentlyContinue).InputTip
			if (-not $override) { $override = "Predeterminado de Windows" }
			Write-Output "=== DISTRIBUCIÓN DE TECLADO Y MÉTODO DE ENTRADA ==="
			Write-Output "Distribución activa:     $($curr.LayoutName)"
			Write-Output "Cultura de entrada:      $($curr.Culture.DisplayName) ($($curr.Culture.Name))"
			Write-Output "Cultura regional del OS: $($cul.DisplayName) ($($cul.Name))"
			Write-Output "Override predeterminado: $override"
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando distribución de teclado: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "list", "installed":
		psCmd := `
			[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null
			$installed = [System.Windows.Forms.InputLanguage]::InstalledInputLanguages
			$userLangs = Get-WinUserLanguageList -ErrorAction SilentlyContinue
			Write-Output "=== DISTRIBUCIONES DE TECLADO INSTALADAS ==="
			foreach ($l in $installed) {
				Write-Output " - $($l.LayoutName) | Cultura: $($l.Culture.Name) ($($l.Culture.DisplayName))"
			}
			if ($userLangs) {
				Write-Output ""
				Write-Output "=== IDIOMAS PREFERIDOS DE WINDOWS ==="
				foreach ($u in $userLangs) {
					$tips = $u.InputMethodTips -join ', '
					Write-Output " - $($u.LanguageTag): $($u.Autonym) (Métodos: $tips)"
				}
			}
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error listando teclados instalados: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "set", "switch", "change":
		target := strings.ToLower(strings.TrimSpace(params.Layout))
		if target == "" {
			return "Debes indicar la distribución de teclado deseada ('layout', ej: 'latam', 'spain', 'us').", false
		}

		// Mapeo amigable a identificadores KLID de Windows (Keyboard Layout Identifier)
		klidMap := map[string]string{
			"latam":         "0000080A",
			"latinoamerica": "0000080A",
			"latin":         "0000080A",
			"es-419":        "0000080A",
			"es-pe":         "0000080A",
			"es-mx":         "0000080A",
			"es-co":         "0000080A",
			"es-ar":         "0000080A",
			"spain":         "0000040A",
			"españa":        "0000040A",
			"es-es":         "0000040A",
			"spanish":       "0000040A",
			"us":            "00000409",
			"usa":           "00000409",
			"en-us":         "00000409",
			"english":       "00000409",
			"ingles":        "00000409",
		}

		klid, ok := klidMap[target]
		if !ok {
			// Si ya tiene formato hex de 8 caracteres ej. 0000080A
			if len(target) == 8 {
				klid = strings.ToUpper(target)
			} else {
				return fmt.Sprintf("Distribución no reconocida: '%s'. Opciones comunes: 'latam' (Latinoamérica), 'spain' (España), 'us' (Inglés Estados Unidos).", target), false
			}
		}

		psCmd := fmt.Sprintf(`
			$src = @"
using System;
using System.Runtime.InteropServices;
public class OzyKB {
    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern IntPtr LoadKeyboardLayout(string pwszKLID, uint Flags);
    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);
    public static void Switch(string klid) {
        IntPtr hkl = LoadKeyboardLayout(klid, 1);
        PostMessage((IntPtr)0xffff, 0x0050, IntPtr.Zero, hkl);
    }
}
"@
			if (-not ([System.Management.Automation.PSTypeName]'OzyKB').Type) {
				Add-Type -TypeDefinition $src
			}
			[OzyKB]::Switch("%s")
			Write-Output "Distribución de teclado cambiada exitosamente a KLID %s."
		`, klid, klid)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error cambiando distribución de teclado: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'status', 'list' o 'set'.", action), false
	}
}

// execOSStartupManager administra los programas y tareas que se ejecutan al iniciar sesión en Windows
func execOSStartupManager(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action  string `json:"action"`  // list, add, remove
		Name    string `json:"name"`    // Nombre del programa o entrada
		Command string `json:"command"` // Ruta o comando ejecutable (para add)
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "list"
	}

	switch action {
	case "list", "status":
		psCmd := `
			Write-Output "=== PROGRAMAS DE INICIO AUTOMÁTICO DE WINDOWS ==="
			
			Write-Output ""
			Write-Output "[Usuario - HKCU\Software\Microsoft\Windows\CurrentVersion\Run]"
			$hkcu = Get-ItemProperty "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run" -ErrorAction SilentlyContinue
			if ($hkcu) {
				$hkcu.PSObject.Properties | Where-Object { $_.Name -notmatch "^PS" } | ForEach-Object {
					Write-Output " - $($_.Name): $($_.Value)"
				}
			} else {
				Write-Output " (Sin entradas registradas)"
			}

			Write-Output ""
			Write-Output "[Sistema - HKLM\Software\Microsoft\Windows\CurrentVersion\Run]"
			$hklm = Get-ItemProperty "HKLM:\Software\Microsoft\Windows\CurrentVersion\Run" -ErrorAction SilentlyContinue
			if ($hklm) {
				$hklm.PSObject.Properties | Where-Object { $_.Name -notmatch "^PS" } | ForEach-Object {
					Write-Output " - $($_.Name): $($_.Value)"
				}
			} else {
				Write-Output " (Sin entradas registradas)"
			}

			Write-Output ""
			Write-Output "[Carpeta de Inicio de Usuario (Startup)]"
			$startupPath = [System.Environment]::GetFolderPath('Startup')
			$files = Get-ChildItem -Path $startupPath -ErrorAction SilentlyContinue
			if ($files) {
				foreach ($f in $files) {
					Write-Output " - $($f.Name)"
				}
			} else {
				Write-Output " (Carpeta vacía)"
			}
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando programas de inicio: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "add", "enable":
		name := strings.TrimSpace(params.Name)
		cmd := strings.TrimSpace(params.Command)
		if name == "" || cmd == "" {
			return "Debes especificar tanto el nombre ('name') como la ruta o comando ('command') a agregar al inicio.", false
		}
		escName := strings.ReplaceAll(name, "'", "''")
		escCmd := strings.ReplaceAll(cmd, "'", "''")

		psCmd := fmt.Sprintf(`
			Set-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run" -Name '%s' -Value '%s' -ErrorAction Stop
			Write-Output "Programa '%s' agregado exitosamente al inicio de Windows (HKCU\Run)."
		`, escName, escCmd, escName)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error agregando programa al inicio: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "remove", "delete", "disable":
		name := strings.TrimSpace(params.Name)
		if name == "" {
			return "Debes indicar el nombre ('name') del programa a eliminar del inicio.", false
		}
		escName := strings.ReplaceAll(name, "'", "''")

		psCmd := fmt.Sprintf(`
			$removed = $false
			$val = Get-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run" -Name '%s' -ErrorAction SilentlyContinue
			if ($val) {
				Remove-ItemProperty -Path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run" -Name '%s' -ErrorAction Stop
				$removed = $true
			}
			$startupPath = [System.Environment]::GetFolderPath('Startup')
			$shortcut = Join-Path $startupPath ('%s.lnk')
			if (Test-Path $shortcut) {
				Remove-Item $shortcut -Force
				$removed = $true
			}
			if ($removed) {
				Write-Output "Programa '%s' retirado exitosamente del inicio de Windows."
			} else {
				Write-Output "No se encontró ninguna entrada de inicio con el nombre '%s'."
			}
		`, escName, escName, escName, escName, escName)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error eliminando programa del inicio: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'list', 'add' o 'remove'.", action), false
	}
}

// execOSNotificationFocus consulta o alterna el estado de notificaciones y modo no molestar de Windows
func execOSNotificationFocus(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action  string `json:"action"`  // status, set
		Enabled *bool  `json:"enabled"` // true: habilitar notificaciones; false: silenciar / no molestar
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		if params.Enabled != nil {
			action = "set"
		} else {
			action = "status"
		}
	}

	switch action {
	case "status", "get":
		psCmd := `
			$p = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings"
			$v = Get-ItemProperty -Path $p -Name "NOC_GLOBAL_SETTING_ALLOW_TOASTS" -ErrorAction SilentlyContinue
			$state = if ($v -ne $null -and $v.NOC_GLOBAL_SETTING_ALLOW_TOASTS -eq 0) {
				"[SILENCIADO / MODO NO MOLESTAR]"
			} else {
				"[HABILITADAS / NORMAL]"
			}
			Write-Output "=== ESTADO DE NOTIFICACIONES DE WINDOWS ==="
			Write-Output "Notificaciones emergentes (Toasts): $state"
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando estado de notificaciones: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "set", "toggle":
		if params.Enabled == nil {
			return "Debes indicar si deseas habilitar ('enabled': true) o silenciar ('enabled': false) las notificaciones.", false
		}
		val := 1
		msg := "Notificaciones emergentes habilitadas normalmente en Windows."
		if !*params.Enabled {
			val = 0
			msg = "Notificaciones emergentes silenciadas (Modo No Molestar activado)."
		}

		psCmd := fmt.Sprintf(`
			$p = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Notifications\Settings"
			Set-ItemProperty -Path $p -Name "NOC_GLOBAL_SETTING_ALLOW_TOASTS" -Value %d -Type DWord -Force
			Write-Output "%s"
		`, val, msg)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error configurando notificaciones: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'status' o 'set'.", action), false
	}
}

// execOSMediaControl controla la reproducción multimedia global de Windows (Spotify, YouTube, VLC, etc.)
func execOSMediaControl(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // play_pause, next, previous, stop
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "play_pause"
	}

	var vkCode byte
	var actionDesc string
	switch action {
	case "play_pause", "play", "pause", "reproducir", "pausar":
		vkCode = 0xB3 // VK_MEDIA_PLAY_PAUSE
		actionDesc = "Reproducir / Pausar (Play/Pause)"
	case "next", "siguiente", "adelantar", "skip":
		vkCode = 0xB0 // VK_MEDIA_NEXT_TRACK
		actionDesc = "Pista siguiente (Next Track)"
	case "previous", "prev", "anterior", "retroceder":
		vkCode = 0xB1 // VK_MEDIA_PREV_TRACK
		actionDesc = "Pista anterior (Previous Track)"
	case "stop", "detener", "parar":
		vkCode = 0xB2 // VK_MEDIA_STOP
		actionDesc = "Detener reproducción (Stop)"
	default:
		return fmt.Sprintf("Acción multimedia no reconocida: '%s'. Opciones: 'play_pause', 'next', 'previous', 'stop'.", action), false
	}

	psCmd := fmt.Sprintf(`
		$src = @"
using System;
using System.Runtime.InteropServices;
public class OzyMedia {
    [DllImport("user32.dll")]
    public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);
    public static void Press(byte key) {
        keybd_event(key, 0, 0, UIntPtr.Zero);
        keybd_event(key, 0, 2, UIntPtr.Zero);
    }
}
"@
		if (-not ([System.Management.Automation.PSTypeName]'OzyMedia').Type) {
			Add-Type -TypeDefinition $src
		}
		[OzyMedia]::Press(%d)
		Write-Output "Control multimedia ejecutado exitosamente: %s."
	`, vkCode, actionDesc)

	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error emitiendo comando multimedia: %s", strings.TrimSpace(string(out))), false
	}
	return strings.TrimSpace(string(out)), true
}

// execOSDisplayConfig consulta y conmuta la configuración de monitores y pantallas (displayswitch.exe)
func execOSDisplayConfig(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // status, extend, clone, internal, external
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		action = "status"
	}

	switch action {
	case "status", "list":
		psCmd := `
			[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null
			$screens = [System.Windows.Forms.Screen]::AllScreens
			Write-Output "=== MONITORES Y PANTALLAS CONECTADAS ($($screens.Count)) ==="
			foreach ($s in $screens) {
				$prim = if ($s.Primary) { "[PRIMARIA]" } else { "[SECUNDARIA]" }
				Write-Output " - $($s.DeviceName) $($prim): $($s.Bounds.Width)x$($s.Bounds.Height) @ ($($s.Bounds.X),$($s.Bounds.Y))"
			}
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando pantallas: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "extend", "extender":
		out, err := exec.CommandContext(ctx, "displayswitch.exe", "/extend").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error conmutando a modo extendido: %s", strings.TrimSpace(string(out))), false
		}
		return "Modo de pantalla configurado a: Extender (Escritorio ampliado en monitores adicionales).", true

	case "clone", "duplicate", "duplicar":
		out, err := exec.CommandContext(ctx, "displayswitch.exe", "/clone").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error conmutando a modo duplicado: %s", strings.TrimSpace(string(out))), false
		}
		return "Modo de pantalla configurado a: Duplicar (Misma imagen reflejada en todas las pantallas).", true

	case "internal", "pc_only", "solo_pc":
		out, err := exec.CommandContext(ctx, "displayswitch.exe", "/internal").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error conmutando a solo pantalla de PC: %s", strings.TrimSpace(string(out))), false
		}
		return "Modo de pantalla configurado a: Solo pantalla de la PC (Monitores externos apagados).", true

	case "external", "second_screen", "segunda_pantalla":
		out, err := exec.CommandContext(ctx, "displayswitch.exe", "/external").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error conmutando a solo segunda pantalla: %s", strings.TrimSpace(string(out))), false
		}
		return "Modo de pantalla configurado a: Solo segunda pantalla.", true

	default:
		return fmt.Sprintf("Modo de pantalla no reconocido: '%s'. Opciones: 'status', 'extend', 'clone', 'internal', 'external'.", action), false
	}
}

// execOSProcessSentinel audita procesos de alto consumo de memoria o CPU y permite terminación limpia
func execOSProcessSentinel(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action    string `json:"action"`     // top_memory, top_cpu, kill
		TopN      int    `json:"top_n"`      // Cantidad de procesos a devolver (default: 10)
		ProcessID int    `json:"process_id"` // PID a finalizar si action == kill
		Name      string `json:"name"`       // Nombre del proceso si action == kill
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		if params.ProcessID > 0 || params.Name != "" {
			action = "kill"
		} else {
			action = "top_memory"
		}
	}

	topN := params.TopN
	if topN <= 0 || topN > 30 {
		topN = 10
	}

	switch action {
	case "top_memory", "memory", "ram":
		psCmd := fmt.Sprintf(`
			$procs = Get-Process -ErrorAction SilentlyContinue | Sort-Object WorkingSet64 -Descending | Select-Object -First %d
			Write-Output "=== TOP %d PROCESOS CON MAYOR CONSUMO DE MEMORIA RAM ==="
			foreach ($p in $procs) {
				$mb = [Math]::Round($p.WorkingSet64 / 1MB, 1)
				$cpuSec = [Math]::Round($p.CPU, 1)
				Write-Output " - $($p.ProcessName) (PID $($p.Id)): $mb MB RAM | CPU acumulado: $cpuSec s"
			}
		`, topN, topN)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando procesos por memoria: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "top_cpu", "cpu":
		psCmd := fmt.Sprintf(`
			$procs = Get-Process -ErrorAction SilentlyContinue | Sort-Object CPU -Descending | Select-Object -First %d
			Write-Output "=== TOP %d PROCESOS CON MAYOR CONSUMO DE CPU ==="
			foreach ($p in $procs) {
				$mb = [Math]::Round($p.WorkingSet64 / 1MB, 1)
				$cpuSec = [Math]::Round($p.CPU, 1)
				Write-Output " - $($p.ProcessName) (PID $($p.Id)): $cpuSec s CPU | RAM: $mb MB"
			}
		`, topN, topN)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando procesos por CPU: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "kill", "terminate", "stop":
		if params.ProcessID <= 0 && strings.TrimSpace(params.Name) == "" {
			return "Debes indicar el PID ('process_id') o nombre del proceso ('name') a finalizar.", false
		}

		psCmd := ""
		if params.ProcessID > 0 {
			psCmd = fmt.Sprintf(`Stop-Process -Id %d -Force -ErrorAction Stop; Write-Output "Proceso con PID %d finalizado exitosamente."`, params.ProcessID, params.ProcessID)
		} else {
			escName := strings.ReplaceAll(params.Name, "'", "''")
			psCmd = fmt.Sprintf(`Stop-Process -Name '%s' -Force -ErrorAction Stop; Write-Output "Proceso '%s' finalizado exitosamente."`, escName, escName)
		}

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error finalizando proceso: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'top_memory', 'top_cpu' o 'kill'.", action), false
	}
}

// execOSSpeakText sintetiza texto en voz hablada directamente por los altavoces mediante Windows SAPI local
func execOSSpeakText(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Text  string `json:"text"`
		Voice string `json:"voice"` // ej: "Microsoft Helena Desktop"
	}
	_ = json.Unmarshal(tc.Input, &params)

	text := strings.TrimSpace(params.Text)
	if text == "" {
		return "Debes indicar el texto a pronunciar ('text').", false
	}

	escText := strings.ReplaceAll(text, "'", "''")
	voiceSelect := ""
	if strings.TrimSpace(params.Voice) != "" {
		escVoice := strings.ReplaceAll(strings.TrimSpace(params.Voice), "'", "''")
		voiceSelect = fmt.Sprintf(`$synth.SelectVoice('%s');`, escVoice)
	} else {
		// Seleccionar automáticamente voz en español si está disponible
		voiceSelect = `
			$esVoice = $synth.GetInstalledVoices() | Where-Object { $_.VoiceInfo.Culture -like 'es*' } | Select-Object -First 1
			if ($esVoice) { $synth.SelectVoice($esVoice.VoiceInfo.Name) }
		`
	}

	psCmd := fmt.Sprintf(`
		Add-Type -AssemblyName System.Speech
		$synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
		%s
		$synth.Speak('%s')
		Write-Output "Texto pronunciado exitosamente vía Windows SAPI local."
	`, voiceSelect, escText)

	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error sintetizando voz hablada: %s", strings.TrimSpace(string(out))), false
	}
	return strings.TrimSpace(string(out)), true
}

