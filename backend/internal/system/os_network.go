package system

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// PortBinding describe un puerto en escucha y el proceso responsable
type PortBinding struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	State       string `json:"state"`
	ProcessID   uint32 `json:"process_id"`
	ProcessName string `json:"process_name,omitempty"`
}

// InspectPorts consulta los puertos TCP en escucha en el sistema o uno específico
func InspectPorts(targetPort int) ([]PortBinding, error) {
	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error ejecutando netstat: %w", err)
	}

	var results []PortBinding
	scanner := bufio.NewScanner(bytes.NewReader(out))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(strings.ToUpper(line), "TCP") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		// Formato: TCP  127.0.0.1:8080  0.0.0.0:0  LISTENING  1234
		localAddr := fields[1]
		state := fields[len(fields)-2]
		pidStr := fields[len(fields)-1]

		if !strings.EqualFold(state, "LISTENING") {
			continue
		}

		colonIdx := strings.LastIndex(localAddr, ":")
		if colonIdx == -1 {
			continue
		}
		portStr := localAddr[colonIdx+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		if targetPort > 0 && port != targetPort {
			continue
		}

		pidVal, _ := strconv.ParseUint(pidStr, 10, 32)
		pid := uint32(pidVal)
		procName := getProcessName(pid)

		results = append(results, PortBinding{
			Port:        port,
			Protocol:    "TCP",
			State:       state,
			ProcessID:   pid,
			ProcessName: procName,
		})
	}

	return results, nil
}

// KillPortProcess finaliza el proceso que está ocupando un puerto específico
func KillPortProcess(ctx context.Context, port int) (string, error) {
	bindings, err := InspectPorts(port)
	if err != nil {
		return "", err
	}
	if len(bindings) == 0 {
		return fmt.Sprintf("El puerto %d no está en uso por ningún proceso.", port), nil
	}

	var killed []string
	for _, b := range bindings {
		if b.ProcessID == 0 {
			continue
		}
		// Finalizar proceso con taskkill
		_ = exec.Command("taskkill", "/PID", fmt.Sprintf("%d", b.ProcessID), "/F", "/T").Run()
		killed = append(killed, fmt.Sprintf("PID %d (%s) en puerto %d", b.ProcessID, b.ProcessName, b.Port))
	}

	if len(killed) == 0 {
		return fmt.Sprintf("No se pudo identificar un PID válido para finalizar en el puerto %d", port), nil
	}
	return fmt.Sprintf("Procesos finalizados con éxito: %s", strings.Join(killed, ", ")), nil
}
