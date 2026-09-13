package daemon

import (
	"log"

	"github.com/getlantern/systray"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
	
	"github.com/ozyassist/backend/internal/audio"
)

type Daemon struct {
	wakeEngine  *audio.WakeWordEngine
	hk          *hotkey.Hotkey
	onWake      func()
}

func NewDaemon(accessKey string, onWake func()) *Daemon {
	// Initialize Wake Word Engine
	engine, err := audio.NewWakeWordEngine(accessKey, onWake)
	if err != nil {
		log.Printf("[Daemon] Advertencia: No se pudo iniciar Wake Word nativo: %v\n", err)
	}

	// Register Alt+Space Hotkey
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModAlt}, hotkey.KeySpace)

	return &Daemon{
		wakeEngine: engine,
		hk:         hk,
		onWake:     onWake,
	}
}

func (d *Daemon) Start() {
	// Start Wake Word Engine if available
	if d.wakeEngine != nil {
		if err := d.wakeEngine.Start(); err != nil {
			log.Printf("[Daemon] Error iniciando Wake Word: %v", err)
		} else {
			log.Println("[Daemon] Motor Wake Word nativo escuchando en segundo plano.")
		}
	}

	// Hotkey requires mainthread on some OS, but on Windows we can run it in a goroutine
	go func() {
		err := d.hk.Register()
		if err != nil {
			log.Printf("[Daemon] Error registrando Atajo Global (Alt+Espacio): %v\n", err)
			return
		}
		log.Println("[Daemon] Atajo Global registrado: Alt + Espacio")

		for {
			<-d.hk.Keydown()
			log.Println("[Daemon] ¡Atajo Global presionado!")
			if d.onWake != nil {
				d.onWake()
			}
		}
	}()

	// Start System Tray
	// systray.Run requires running on the main thread and blocks indefinitely.
	// This usually means it should be called at the very end of main.go via mainthread.Init
	mainthread.Init(func() {
		systray.Run(d.onReady, d.onExit)
	})
}

func (d *Daemon) Stop() {
	if d.wakeEngine != nil {
		d.wakeEngine.Stop()
	}
	if d.hk != nil {
		d.hk.Unregister()
	}
	systray.Quit()
}

func (d *Daemon) onReady() {
	systray.SetTitle("OzyAssist")
	systray.SetTooltip("OzyAssist AI - Nativo")
	
	mWake := systray.AddMenuItem("Despertar a Ozy", "Invoca al asistente")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Salir", "Cierra OzyAssist por completo")

	go func() {
		for {
			select {
			case <-mWake.ClickedCh:
				if d.onWake != nil {
					d.onWake()
				}
			case <-mQuit.ClickedCh:
				d.Stop()
			}
		}
	}()
}

func (d *Daemon) onExit() {
	log.Println("[Daemon] Cerrando OzyAssist Nativo...")
}
