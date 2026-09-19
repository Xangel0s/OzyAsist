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

// execOSAudioDevice lista o conmuta dispositivos de audio y controla volumen y mute en Windows
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
			Write-Output "🔊 Volumen maestro: $vol%% | Silencio (Mute): $(if ($mute) { 'ACTIVADO (Silenciado)' } else { 'DESACTIVADO (Con sonido)' })"
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
			Write-Output "🔊 Volumen maestro ajustado exitosamente al $vol%%."
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
			Write-Output $(if ($%t) { "🔇 Audio maestro silenciado (Mute ACTIVADO)." } else { "🔊 Audio maestro reactivado (Mute DESACTIVADO)." })
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
			Write-Output "🔊 Audio maestro reactivado (Mute DESACTIVADO)."
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
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'list', 'set', 'get_volume', 'set_volume' o 'mute'.", action), false
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

// execOSHardwareInspector inspecciona hardware físico, puertos USB, estado de periféricos en uso (cámara/mic) y telemetría de sensores
func execOSHardwareInspector(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // devices (usb), in_use (privacy), telemetry (system, sensors)
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" || strings.Contains(action, "usb") || strings.Contains(action, "device") || strings.Contains(action, "periferic") || strings.Contains(action, "puerto") {
		action = "devices"
	} else if strings.Contains(action, "use") || strings.Contains(action, "uso") || strings.Contains(action, "cam") || strings.Contains(action, "mic") || strings.Contains(action, "privac") {
		action = "in_use"
	} else if strings.Contains(action, "telem") || strings.Contains(action, "syst") || strings.Contains(action, "sens") || strings.Contains(action, "temp") || strings.Contains(action, "cpu") || strings.Contains(action, "ram") || strings.Contains(action, "gpu") || strings.Contains(action, "disk") || strings.Contains(action, "bater") {
		action = "telemetry"
	}

	switch action {
	case "devices", "usb":
		psCmd := `
			$pnp = Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.Class -in @('USB', 'Camera', 'Image', 'Media', 'Bluetooth', 'Mouse', 'Keyboard', 'DiskDrive', 'Ports') } | Select-Object FriendlyName, Class, Status
			$grouped = $pnp | Group-Object Class
			$sb = "=== DISPOSITIVOS Y PUERTOS FÍSICOS CONECTADOS ===" + [Environment]::NewLine
			foreach ($g in $grouped) {
				$sb += [Environment]::NewLine + "[$($g.Name)]" + [Environment]::NewLine
				foreach ($item in $g.Group) {
					$sb += " - $($item.FriendlyName) ($($item.Status))" + [Environment]::NewLine
				}
			}
			$sb.Trim()
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando dispositivos físicos: %s", strings.TrimSpace(string(out))), false
		}
		res := strings.TrimSpace(string(out))
		if res == "" {
			return "No se encontraron dispositivos físicos activos en los buses estándar.", true
		}
		return res, true

	case "in_use", "privacy":
		psCmd := `
			$checkInUse = {
				param($capability)
				$results = @()
				$paths = @(
					"HKCU:\Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\$capability",
					"HKLM:\Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\$capability"
				)
				foreach ($p in $paths) {
					if (Test-Path $p) {
						Get-ChildItem -Path $p -Recurse -ErrorAction SilentlyContinue | ForEach-Object {
							$prop = Get-ItemProperty -Path $_.PSPath -ErrorAction SilentlyContinue
							if ($prop -and $prop.LastUsedTimeStop -ne $null) {
								$active = ($prop.LastUsedTimeStop -eq 0 -or $prop.LastUsedTimeStart -gt $prop.LastUsedTimeStop)
								$cleanName = $_.PSChildName -replace '#', '\'
								$results += [PSCustomObject]@{
									App = $cleanName
									InUse = $active
								}
							}
						}
					}
				}
				return $results
			}

			$cam = & $checkInUse "webcam"
			$camActive = $cam | Where-Object { $_.InUse -eq $true }
			$camTxt = if ($camActive) { "🔴 EN USO por: " + (($camActive | ForEach-Object { $_.App }) -join ", ") } else { "🟢 INACTIVA (Ninguna aplicación la está usando)" }

			$mic = & $checkInUse "microphone"
			$micActive = $mic | Where-Object { $_.InUse -eq $true }
			$micTxt = if ($micActive) { "🔴 EN USO por: " + (($micActive | ForEach-Object { $_.App }) -join ", ") } else { "🟢 INACTIVO (Ninguna aplicación lo está usando)" }

			Write-Output "=== ESTADO DE PRIVACIDAD Y PERIFÉRICOS ACTIVOS ==="
			Write-Output "📸 Cámara Web: $camTxt"
			Write-Output "🎙️ Micrófono:  $micTxt"
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando periféricos en uso: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "telemetry", "system", "sensors":
		psCmd := `
			# CPU
			$cpu = Get-CimInstance Win32_Processor -ErrorAction SilentlyContinue | Select-Object -First 1 Name, NumberOfCores, NumberOfLogicalProcessors
			$cpuLoad = (Get-CimInstance Win32_Processor -ErrorAction SilentlyContinue | Measure-Object -Property LoadPercentage -Average).Average
			if (-not $cpuLoad) { $cpuLoad = 0 }

			# RAM
			$os = Get-CimInstance Win32_OperatingSystem -ErrorAction SilentlyContinue
			$totalRAM_GB = [Math]::Round($os.TotalVisibleMemorySize / 1MB, 2)
			$freeRAM_GB = [Math]::Round($os.FreePhysicalMemory / 1MB, 2)
			$usedRAM_GB = [Math]::Round($totalRAM_GB - $freeRAM_GB, 2)
			$ramUsagePct = if ($totalRAM_GB -gt 0) { [Math]::Round(($usedRAM_GB / $totalRAM_GB) * 100, 1) } else { 0 }

			# Disks
			$disks = Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" -ErrorAction SilentlyContinue | ForEach-Object {
				$size = [Math]::Round($_.Size / 1GB, 1)
				$free = [Math]::Round($_.FreeSpace / 1GB, 1)
				$used = [Math]::Round($size - $free, 1)
				$pct = if ($size -gt 0) { [Math]::Round(($used / $size) * 100, 1) } else { 0 }
				" - Disco $($_.DeviceID) ($($_.VolumeName)): $free GB libres de $size GB ($pct% ocupado)"
			}

			# GPU
			$gpus = Get-CimInstance Win32_VideoController -ErrorAction SilentlyContinue | ForEach-Object {
				$vram = [Math]::Round($_.AdapterRAM / 1MB, 0)
				" - $($_.Name) ($vram MB VRAM)"
			}

			# NVIDIA Temp & Utilization
			$nvidiaInfo = ""
			if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) {
				try {
					$nv = nvidia-smi --query-gpu=temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.used --format=csv,noheader,nounits 2>$null
					$parts = $nv -split ','
					if ($parts.Count -ge 5) {
						$nvidiaInfo = "   🌡️ Temp GPU NVIDIA: $($parts[0].Trim()) °C | Carga: $($parts[1].Trim())% | VRAM Usada: $($parts[4].Trim()) / $($parts[3].Trim()) MB"
					}
				} catch {}
			}

			# Thermal ACPI
			$acpiTemp = ""
			$tz = Get-CimInstance -Namespace "root/cimv2" -ClassName "Win32_PerfFormattedData_Counters_ThermalZoneInformation" -ErrorAction SilentlyContinue | Select-Object -First 1
			if ($tz -and $tz.Temperature) {
				$celsius = [Math]::Round($tz.Temperature - 273.15, 1)
				$acpiTemp = "🌡️ Temperatura ACPI Sistema: $celsius °C"
			}

			# Battery
			$bat = Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue | Select-Object -First 1
			$batText = "No presente (Equipo de escritorio)"
			if ($bat) {
				$status = if ($bat.BatteryStatus -eq 2) { "Conectado a CA (Cargando / Completa)" } else { "Descargando" }
				$batText = "$($bat.EstimatedChargeRemaining)% ($status)"
			}

			Write-Output "=== TELEMETRÍA DE HARDWARE Y RECURSOS ==="
			if ($cpu) {
				Write-Output "⚡ CPU: $($cpu.Name)"
				Write-Output "   Núcleos: $($cpu.NumberOfCores) físicos, $($cpu.NumberOfLogicalProcessors) lógicos | Uso actual: $cpuLoad%"
			}
			Write-Output "🧠 Memoria RAM: $usedRAM_GB GB usados de $totalRAM_GB GB ($ramUsagePct% en uso, $freeRAM_GB GB disponibles)"
			Write-Output "💾 Almacenamiento:"
			$disks | ForEach-Object { Write-Output $_ }
			Write-Output "🎮 Gráficos / GPU:"
			$gpus | ForEach-Object { Write-Output $_ }
			if ($nvidiaInfo) { Write-Output $nvidiaInfo }
			if ($acpiTemp) { Write-Output "🌡️ Térmico: $acpiTemp" }
			Write-Output "🔋 Batería: $batText"
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando telemetría de hardware: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'devices', 'in_use' o 'telemetry'.", action), false
	}
}

