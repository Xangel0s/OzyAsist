package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/api"
	"github.com/ozyassist/backend/internal/daemon"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/db/models"
	"github.com/ozyassist/backend/internal/mcp"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/tui"
	"github.com/ozyassist/backend/internal/voice"
)

const banner = `
  ██████╗ ███████╗██╗   ██╗ █████╗ ███████╗███████╗██╗███████╗████████╗
 ██╔═══██╗╚══███╔╝╚██╗ ██╔╝██╔══██╗██╔════╝██╔════╝██║██╔════╝╚══██╔══╝
 ██║   ██║  ███╔╝  ╚████╔╝ ███████║███████╗███████╗██║███████╗   ██║   
 ██║   ██║ ███╔╝    ╚██╔╝  ██╔══██║╚════██║╚════██║██║╚════██║   ██║   
 ╚██████╔╝███████╗   ██║   ██║  ██║███████║███████║██║███████║   ██║   
  ╚═════╝ ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝╚═╝╚══════╝   ╚═╝   
             ⚡ TUI & OS Control Agent (Hermes Mode)
`

func printUsage() {
	fmt.Print(banner)
	fmt.Print(`Uso:
  ozy [comando] [argumentos]

Comandos:
  ozy (o ozy run)         Inicia la TUI interactiva con streaming y control del SO
  ozy exec "<orden>"      Ejecuta una orden de forma directa y autónoma (headless)
  ozy voice               Modo consola de voz continua ("Hey Ozy")
  ozy daemon              Inicia el servicio en segundo plano (System Tray & Hotkeys)
  ozy serve               Inicia el servidor API HTTP/WebSocket tradicional
  ozy help                Muestra esta ayuda

Ejemplos:
  ozy exec "ordena mis carpetas"
  ozy exec "muestra telemetría del sistema"
  ozy run
`)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	args := os.Args[1:]
	cmd := "run"
	if len(args) > 0 {
		cmd = strings.ToLower(args[0])
	}

	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		printUsage()
		return
	}

	prov, chat := initCore()
	defer mcp.DefaultRegistry.Shutdown()

	switch cmd {
	case "run", "tui":
		runTUI(prov, chat)

	case "exec":
		if len(args) < 2 {
			fmt.Println("Error: Debes especificar una orden. Ej: ozy exec \"ordena mis carpetas\"")
			os.Exit(1)
		}
		prompt := strings.Join(args[1:], " ")
		runExec(prov, chat, prompt)

	case "voice":
		runVoiceConsole(prov, chat)

	case "daemon":
		runDaemon()

	case "serve":
		runServer()

	default:
		// Si el usuario pasa directamente un prompt entre comillas (ej: ozy "ordena mis carpetas")
		if !strings.HasPrefix(cmd, "-") {
			prompt := strings.Join(args, " ")
			runExec(prov, chat, prompt)
		} else {
			printUsage()
		}
	}
}

func initCore() (providers.Provider, *models.Chat) {
	dbPath := os.Getenv("OZY_DB_PATH")
	if dbPath == "" {
		cwd, _ := os.Getwd()
		candidate := filepath.Join(cwd, "data", "ozyassist.db")
		if _, err := os.Stat(candidate); err == nil {
			dbPath = candidate
		} else {
			dbPath = filepath.Join(cwd, "backend", "data", "ozyassist.db")
		}
	}

	if err := db.Init(dbPath); err != nil {
		log.Printf("[Advertencia] Base de datos en memoria/local temporal: %v", err)
	}
	_ = db.EnsureDefaultUser()

	providers.InitProviders()
	_ = mcp.DefaultRegistry.InitFromConfigFile(context.Background(), "")

	prov := providers.GetDefaultOrFirstProvider()

	var activeChat *models.Chat
	chats, err := db.ListChats()
	if err == nil && len(chats) > 0 {
		activeChat = &chats[0]
	} else {
		activeChat = &models.Chat{
			ID:        uuid.NewString(),
			UserID:    db.DefaultUserID(),
			Name:      "TUI Session",
			Mode:      "chat",
			CreatedAt: time.Now(),
		}
		_ = db.CreateChat(activeChat)
	}

	if activeChat != nil && prov != nil {
		oldProv := activeChat.Provider
		activeChat.Provider = prov.Name()
		if activeChat.Model == "" || oldProv != prov.Name() {
			if prov.Name() == "openrouter" {
				activeChat.Model = "deepseek/deepseek-chat"
			} else if len(prov.Models()) > 0 {
				activeChat.Model = prov.Models()[0]
			}
		}
		_ = db.UpdateChat(activeChat)
	}

	return prov, activeChat
}

func runTUI(prov providers.Provider, chat *models.Chat) {
	// Redirigir logs a ozy.log para evitar corromper la pantalla de Bubble Tea
	logFile, err := os.OpenFile("ozy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	} else {
		log.SetOutput(io.Discard)
	}

	cfg := voice.AutoDetectConfig()
	voiceReady := cfg.HasSTT()

	// Crear motor de escucha nativa de Wake Word unificado
	voiceEngine := voice.NewNativeVoiceEngine(cfg, func(ctx context.Context, command string, onDelta func(string), onComplete func(string)) {
		tui.SendVoiceCommand(command)
	})
	voiceEngine.OnStateChange = func(s voice.VoiceEngineState, desc string) {
		tui.SendVoiceStatus(voiceEngine.IsRunning(), desc)
	}

	m := tui.InitialModel(prov, chat, voiceReady)
	p := tea.NewProgram(m, tea.WithAltScreen())
	tui.SetProgram(p)

	engineCtx, cancelEngine := context.WithCancel(context.Background())
	defer cancelEngine()

	adapter := &voiceAdapter{
		engine:    voiceEngine,
		engineCtx: engineCtx,
	}
	tui.SetVoiceController(adapter)

	// Iniciar motor en paralelo únicamente si STT está listo
	if voiceReady {
		_ = voiceEngine.Start(engineCtx)
	}
	defer voiceEngine.Stop()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error ejecutando TUI: %v\n", err)
		os.Exit(1)
	}
}

