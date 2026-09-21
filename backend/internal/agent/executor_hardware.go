package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/system"
)

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

func execOSHardwareInspector(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // devices (usb), in_use (privacy), telemetry (system, sensors)
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if strings.Contains(action, "health") || strings.Contains(action, "salud") || strings.Contains(action, "diagnos") || strings.Contains(action, "audit") || strings.Contains(action, "smart") {
		action = "health"
	} else if action == "" || strings.Contains(action, "usb") || strings.Contains(action, "device") || strings.Contains(action, "periferic") || strings.Contains(action, "puerto") {
		action = "devices"
	} else if strings.Contains(action, "use") || strings.Contains(action, "uso") || strings.Contains(action, "cam") || strings.Contains(action, "mic") || strings.Contains(action, "privac") {
		action = "in_use"
	} else if strings.Contains(action, "telem") || strings.Contains(action, "syst") || strings.Contains(action, "sens") || strings.Contains(action, "temp") || strings.Contains(action, "cpu") || strings.Contains(action, "ram") || strings.Contains(action, "gpu") || strings.Contains(action, "disk") || strings.Contains(action, "bater") {
		action = "telemetry"
	}

	switch action {
	case "health", "salud", "diagnostico", "audit":
		psCmd := `
			# 1. SMART Disks
			$disks = Get-CimInstance -ClassName Win32_DiskDrive -ErrorAction SilentlyContinue | Select-Object Model, Status
			$badDisks = @()
			foreach ($d in $disks) {
				if ($d.Status -ne "OK") {
					$badDisks += "$($d.Model) (Estado: $($d.Status))"
				}
			}

			# 2. Storage Capacity Alerts
			$volAlerts = @()
			Get-CimInstance Win32_LogicalDisk -Filter "DriveType=3" -ErrorAction SilentlyContinue | ForEach-Object {
				$size = [Math]::Round($_.Size / 1GB, 1)
				$free = [Math]::Round($_.FreeSpace / 1GB, 1)
				if ($size -gt 0) {
					$pct = [Math]::Round((($size - $free) / $size) * 100, 1)
					if ($pct -ge 90) {
						$volAlerts += "Disco $($_.DeviceID) saturado: $free GB libres de $size GB ($pct% ocupado)"
					}
				}
			}

			# 3. Thermal Alerts
			$thermalAlerts = @()
			if (Get-Command nvidia-smi -ErrorAction SilentlyContinue) {
				try {
					$nv = nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader,nounits 2>$null
					$gpuT = [int]($nv.Trim())
					if ($gpuT -ge 85) {
						$thermalAlerts += "Temperatura de GPU elevada: $gpuT °C (riesgo de thermal throttling)"
					}
				} catch {}
			}
			$tz = Get-CimInstance -Namespace "root/cimv2" -ClassName "Win32_PerfFormattedData_Counters_ThermalZoneInformation" -ErrorAction SilentlyContinue | Select-Object -First 1
			if ($tz -and $tz.Temperature) {
				$sysT = [Math]::Round($tz.Temperature - 273.15, 1)
				if ($sysT -ge 85) {
					$thermalAlerts += "Temperatura del sistema ACPI elevada: $sysT °C"
				}
			}

			# 4. RAM Pressure
			$ramAlerts = @()
			$os = Get-CimInstance Win32_OperatingSystem -ErrorAction SilentlyContinue
			$freeRAM_MB = [Math]::Round($os.FreePhysicalMemory / 1024, 0)
			if ($freeRAM_MB -lt 1500) {
				$ramAlerts += "Memoria RAM libre baja: $freeRAM_MB MB disponibles"
			}

			# 5. Battery
			$batAlerts = @()
			$bat = Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue | Select-Object -First 1
			if ($bat -and $bat.BatteryStatus -ne 2 -and $bat.EstimatedChargeRemaining -le 20) {
				$batAlerts += "Batería baja y desconectada de la corriente: $($bat.EstimatedChargeRemaining)%"
			}

			# 6. WHEA Events
			$whea = Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName=@('Microsoft-Windows-WHEA-Logger', 'disk'); Level=2; StartTime=(Get-Date).AddDays(-3)} -MaxEvents 3 -ErrorAction SilentlyContinue

			# Formar Reporte
			$alerts = @()
			$alerts += $badDisks
			$alerts += $volAlerts
			$alerts += $thermalAlerts
			$alerts += $ramAlerts
			$alerts += $batAlerts

			$statusSymbol = if ($alerts.Count -eq 0) { "[HARDWARE SALUDABLE]" } else { "[ATENCION PREVENTIVA REQUERIDA]" }
			if ($badDisks.Count -gt 0 -or $whea) {
				$statusSymbol = "[ALERTA DE FALLO DE HARDWARE]"
			}

			Write-Output "=== AUDITORÍA Y SALUD DEL HARDWARE ==="
			Write-Output "Diagnóstico General: $statusSymbol"
			Write-Output ""
			Write-Output "1. Integridad SMART de Discos: $(if ($badDisks.Count -eq 0) { 'Todos los discos en estado OK' } else { ($badDisks -join '; ') })"
			Write-Output "2. Estado Térmico: $(if ($thermalAlerts.Count -eq 0) { 'Temperaturas dentro de los rangos seguros' } else { ($thermalAlerts -join '; ') })"
			Write-Output "3. Almacenamiento: $(if ($volAlerts.Count -eq 0) { 'Espacio suficiente en todos los volúmenes' } else { ($volAlerts -join '; ') })"
			Write-Output "4. Memoria RAM: $(if ($ramAlerts.Count -eq 0) { "RAM disponible saludable ($([Math]::Round($freeRAM_MB/1024, 2)) GB libres)" } else { ($ramAlerts -join '; ') })"
			Write-Output "5. Energía / Batería: $(if ($batAlerts.Count -eq 0) { 'Alimentación estable' } else { ($batAlerts -join '; ') })"
			if ($whea) {
				Write-Output "6. Eventos Críticos WHEA: Se detectaron $(@($whea).Count) alertas de hardware en el registro del sistema."
			} else {
				Write-Output "6. Eventos Críticos WHEA: 0 errores de arquitectura de hardware registrados."
			}

			if ($alerts.Count -gt 0) {
				Write-Output ""
				Write-Output "[RECOMENDACIONES DE OZY]"
				foreach ($a in $alerts) {
					Write-Output " - $a"
				}
			}
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error diagnosticando salud del hardware: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "devices", "usb":
		psCmd := `
			[Console]::OutputEncoding = [System.Text.Encoding]::UTF8;
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
			$camTxt = if ($camActive) { "[EN USO por: " + (($camActive | ForEach-Object { $_.App }) -join ", ") + "]" } else { "[INACTIVA] (Ninguna aplicación la está usando)" }

			$mic = & $checkInUse "microphone"
			$micActive = $mic | Where-Object { $_.InUse -eq $true }
			$micTxt = if ($micActive) { "[EN USO por: " + (($micActive | ForEach-Object { $_.App }) -join ", ") + "]" } else { "[INACTIVO] (Ninguna aplicación lo está usando)" }

			Write-Output "=== ESTADO DE PRIVACIDAD Y PERIFÉRICOS ACTIVOS ==="
			Write-Output "Cámara Web: $camTxt"
			Write-Output "Micrófono:  $micTxt"
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
						$nvidiaInfo = "   Temp GPU NVIDIA: $($parts[0].Trim()) °C | Carga: $($parts[1].Trim())% | VRAM Usada: $($parts[4].Trim()) / $($parts[3].Trim()) MB"
					}
				} catch {}
			}

			# Thermal ACPI
			$acpiTemp = ""
			$tz = Get-CimInstance -Namespace "root/cimv2" -ClassName "Win32_PerfFormattedData_Counters_ThermalZoneInformation" -ErrorAction SilentlyContinue | Select-Object -First 1
			if ($tz -and $tz.Temperature) {
				$celsius = [Math]::Round($tz.Temperature - 273.15, 1)
				$acpiTemp = "Temperatura ACPI Sistema: $celsius °C"
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
				Write-Output "CPU: $($cpu.Name)"
				Write-Output "   Núcleos: $($cpu.NumberOfCores) físicos, $($cpu.NumberOfLogicalProcessors) lógicos | Uso actual: $cpuLoad%"
			}
			Write-Output "Memoria RAM: $usedRAM_GB GB usados de $totalRAM_GB GB ($ramUsagePct% en uso, $freeRAM_GB GB disponibles)"
			Write-Output "Almacenamiento:"
			$disks | ForEach-Object { Write-Output $_ }
			Write-Output "Gráficos / GPU:"
			$gpus | ForEach-Object { Write-Output $_ }
			if ($nvidiaInfo) { Write-Output $nvidiaInfo }
			if ($acpiTemp) { Write-Output "Térmico: $acpiTemp" }
			Write-Output "Batería: $batText"
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando telemetría de hardware: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'health', 'devices', 'in_use' o 'telemetry'.", action), false
	}
}

