package agent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// OSHardwareControl ajusta configuraciones físicas del sistema como brillo y volumen.
func OSHardwareControl(ctx context.Context, setting string, value int) (string, error) {
	setting = strings.ToLower(strings.TrimSpace(setting))
	
	if setting == "brightness" || setting == "brillo" {
		if value < 0 { value = 0 }
		if value > 100 { value = 100 }
		
		cmdStr := fmt.Sprintf(`(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1, %d)`, value)
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", cmdStr)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("error ajustando brillo: %v", err)
		}
		return fmt.Sprintf("Brillo ajustado al %d%%", value), nil
	}
	
	if setting == "volume" || setting == "volumen" {
		if value < 0 { value = 0 }
		if value > 100 { value = 100 }
		
		// Script C# embebido para controlar el volumen vía Windows CoreAudio API
		cmdStr := fmt.Sprintf(`
Add-Type -TypeDefinition @"
using System.Runtime.InteropServices;
[Guid("5CDF2C82-841E-4546-9722-0CF74078229A"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IAudioEndpointVolume {
    int f(); int g(); int h(); int i();
    int SetMasterVolumeLevelScalar(float fLevel, System.Guid pguidEventContext);
    int j(); int GetMasterVolumeLevelScalar(out float pfLevel);
}
[Guid("D666063F-1587-4E43-81F1-B948E807363F"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDevice {
    int Activate(ref System.Guid id, int clsCtx, int activationParams, out IAudioEndpointVolume aev);
}
[Guid("A95664D2-9614-4F35-A746-DE8DB63617E6"), InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
interface IMMDeviceEnumerator {
    int f(); int GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice endpoint);
}
[ComImport, Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")] class MMDeviceEnumeratorComObject { }
public class Audio {
    public static void SetVolume(float level) {
        var enumerator = (IMMDeviceEnumerator)(new MMDeviceEnumeratorComObject());
        enumerator.GetDefaultAudioEndpoint(0, 1, out var device);
        var aevGuid = typeof(IAudioEndpointVolume).GUID;
        device.Activate(ref aevGuid, 1, 0, out var aev);
        aev.SetMasterVolumeLevelScalar(level, System.Guid.Empty);
    }
}
"@
[Audio]::SetVolume(%f)
`, float32(value)/100.0)
		
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", cmdStr)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("error ajustando volumen: %v", err)
		}
		return fmt.Sprintf("Volumen del sistema ajustado al %d%%", value), nil
	}
	
	return "", fmt.Errorf("configuración de hardware desconocida: %s", setting)
}

// OSPowerState cambia el estado de energía de la computadora.
func OSPowerState(ctx context.Context, state string) (string, error) {
	state = strings.ToLower(strings.TrimSpace(state))
	
	var cmd *exec.Cmd
	switch state {
	case "shutdown", "apagar":
		cmd = exec.CommandContext(ctx, "shutdown", "/s", "/t", "0")
	case "restart", "reiniciar":
		cmd = exec.CommandContext(ctx, "shutdown", "/r", "/t", "0")
	case "sleep", "suspender":
		// En Windows, esto requiere hibernación desactivada para que suspenda de verdad, o llamadas directas
		cmd = exec.CommandContext(ctx, "rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
	default:
		return "", fmt.Errorf("estado de energía desconocido: %s", state)
	}
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error cambiando estado de energía a %s: %v", state, err)
	}
	return fmt.Sprintf("Ejecutando comando de %s...", state), nil
}

// OSLaunchApp lanza un programa o binario en el sistema.
func OSLaunchApp(ctx context.Context, appName string) (string, error) {
	cmdStr := fmt.Sprintf(`Start-Process "%s"`, appName)
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", cmdStr)
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error lanzando aplicación %s: %v", appName, err)
	}
	return fmt.Sprintf("Aplicación '%s' lanzada con éxito.", appName), nil
}
