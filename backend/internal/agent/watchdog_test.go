package agent

import (
	"testing"
	"time"
)

func TestWatchdogRegistry_StartListStop(t *testing.T) {
	reg := &WatchdogRegistry{
		tasks: make(map[string]*WatchdogTask),
	}

	// 1. Iniciar un monitor de puerto
	task := reg.Start(WatchdogPort, "8080", 5*time.Second, "El servidor en 8080 no responde")
	if task == nil || task.ID == "" {
		t.Fatalf("WatchdogStart debería devolver una tarea con ID")
	}

	// 2. Listar monitores
	list := reg.List()
	if len(list) != 1 {
		t.Fatalf("Esperaba 1 monitor en la lista, obtuve %d", len(list))
	}
	if list[0].Target != "8080" {
		t.Errorf("Esperaba target 8080, obtuve %s", list[0].Target)
	}

	// 3. Detener monitor
	stopped := reg.Stop(task.ID)
	if !stopped {
		t.Errorf("WatchdogStop debería devolver true")
	}

	// 4. Verificar lista vacía
	if len(reg.List()) != 0 {
		t.Errorf("La lista debería estar vacía tras detener el monitor")
	}
}
