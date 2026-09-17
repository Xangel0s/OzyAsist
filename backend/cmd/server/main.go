package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ozyassist/backend/internal/agent"
	"github.com/ozyassist/backend/internal/api"
	"github.com/ozyassist/backend/internal/api/handlers"
	"github.com/ozyassist/backend/internal/api/ws"
	"github.com/ozyassist/backend/internal/browser"
	"github.com/ozyassist/backend/internal/connectors/google"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/mcp"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/search"
	"github.com/ozyassist/backend/internal/security"
	"github.com/ozyassist/backend/internal/system"
	"github.com/ozyassist/backend/internal/taskengine"
	"github.com/ozyassist/backend/internal/vision"
	"github.com/ozyassist/backend/internal/voice"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
		log.Fatalf("Error inicializando base de datos: %v", err)
	}
	defer db.Close()

	if err := db.EnsureDefaultUser(); err != nil {
		log.Fatalf("Error creando usuario por defecto: %v", err)
	}

	providers.InitProviders()
	memory.BuildSystemIndex()
	// Inicializar Verificador y Auto-Reparación (Self-Healing)
	verifier := agent.NewVerifier()
	var defaultProv providers.Provider
	availableProv := providers.Available()
	if len(availableProv) > 0 {
		if prov, err := providers.Get(availableProv[0]); err == nil {
			defaultProv = prov
		}
	}
	healer := agent.NewHealer(defaultProv, verifier)

	// Inicializar Memoria Jerárquica Continua
	memory.InitContinuousMemory(db.DB, defaultProv)

	// Inicializar Browser CDP y Telemetría del Sistema
	cdpURL := os.Getenv("OZY_CDP_URL")
	if cdpURL == "" {
		cdpURL = "http://localhost:9222"
	}
	browserClient := browser.NewCDPClient(cdpURL)
	telemetryCollector := system.NewCollector()

	// Inicializar Conector Google Workspace
	googleClient := google.NewClient(nil)
	oauthKey := []byte(os.Getenv("OZY_OAUTH_KEY"))
	if len(oauthKey) != 32 {
		oauthKey = []byte("ozyassist_oauth_secret_key_32b!!")
	}

	// Inicializar Percepción Visual y Pipeline de Audio Local
	visionCapture := vision.NewScreenCapture()
	voicePipeline := voice.NewAudioPipeline(voice.Config{
		WakeWordBinary: os.Getenv("OZY_WAKEWORD_BIN"),
		WhisperBinary:  os.Getenv("OZY_WHISPER_BIN"),
		PiperBinary:    os.Getenv("OZY_PIPER_BIN"),
		PiperModel:     os.Getenv("OZY_PIPER_MODEL"),
	})

	// Inicializar Tríada Cognitiva (Charc: Auditor local, Nine: Estratega profundo)
	triad := agent.NewCognitiveTriad(defaultProv, defaultProv)

	// Inicializar Motor de DeepSearch con Citas Estructuradas
	deepSearchEngine := search.NewDeepSearchEngine(defaultProv, nil)

	// Inicializar Task Engine en segundo plano con WebSocket Broadcaster
	taskBroadcaster := ws.NewWSBroadcaster()

	// Start MCP Servers
	if mcpConns, err := db.GetConnectedMCPConnectors(); err == nil {
		var mcpConfigs []mcp.ServerConfig
		
		// Inyectar el servidor híbrido Ozy-Core (Rust)
		mcpConfigs = append(mcpConfigs, mcp.ServerConfig{
			ID:      "ozy_core_native",
			Name:    "ozy-core",
			Command: "./ozy-core/target/release/ozy-core.exe",
			Args:    []string{},
			Env:     []string{},
		})

		for _, c := range mcpConns {
			if c.Status == "connected" {
				mcpConfigs = append(mcpConfigs, mcp.ServerConfig{
					ID:      c.ID,
					Name:    c.Name,
					Command: c.Command,
					Args:    c.GetArgs(),
					Env:     c.GetEnv(),
				})
			}
		}
		// We use context.Background() since they live for the app lifetime
		ctx := context.Background()
		mcp.DefaultRegistry.InitFromDB(ctx, mcpConfigs)
	}

	engine := taskengine.New(db.DB, taskBroadcaster, 100)
	engine.SetVerifier(verifier)
	engine.SetHealer(healer)
	engine.SetSupervisor(triad)
	engine.SetBrowserClient(browserClient)
	engine.SetTelemetryCollector(telemetryCollector)
	engine.SetGoogleClient(googleClient, oauthKey)
	engine.SetVisionCapture(visionCapture)
	engine.SetVoicePipeline(voicePipeline)
	engine.SetDeepSearchEngine(deepSearchEngine)

	// Inicializar Seguridad EDR (Cadena inmutable SHA-256 y Watchdog)
	auditLogger := security.NewAuditLogger(db.DB)
	watchdog := security.NewWatchdog(taskBroadcaster, auditLogger)
	watchdog.Start(10 * time.Second)
	defer watchdog.Stop()

	// Inicializar Time-Travel Undo Engine
	undoEngine := system.NewUndoEngine(db.DB)
	engine.SetAuditLogger(auditLogger)
	engine.SetUndoEngine(undoEngine)
	handlers.SetSecurityComponents(auditLogger, undoEngine, watchdog)

	// Inicializar Smart Clipboard Watcher (Proactividad contextual)
	clipboardWatcher := system.NewClipboardWatcher(taskBroadcaster, 800*time.Millisecond)
	clipboardWatcher.Start()
	defer clipboardWatcher.Stop()

	engine.Start(3)
	handlers.SetTaskEngine(engine)
	defer engine.Stop()

	router := api.NewRouter()

	port := os.Getenv("OZY_PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Printf("Servidor iniciado en puerto %s", port)
		if err := router.Run(":" + port); err != nil {
			log.Fatalf("Error iniciando servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-quit
		log.Println("Apagando servidor...")
		engine.Stop()
		ws.ShutdownAll()
		os.Exit(0)
	}()

	// Mantener el proceso principal activo
	select {}
}