type voiceAdapter struct {
	engine    *voice.NativeVoiceEngine
	engineCtx context.Context
}

func (a *voiceAdapter) HasSTT() bool {
	return a.engine.HasSTT()
}

func (a *voiceAdapter) IsRunning() bool {
	return a.engine.IsRunning()
}

func (a *voiceAdapter) Start() error {
	return a.engine.Start(a.engineCtx)
}

func (a *voiceAdapter) Stop() {
	a.engine.Stop()
}

func (a *voiceAdapter) UpdateKey(provider, key string) error {
	cfg := voice.Config{}
	if provider == "groq" {
		cfg.GroqAPIKey = key
	} else if provider == "openai" {
		cfg.OpenAIAPIKey = key
	}
	a.engine.UpdateConfig(cfg)
	return nil
}

func runExec(prov providers.Provider, _ *models.Chat, prompt string) {
	fmt.Printf("\n⚡ OZYASSIST EXEC: \"%s\"\n\n", prompt)

	chatName := prompt
	if len(chatName) > 40 {
		chatName = chatName[:40] + "..."
	}
	execChat := &models.Chat{
		ID:        uuid.NewString(),
		UserID:    db.DefaultUserID(),
		Name:      "CLI: " + chatName,
		Mode:      "chat",
		Provider:  prov.Name(),
		CreatedAt: time.Now(),
	}
	if prov.Name() == "openrouter" {
		execChat.Model = "deepseek/deepseek-chat"
	} else if len(prov.Models()) > 0 {
		execChat.Model = prov.Models()[0]
	}
	_ = db.CreateChat(execChat)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	params := agent.AgentLoopParams{
		Provider:        prov,
		Chat:            execChat,
		UserMessage:     prompt,
		PermissionLevel: "autonomous",
		VoiceMode:       false,
		Emit: func(evt agent.AgentEvent) {
			switch evt.Type {
			case "message:delta":
				fmt.Print(evt.Content)

			case "tool:call":
				fmt.Printf("\n⚙️  [HERRAMIENTA] %s\n   Parámetros: %s\n", evt.ToolName, evt.ToolInput)

			case "tool:result":
				status := "✓ Éxito"
				if !evt.ToolSuccess {
					status = "✗ Error"
				}
				fmt.Printf("   Resultado (%s): %s\n\n", status, strings.TrimSpace(evt.ToolOutput))

			case "agent:completed":
				fmt.Println("\n\n✓ Tarea completada con éxito.")
				wg.Done()

			case "error":
				fmt.Printf("\n❌ Error: %s\n", evt.Error)
				wg.Done()
			}
		},
	}

	_ = agent.StartAgentLoop(ctx, params)
	wg.Wait()
}

func runVoiceConsole(prov providers.Provider, chat *models.Chat) {
	fmt.Print(banner)
	cfg := voice.AutoDetectConfig()
	if !cfg.HasSTT() {
		fmt.Println("\n❌ Error: No hay motor de transcripción STT configurado.")
		fmt.Println("👉 Para usar el modo voz autónomo, configura una clave gratuita de Groq o de OpenAI en tu .env:")
		fmt.Println("   GROQ_API_KEY=tu_clave_de_groq  (Consigue una gratis en https://console.groq.com/keys)")
		fmt.Println("   o OPENAI_API_KEY=tu_clave_de_openai")
		fmt.Println("\nTambién puedes iniciar 'ozy' y escribir: /key groq <tu-clave>")
		os.Exit(1)
	}

	fmt.Println("🎙️ Modo Voz Autónomo Activado.")
	fmt.Println("Di 'Hey Ozy' o 'Hey Osi' seguido de tu orden (ej: 'Hey Ozy ordena mis carpetas')...")
	fmt.Println("Presiona Ctrl+C para salir.")

	engine := voice.NewNativeVoiceEngine(cfg, func(ctx context.Context, command string, onDelta func(string), onComplete func(string)) {
		fmt.Printf("\n[VOZ RECONOCIDA]: %s\n", command)
		params := agent.AgentLoopParams{
			Provider:        prov,
			Chat:            chat,
			UserMessage:     command,
			PermissionLevel: "autonomous",
			VoiceMode:       true,
			Emit: func(evt agent.AgentEvent) {
				switch evt.Type {
				case "message:delta":
					fmt.Print(evt.Content)
					onDelta(evt.Content)
				case "tool:call":
					fmt.Printf("\n⚙️  [HERRAMIENTA] %s... ", evt.ToolName)
				case "tool:result":
					fmt.Println("✓")
				case "agent:completed":
					fmt.Println()
					onComplete(evt.Content)
				}
			},
		}
		_ = agent.StartAgentLoop(ctx, params)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Error iniciando motor de voz: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("\nCerrando modo voz...")
	engine.Stop()
}

func runDaemon() {
	fmt.Println("Iniciando OzyAssist en modo Daemon...")
	onWake := func() {
		log.Println("[Daemon] ¡Ozy ha despertado vía demonio nativo!")
	}
	d := daemon.NewDaemon(os.Getenv("PICOVOICE_ACCESS_KEY"), onWake)
	d.Start()
}

func runServer() {
	router := api.NewRouter()
	port := os.Getenv("OZY_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Iniciando servidor API en puerto %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}
