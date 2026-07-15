package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ozyassist/backend/internal/api"
	"github.com/ozyassist/backend/internal/api/ws"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/providers"
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
	<-quit
	log.Println("Apagando servidor...")
	ws.ShutdownAll()
}
