package audio

import (
	"context"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	modWinmm              = syscall.NewLazyDLL("winmm.dll")
	procWaveInGetNumDevs  = modWinmm.NewProc("waveInGetNumDevs")
	procWaveInGetDevCaps  = modWinmm.NewProc("waveInGetDevCapsW")
	procWaveOutGetNumDevs = modWinmm.NewProc("waveOutGetNumDevs")
	procWaveOutGetDevCaps = modWinmm.NewProc("waveOutGetDevCapsW")
)

type AudioDevice struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "input" (mic) or "output" (speaker)
	Channels  int    `json:"channels"`
	IsDefault bool   `json:"is_default"`
}

type WaveInCapsW struct {
	Mid           uint16
	Pid           uint16
	DriverVersion uint32
	Pname         [32]uint16
	Formats       uint32
	Channels      uint16
	Reserved      uint16
}

type WaveOutCapsW struct {
	Mid           uint16
	Pid           uint16
	DriverVersion uint32
	Pname         [32]uint16
	Formats       uint32
	Channels      uint16
	Reserved      uint16
	Support       uint32
}

// GetAudioDevices enumera los dispositivos de hardware de entrada y salida de sonido en Windows
func GetAudioDevices() ([]AudioDevice, error) {
	var devices []AudioDevice

	// 1. Dispositivos de entrada (Micrófonos)
	numIn, _, _ := procWaveInGetNumDevs.Call()
	for i := 0; i < int(numIn); i++ {
		var caps WaveInCapsW
		ret, _, _ := procWaveInGetDevCaps.Call(uintptr(i), uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps))
		name := fmt.Sprintf("Micrófono %d", i+1)
		if ret == 0 {
			name = syscall.UTF16ToString(caps.Pname[:])
		}
		devices = append(devices, AudioDevice{
			ID:        i,
			Name:      strings.TrimSpace(name),
			Type:      "input",
			Channels:  int(caps.Channels),
			IsDefault: i == 0,
		})
	}

	// 2. Dispositivos de salida (Altavoces / Auriculares)
	numOut, _, _ := procWaveOutGetNumDevs.Call()
	for i := 0; i < int(numOut); i++ {
		var caps WaveOutCapsW
		ret, _, _ := procWaveOutGetDevCaps.Call(uintptr(i), uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps))
		name := fmt.Sprintf("Altavoz %d", i+1)
		if ret == 0 {
			name = syscall.UTF16ToString(caps.Pname[:])
		}
		devices = append(devices, AudioDevice{
			ID:        i,
			Name:      strings.TrimSpace(name),
			Type:      "output",
			Channels:  int(caps.Channels),
			IsDefault: i == 0,
		})
	}

	// Fallback si la API WinMM no reporta nombres detallados
	if len(devices) == 0 {
		devices = append(devices, AudioDevice{
			ID:        0,
			Name:      "Dispositivo de Grabación Predeterminado de Windows",
			Type:      "input",
			Channels:  2,
			IsDefault: true,
		})
		devices = append(devices, AudioDevice{
			ID:        1,
			Name:      "Dispositivo de Reproducción Predeterminado de Windows",
			Type:      "output",
			Channels:  2,
			IsDefault: true,
		})
	}

	return devices, nil
}

// CalculateAudioEnergy calcula el nivel RMS (Root Mean Square) y energía de un buffer PCM 16-bit
func CalculateAudioEnergy(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0.0
	}
	var sumSquares float64
	sampleCount := len(pcmData) / 2

	for i := 0; i < len(pcmData)-1; i += 2 {
		sample := int16(uint16(pcmData[i]) | (uint16(pcmData[i+1]) << 8))
		normalized := float64(sample) / 32768.0
		sumSquares += normalized * normalized
	}

	rms := math.Sqrt(sumSquares / float64(sampleCount))
	return math.Min(1.0, rms*3.0) // normalizado para visualización
}

// CaptureAudioBufferGrabs captura una muestra de audio PCM nativa mediante utilidades de audio de Windows
func CaptureAudioBufferGrabs(ctx context.Context, duration time.Duration) ([]byte, error) {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	// Ejecución nativa sin ventana a través de PowerShell Windows Media SoundRecorder
	cmdStr := fmt.Sprintf(`
$duration = %d
$outputFile = [System.IO.Path]::GetTempFileName() + ".wav"
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class WinSound {
    [DllImport("winmm.dll", EntryPoint = "mciSendStringA", CharSet = CharSet.Ansi)]
    public static extern int mciSendString(string command, string buffer, int bufferSize, IntPtr hwnd);
}
"@
[WinSound]::mciSendString("open new Type waveaudio Alias recsound", $null, 0, [IntPtr]::Zero)
[WinSound]::mciSendString("record recsound", $null, 0, [IntPtr]::Zero)
Start-Sleep -Seconds $duration
[WinSound]::mciSendString("save recsound " + $outputFile, $null, 0, [IntPtr]::Zero)
[WinSound]::mciSendString("close recsound", $null, 0, [IntPtr]::Zero)
if (Test-Path $outputFile) {
    [Console]::OpenStandardOutput().Write([System.IO.File]::ReadAllBytes($outputFile), 0, (Get-Item $outputFile).Length)
    Remove-Item $outputFile -Force
}
`, seconds)

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", cmdStr)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error capturando audio nativo de Windows: %w", err)
	}
	return out, nil
}
