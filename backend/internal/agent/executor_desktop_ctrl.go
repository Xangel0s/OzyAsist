package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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

// execOSAudioDevice administra dispositivos de salida de sonido, volumen maestro y silencio en Windows
func execOSAudioDevice(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string   `json:"action"` // list, status, set, get_volume, set_volume, mute, unmute
		Name   string   `json:"name"`
		Level  *float64 `json:"level"`
		Mute   *bool    `json:"mute"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		if params.Level != nil {
			action = "set_volume"
		} else if params.Mute != nil {
			action = "mute"
		} else {
			action = "list"
		}
	}

	coreAudioDef := `@'
using System;
using System.Runtime.InteropServices;
namespace OzyAudio {
    [Guid("5CDF2C82-841E-4546-9722-0CF74078229A"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IAudioEndpointVolume {
        [PreserveSig] int RegisterControlChangeNotify(IntPtr pNotify);
        [PreserveSig] int UnregisterControlChangeNotify(IntPtr pNotify);
        [PreserveSig] int GetChannelCount(out uint pnChannelCount);
        [PreserveSig] int SetMasterVolumeLevel(float fLevelDB, ref Guid pguidEventContext);
        [PreserveSig] int SetMasterVolumeLevelScalar(float fLevel, ref Guid pguidEventContext);
        [PreserveSig] int GetMasterVolumeLevel(out float pfLevelDB);
        [PreserveSig] int GetMasterVolumeLevelScalar(out float pfLevel);
        [PreserveSig] int SetChannelVolumeLevel(uint nChannel, float fLevelDB, ref Guid pguidEventContext);
        [PreserveSig] int SetChannelVolumeLevelScalar(uint nChannel, float fLevel, ref Guid pguidEventContext);
        [PreserveSig] int GetChannelVolumeLevel(uint nChannel, out float pfLevelDB);
        [PreserveSig] int GetChannelVolumeLevelScalar(uint nChannel, out float pfLevel);
        [PreserveSig] int SetMute([MarshalAs(UnmanagedType.Bool)] bool bMute, ref Guid pguidEventContext);
        [PreserveSig] int GetMute([MarshalAs(UnmanagedType.Bool)] out bool pbMute);
        [PreserveSig] int GetVolumeStepInfo(out uint pnStep, out uint pnStepCount);
        [PreserveSig] int VolumeStepUp(ref Guid pguidEventContext);
        [PreserveSig] int VolumeStepDown(ref Guid pguidEventContext);
        [PreserveSig] int QueryHardwareSupport(out uint pdwHardwareSupportMask);
        [PreserveSig] int GetVolumeRange(out float pflVolumeMindB, out float pflVolumeMaxdB, out float pflVolumeIncrementdB);
    }
    [Guid("D666063F-1587-4E43-81F1-B948E807363F"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IMMDevice {
        [PreserveSig] int Activate(ref Guid iid, int dwClsCtx, IntPtr pActivationParams, [MarshalAs(UnmanagedType.IUnknown)] out object ppInterface);
    }
    [Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    public interface IMMDeviceEnumerator {
        [PreserveSig] int EnumAudioEndpoints(int dataFlow, int dwStateMask, out IntPtr ppDevices);
        [PreserveSig] int GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice ppEndpoint);
    }
    [ComImport, Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")]
    public class MMDeviceEnumeratorComObject {}
    public class MasterAudio {
        public static float GetVolume() {
            var enumerator = (IMMDeviceEnumerator)(new MMDeviceEnumeratorComObject());
            IMMDevice dev;
            if (enumerator.GetDefaultAudioEndpoint(0, 1, out dev) != 0 || dev == null) return -1f;
            Guid iid = typeof(IAudioEndpointVolume).GUID;
            object epvObj;
            if (dev.Activate(ref iid, 23, IntPtr.Zero, out epvObj) != 0 || epvObj == null) return -1f;
            var epv = (IAudioEndpointVolume)epvObj;
            float vol = 0;
            epv.GetMasterVolumeLevelScalar(out vol);
            return vol * 100f;
        }
        public static bool GetMute() {
            var enumerator = (IMMDeviceEnumerator)(new MMDeviceEnumeratorComObject());
            IMMDevice dev;
            if (enumerator.GetDefaultAudioEndpoint(0, 1, out dev) != 0 || dev == null) return false;
            Guid iid = typeof(IAudioEndpointVolume).GUID;
            object epvObj;
            if (dev.Activate(ref iid, 23, IntPtr.Zero, out epvObj) != 0 || epvObj == null) return false;
            var epv = (IAudioEndpointVolume)epvObj;
            bool mute = false;
            epv.GetMute(out mute);
            return mute;
        }
        public static void SetVolume(float pct) {
            var enumerator = (IMMDeviceEnumerator)(new MMDeviceEnumeratorComObject());
            IMMDevice dev;
            if (enumerator.GetDefaultAudioEndpoint(0, 1, out dev) != 0 || dev == null) return;
            Guid iid = typeof(IAudioEndpointVolume).GUID;
            object epvObj;
            if (dev.Activate(ref iid, 23, IntPtr.Zero, out epvObj) != 0 || epvObj == null) return;
            var epv = (IAudioEndpointVolume)epvObj;
            Guid g = Guid.Empty;
            epv.SetMasterVolumeLevelScalar(pct / 100f, ref g);
        }
        public static void SetMute(bool mute) {
            var enumerator = (IMMDeviceEnumerator)(new MMDeviceEnumeratorComObject());
            IMMDevice dev;
            if (enumerator.GetDefaultAudioEndpoint(0, 1, out dev) != 0 || dev == null) return;
            Guid iid = typeof(IAudioEndpointVolume).GUID;
            object epvObj;
            if (dev.Activate(ref iid, 23, IntPtr.Zero, out epvObj) != 0 || epvObj == null) return;
            var epv = (IAudioEndpointVolume)epvObj;
            Guid g = Guid.Empty;
            epv.SetMute(mute, ref g);
        }
    }
}
'@`

	switch action {
	case "get_volume", "volume", "get":
		psCmd := fmt.Sprintf(`
			$src = %s
			if (-not ([System.Management.Automation.PSTypeName]'OzyAudio.MasterAudio').Type) { Add-Type -TypeDefinition $src -Language CSharp }
			$vol = [Math]::Round([OzyAudio.MasterAudio]::GetVolume(), 1)
			$mute = [OzyAudio.MasterAudio]::GetMute()
			Write-Output "Volumen maestro: $vol%% | Silencio (Mute): $(if ($mute) { 'ACTIVADO (Silenciado)' } else { 'DESACTIVADO (Con sonido)' })"
		`, coreAudioDef)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error obteniendo volumen de audio: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "set_volume", "set_vol":
		if params.Level == nil {
			return "Debes indicar el nivel de volumen ('level', entre 0 y 100).", false
		}
		lvl := *params.Level
		if lvl < 0 {
			lvl = 0
		}
		if lvl > 100 {
			lvl = 100
		}
		psCmd := fmt.Sprintf(`
			$src = %s
			if (-not ([System.Management.Automation.PSTypeName]'OzyAudio.MasterAudio').Type) { Add-Type -TypeDefinition $src -Language CSharp }
			[OzyAudio.MasterAudio]::SetVolume(%f)
			$vol = [Math]::Round([OzyAudio.MasterAudio]::GetVolume(), 1)
			Write-Output "Volumen maestro ajustado exitosamente al $vol%%."
		`, coreAudioDef, lvl)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error ajustando volumen de audio: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "mute", "silence":
		muteVal := true
		if params.Mute != nil {
			muteVal = *params.Mute
		}
		psCmd := fmt.Sprintf(`
			$src = %s
			if (-not ([System.Management.Automation.PSTypeName]'OzyAudio.MasterAudio').Type) { Add-Type -TypeDefinition $src -Language CSharp }
			[OzyAudio.MasterAudio]::SetMute($%t)
			Write-Output $(if ($%t) { "Audio maestro silenciado (Mute ACTIVADO)." } else { "Audio maestro reactivado (Mute DESACTIVADO)." })
		`, coreAudioDef, muteVal, muteVal)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error configurando mute: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "unmute":
		psCmd := fmt.Sprintf(`
			$src = %s
			if (-not ([System.Management.Automation.PSTypeName]'OzyAudio.MasterAudio').Type) { Add-Type -TypeDefinition $src -Language CSharp }
			[OzyAudio.MasterAudio]::SetMute($false)
			Write-Output "Audio maestro reactivado (Mute DESACTIVADO)."
		`, coreAudioDef)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error reactivando sonido: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

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
			badge := "[ALTAVOZ]"
			lower := strings.ToLower(name)
			if strings.Contains(lower, "auricular") || strings.Contains(lower, "headphone") || strings.Contains(lower, "buds") {
				badge = "[AURICULAR]"
			} else if strings.Contains(lower, "fxsound") {
				badge = "[VIRTUAL]"
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
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'list', 'set', 'get_volume', 'set_volume' o 'mute'.", action), false
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

// execOSPowerProfile inspecciona y ajusta planes de energía de Windows y brillo de pantalla
func execOSPowerProfile(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action     string `json:"action"`     // status, set_plan, set_brightness
		Plan       string `json:"plan"`       // balanced, high_performance, power_saver
		Brightness *int   `json:"brightness"` // 0..100
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" {
		if params.Brightness != nil {
			action = "set_brightness"
		} else if params.Plan != "" {
			action = "set_plan"
		} else {
			action = "status"
		}
	}

	switch action {
	case "status", "list":
		psCmd := `
			$active = powercfg /getactivescheme
			$planName = if ($active -match '\((.*)\)') { $matches[1] } else { $active }
			$b = Get-CimInstance -Namespace root/wmi -ClassName WmiMonitorBrightness -ErrorAction SilentlyContinue | Select-Object -ExpandProperty CurrentBrightness -First 1
			$bStr = if ($b -ne $null) { "$b%" } else { "No regulable o externo" }
			$bat = Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue | Select-Object -First 1
			$pwr = if ($bat) { if ($bat.BatteryStatus -eq 2) { "Conectado a CA" } else { "Batería ($($bat.EstimatedChargeRemaining)%)" } } else { "Alimentación CA (Desktop)" }
			Write-Output "=== PERFIL DE ENERGÍA Y PANTALLA ==="
			Write-Output "Plan de energía activo: $planName"
			Write-Output "Brillo de pantalla:     $bStr"
			Write-Output "Fuente de poder:        $pwr"
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando perfil de energía: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "set_plan":
		plan := strings.ToLower(strings.TrimSpace(params.Plan))
		guidMap := map[string]string{
			"balanced":         "381b4222-f694-41f0-9685-ff5bb260df2e",
			"equilibrado":      "381b4222-f694-41f0-9685-ff5bb260df2e",
			"high_performance": "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c",
			"alto_rendimiento": "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c",
			"power_saver":      "a1841308-3541-4fab-bc81-f71556f20b4a",
			"economizador":     "a1841308-3541-4fab-bc81-f71556f20b4a",
			"ahorro":           "a1841308-3541-4fab-bc81-f71556f20b4a",
		}
		guid, ok := guidMap[plan]
		if !ok {
			return fmt.Sprintf("Plan desconocido: '%s'. Opciones: 'balanced' (equilibrado), 'high_performance' (alto rendimiento), 'power_saver' (economizador).", plan), false
		}
		out, err := exec.CommandContext(ctx, "powercfg", "/setactive", guid).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error cambiando plan de energía: %s", strings.TrimSpace(string(out))), false
		}
		return fmt.Sprintf("Plan de energía cambiado exitosamente a: %s (GUID: %s).", plan, guid), true

	case "set_brightness":
		if params.Brightness == nil {
			return "Debes indicar el porcentaje de brillo ('brightness', de 0 a 100).", false
		}
		b := *params.Brightness
		if b < 0 {
			b = 0
		}
		if b > 100 {
			b = 100
		}
		psCmd := fmt.Sprintf(`
			$m = Get-CimInstance -Namespace root/wmi -ClassName WmiMonitorBrightnessMethods -ErrorAction SilentlyContinue
			if ($m) {
				Invoke-CimMethod -InputObject $m -MethodName WmiSetBrightness -Arguments @{Timeout = 1; Brightness = %d} | Out-Null
				Write-Output "Brillo de pantalla ajustado exitosamente al %d%%."
			} else {
				Write-Output "El control de brillo por software no está soportado en este monitor o adaptador."
			}
		`, b, b)
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error ajustando brillo: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción de energía desconocida: '%s'. Usa 'status', 'set_plan' o 'set_brightness'.", action), false
	}
}

// execOSToastNotify emite una notificación emergente Toast nativa en Windows 10/11
func execOSToastNotify(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	title := strings.TrimSpace(params.Title)
	if title == "" {
		title = "OzyAssist"
	}
	message := strings.TrimSpace(params.Message)
	if message == "" {
		return "Debes proporcionar un 'message' para la notificación.", false
	}

	escTitle := strings.ReplaceAll(strings.ReplaceAll(title, "'", "''"), "`", "")
	escMessage := strings.ReplaceAll(strings.ReplaceAll(message, "'", "''"), "`", "")

	psCmd := fmt.Sprintf(`
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
		$textNodes = $template.GetElementsByTagName('text')
		$textNodes.Item(0).AppendChild($template.CreateTextNode('%s')) | Out-Null
		$textNodes.Item(1).AppendChild($template.CreateTextNode('%s')) | Out-Null
		$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
		$appId = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
		[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show($toast)
		Write-Output "OK"
	`, escTitle, escMessage)

	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error enviando notificación Toast: %s", strings.TrimSpace(string(out))), false
	}
	return fmt.Sprintf("Notificación Toast nativa enviada a Windows: '%s - %s'.", title, message), true
}
