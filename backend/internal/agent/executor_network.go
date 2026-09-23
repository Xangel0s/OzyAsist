package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ozyassist/backend/internal/providers"
)

// execOSWifiManager gestiona el estado, auditoría y escaneo de redes Wi-Fi
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
		res := strings.TrimSpace(string(out))
		if err != nil || strings.Contains(res, "permiso de ubicación") || strings.Contains(res, "requiere elevación") {
			// Fallback robusto a PowerShell para extraer SSID, adaptador, velocidad y conectividad sin requerir permisos de ubicación
			psCmd := `
				[Console]::OutputEncoding = [System.Text.Encoding]::UTF8;
				$prof = Get-NetConnectionProfile -InterfaceAlias 'Wi-Fi' -ErrorAction SilentlyContinue
				if (-not $prof) { $prof = Get-NetConnectionProfile -ErrorAction SilentlyContinue | Where-Object { $_.IPv4Connectivity -eq 'Internet' } | Select-Object -First 1 }
				$ad = Get-NetAdapter -Name 'Wi-Fi*' -ErrorAction SilentlyContinue | Select-Object -First 1
				$ip = (Get-NetIPAddress -InterfaceAlias 'Wi-Fi' -AddressFamily IPv4 -ErrorAction SilentlyContinue | Select-Object -First 1).IPAddress
				$gw = (Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Select-Object -First 1).NextHop
				$dns = (Get-DnsClientServerAddress -InterfaceAlias 'Wi-Fi' -AddressFamily IPv4 -ErrorAction SilentlyContinue).ServerAddresses -join ', '

				Write-Output "=== ESTADO DE CONEXIÓN WI-FI ==="
				if ($prof) { Write-Output ("- Red / SSID: " + $prof.Name) }
				if ($ad) {
					Write-Output ("- Adaptador: " + $ad.InterfaceDescription)
					Write-Output ("- Estado: " + $ad.Status)
					Write-Output ("- Velocidad de enlace: " + $ad.LinkSpeed)
				}
				if ($ip) { Write-Output ("- IP Local: " + $ip) }
				if ($gw) { Write-Output ("- Puerta de enlace: " + $gw) }
				if ($dns) { Write-Output ("- Servidores DNS: " + $dns) }
				if ($prof) { Write-Output ("- Conectividad a Internet: " + $prof.IPv4Connectivity) }
			`
			psOut, psErr := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
			if psErr == nil && len(strings.TrimSpace(string(psOut))) > 0 {
				return strings.TrimSpace(string(psOut)), true
			}
			return fmt.Sprintf("Error consultando interfaz Wi-Fi: %s", res), false
		}
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
				sb.WriteString("\n[RED] " + currentSSID + "\n")
			} else if strings.HasPrefix(lTrim, "Tipo de red") || strings.HasPrefix(lTrim, "Autenticación") || strings.HasPrefix(lTrim, "Cifrado") {
				sb.WriteString("   - " + lTrim + "\n")
			}
		}
		return sb.String(), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'status' o 'networks'.", action), false
	}
}

// execOSNetworkDiagnostics evalúa latencia ICMP, puerta de enlace, vaciado de caché DNS y configuración IP
func execOSNetworkDiagnostics(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Action string `json:"action"` // test (ping), flush_dns, ip_info
		Host   string `json:"host"`
	}
	_ = json.Unmarshal(tc.Input, &params)

	action := strings.ToLower(strings.TrimSpace(params.Action))
	if action == "" || strings.Contains(action, "ping") || strings.Contains(action, "test") || strings.Contains(action, "latenc") {
		action = "test"
	} else if strings.Contains(action, "flush") || strings.Contains(action, "dns") {
		action = "flush_dns"
	} else if strings.Contains(action, "ip") || strings.Contains(action, "info") {
		action = "ip_info"
	}

	switch action {
	case "test", "ping":
		target := strings.TrimSpace(params.Host)
		if target == "" {
			target = "1.1.1.1"
		}
		escTarget := strings.ReplaceAll(target, "'", "''")

		psCmd := fmt.Sprintf(`
			[Console]::OutputEncoding = [System.Text.Encoding]::UTF8;
			$p = Test-Connection -ComputerName '%s' -Count 3 -ErrorAction SilentlyContinue
			$gw = (Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Select-Object -First 1).NextHop
			if ($p) {
				$latProp = if ($p.PSObject.Properties['Latency']) { 'Latency' } else { 'ResponseTime' }
				$avgLat = [Math]::Round(($p | Measure-Object -Property $latProp -Average).Average, 1)
				$loss = 3 - ($p | Measure-Object).Count
				Write-Output "=== DIAGNÓSTICO DE RED Y LATENCIA ==="
				Write-Output "Destino:           %s"
				Write-Output "Latencia media:    $avgLat ms"
				Write-Output "Paquetes perdidos: $loss de 3"
				Write-Output "Puerta de enlace:  $gw"
				Write-Output "Estado: [OPERATIVO] Conectividad a internet activa."
			} else {
				Write-Output "Estado: [ERROR] No se pudo establecer conexión con '%s'. Puerta de enlace: $gw. Posible corte de red o bloqueo ICMP."
			}
		`, escTarget, escTarget, escTarget)

		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error diagnosticando red: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	case "flush_dns":
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", "Clear-DnsClientCache").CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error vaciando caché DNS: %s", strings.TrimSpace(string(out))), false
		}
		return "Caché DNS de Windows vaciada exitosamente (Clear-DnsClientCache).", true

	case "ip_info":
		psCmd := `
			$ips = Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue | Where-Object { $_.InterfaceAlias -match 'Wi-Fi|Ethernet' -and $_.IPAddress -notmatch '^169\.' } | Select-Object InterfaceAlias, IPAddress, PrefixLength
			$gw = (Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Select-Object -First 1).NextHop
			$dns = (Get-DnsClientServerAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue | Where-Object { $_.ServerAddresses.Count -gt 0 } | Select-Object -First 1).ServerAddresses -join ', '
			Write-Output "=== CONFIGURACIÓN DE RED LOCAL ==="
			foreach ($i in $ips) {
				Write-Output " - Interfaz: $($i.InterfaceAlias) | IPv4: $($i.IPAddress)/$($i.PrefixLength)"
			}
			Write-Output "Puerta de enlace predeterminada: $gw"
			Write-Output "Servidores DNS: $dns"
		`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error consultando IPs: %s", strings.TrimSpace(string(out))), false
		}
		return strings.TrimSpace(string(out)), true

	default:
		return fmt.Sprintf("Acción desconocida: '%s'. Usa 'test', 'flush_dns' o 'ip_info'.", action), false
	}
}
