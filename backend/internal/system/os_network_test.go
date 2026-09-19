package system

import (
	"net"
	"os"
	"testing"
)

func TestInspectPorts(t *testing.T) {
	// Levantar listener temporal en puerto dinámico
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo abrir puerto de prueba: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	t.Logf("Escuchando temporalmente en puerto %d", port)

	bindings, err := InspectPorts(port)
	if err != nil {
		t.Fatalf("InspectPorts falló: %v", err)
	}

	if len(bindings) == 0 {
		t.Fatalf("no se detectó el puerto %d que acabamos de abrir", port)
	}

	found := false
	currentPID := uint32(os.Getpid())
	for _, b := range bindings {
		if b.Port == port {
			found = true
			t.Logf("Puerto %d detectado: PID=%d, Proceso=%q, Estado=%s", b.Port, b.ProcessID, b.ProcessName, b.State)
			if b.ProcessID != currentPID {
				t.Logf("Aviso: PID reportado %d vs actual %d", b.ProcessID, currentPID)
			}
			break
		}
	}

	if !found {
		t.Errorf("no se encontró binding para el puerto %d", port)
	}
}