// execOSPowerProfile consulta o cambia planes de energía de Windows y ajusta el brillo de pantalla

func execOSSmartOrganizer(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action    string `json:"action"` // duplicates, clutter
		TargetDir string `json:"target_dir"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" || strings.Contains(action, "dup") {
		action = "duplicates"
	} else if strings.Contains(action, "clutter") || strings.Contains(action, "old") || strings.Contains(action, "instal") {
		action = "clutter"
	}

	targetDir := strings.TrimSpace(params.TargetDir)
	if targetDir == "" {
		targetDir = system.ResolveUserPath("Downloads")
	} else {
		targetDir = system.ResolveUserPath(targetDir)
	}

	switch action {
	case "duplicates":
		// Agrupar archivos por tamaño primero para evitar hashear archivos innecesarios
		type fileMeta struct {
			path string
			size int64
		}
		bySize := make(map[int64][]fileMeta)

		err := filepath.Walk(targetDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				name := info.Name()
				if name != filepath.Base(targetDir) && (strings.HasPrefix(name, ".") || name == "node_modules" || name == ".git") {
					return filepath.SkipDir
				}
				return nil
			}
			if info.Size() > 50*1024 { // Archivos mayores a 50KB
				bySize[info.Size()] = append(bySize[info.Size()], fileMeta{path: p, size: info.Size()})
			}
			return nil
		})
		if err != nil {
			return fmt.Sprintf("Error examinando carpeta: %v", err), false
		}

		type dupGroup struct {
			hash  string
			size  int64
			files []string
		}
		byHash := make(map[string]*dupGroup)

		hashFile := func(path string) string {
			f, err := os.Open(path)
			if err != nil {
				return ""
			}
			defer f.Close()
			h := sha256.New()
			_, _ = io.CopyN(h, f, 2*1024*1024)
			return fmt.Sprintf("%x", h.Sum(nil))
		}

		var totalWastedBytes int64
		for _, files := range bySize {
			if len(files) < 2 {
				continue
			}
			for _, fm := range files {
				h := hashFile(fm.path)
				if h == "" {
					continue
				}
				if grp, ok := byHash[h]; ok {
					grp.files = append(grp.files, fm.path)
					totalWastedBytes += fm.size
				} else {
					byHash[h] = &dupGroup{
						hash:  h,
						size:  fm.size,
						files: []string{fm.path},
					}
				}
			}
		}

		var foundGroups []*dupGroup
		for _, g := range byHash {
			if len(g.files) > 1 {
				foundGroups = append(foundGroups, g)
			}
		}

		if len(foundGroups) == 0 {
			return fmt.Sprintf("No se encontraron archivos duplicados en '%s'.", targetDir), true
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("=== ARCHIVOS DUPLICADOS DETECTADOS EN: %s ===\n", targetDir))
		sb.WriteString(fmt.Sprintf("Espacio recuperable estimado: %.2f MB\n\n", float64(totalWastedBytes)/(1024*1024)))
		for i, g := range foundGroups {
			if i >= 10 {
				sb.WriteString(fmt.Sprintf("... y %d grupos más de duplicados.\n", len(foundGroups)-10))
				break
			}
			sb.WriteString(fmt.Sprintf("Grupo #%d (%.2f MB cada uno):\n", i+1, float64(g.size)/(1024*1024)))
			for _, f := range g.files {
				sb.WriteString(fmt.Sprintf("  - %s\n", f))
			}
		}
		return sb.String(), true

	case "clutter":
		psCmd := fmt.Sprintf(`
			$dir = '%s'
			$exts = @('.exe', '.msi', '.iso', '.zip', '.tmp')
			$cutoff = (Get-Date).AddDays(-14)
			$files = Get-ChildItem -Path $dir -File -ErrorAction SilentlyContinue | Where-Object {
				$_.Extension -in $exts -and $_.LastWriteTime -lt $cutoff
			} | Select-Object Name, Length, LastWriteTime
			if ($files) {
				$totalMB = [Math]::Round(($files | Measure-Object -Property Length -Sum).Sum / 1MB, 2)
				Write-Output "=== INSTALADORES Y ARCHIVOS HUÉRFANOS (>14 DÍAS) ==="
				Write-Output "Carpeta: $dir | Espacio recuperable: $totalMB MB"
				foreach ($f in $files | Select-Object -First 15) {
					$mb = [Math]::Round($f.Length / 1MB, 2)
					$days = [Math]::Round(((Get-Date) - $f.LastWriteTime).TotalDays, 0)
					Write-Output " - $($f.Name) ($mb MB, modificado hace $days días)"
				}
			} else {
				Write-Output "No se detectaron instaladores ni temporales antiguos en $dir."
			}
		`, strings.ReplaceAll(targetDir, "'", "''"))

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error buscando archivos obsoletos: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'duplicates' o 'clutter'.", action), false
	}
}

