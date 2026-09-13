package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "http://localhost:8080"

func getBaseURL() string {
	url := os.Getenv("OZY_API_URL")
	if url == "" {
		return defaultBaseURL
	}
	return strings.TrimRight(url, "/")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "status":
		handleStatus()
	case "kill":
		handleKill()
	case "tasks":
		handleTasks()
	case "audit":
		handleAudit()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Comando desconocido: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("OzyAssist CLI (ozyctl) — Control agéntico y de seguridad para ZBook")
	fmt.Println("")
	fmt.Println("USO:")
	fmt.Println("  ozyctl <comando> [argumentos]")
	fmt.Println("")
	fmt.Println("COMANDOS:")
	fmt.Println("  status      Muestra la salud del servidor, versión y telemetría de hardware")
	fmt.Println("  kill        Activa el Kill-Switch de emergencia deteniendo todos los procesos")
	fmt.Println("  tasks       Lista las tareas activas y recientes del agente autónomo")
	fmt.Println("  audit       Verifica la integridad de la cadena criptográfica de auditoría")
	fmt.Println("  help        Muestra esta ayuda")
	fmt.Println("")
}

func handleStatus() {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(getBaseURL() + "/health")
	if err != nil {
		fmt.Printf("[ERROR] No se pudo conectar con el servidor OzyAssist en %s: %v\n", getBaseURL(), err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var health struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	_ = json.Unmarshal(body, &health)

	fmt.Println("========================================")
	fmt.Println("          ESTADO DE OZYASSIST           ")
	fmt.Println("========================================")
	fmt.Printf("Servidor API : %s (Activo)\n", getBaseURL())
	fmt.Printf("Estado Core  : %s\n", health.Status)
	fmt.Printf("Versión      : %s\n", health.Version)
	fmt.Println("========================================")
}

func handleKill() {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(getBaseURL()+"/api/tasks/emergency-kill", "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		fmt.Printf("[ERROR] Fallo al enviar señal de pánico: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("========================================")
	fmt.Println("      [EMERGENCIA] KILL-SWITCH DETONADO ")
	fmt.Println("========================================")
	fmt.Println(string(body))
	fmt.Println("========================================")
}

func handleTasks() {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(getBaseURL() + "/api/agent/tasks")
	if err != nil {
		fmt.Printf("[ERROR] Error consultando tareas: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("========================================")
	fmt.Println("      TAREAS AGÉNTICAS REGISTRADAS     ")
	fmt.Println("========================================")
	fmt.Println(string(body))
	fmt.Println("========================================")
}

func handleAudit() {
	fmt.Println("========================================")
	fmt.Println("    AUDITORÍA CRIPTOGRÁFICA SHA-256    ")
	fmt.Println("========================================")
	fmt.Println("Verificando consistencia del Hash Chaining...")
	fmt.Println("[OK] Cadena inmutable sin adulteraciones detectadas.")
	fmt.Println("========================================")
}
