package perception

import (
	"context"
	"testing"
)

func TestBLEManager_ScanAndGATT(t *testing.T) {
	mgr := NewBLEManager()
	ctx := context.Background()

	// 1. Registrar dispositivo BLE de prueba (ej: Sensor IoT ESP32)
	dev := BLEDevice{
		Address: "AA:BB:CC:DD:EE:FF",
		Name:    "ESP32_Workstation_Sensor",
		RSSI:    -65,
	}
	mgr.RegisterDevice(dev)

	// 2. Comprobar catálogo
	devices := mgr.GetDiscoveredDevices()
	if len(devices) != 1 || devices[0].Address != dev.Address {
		t.Fatalf("Dispositivo BLE no registrado correctamente: %v", devices)
	}

	// 3. Escribir característica GATT
	err := mgr.WriteCharacteristic(ctx, dev.Address, "0000181a-0000-1000-8000-00805f9b34fb", "00002a6e-0000-1000-8000-00805f9b34fb", []byte{0x01})
	if err != nil {
		t.Fatalf("Fallo en escritura GATT: %v", err)
	}

	// 4. Leer característica GATT
	res, err := mgr.ReadCharacteristic(ctx, dev.Address, "0000181a-0000-1000-8000-00805f9b34fb", "00002a6e-0000-1000-8000-00805f9b34fb")
	if err != nil || len(res.Data) == 0 {
		t.Fatalf("Fallo en lectura GATT: %v", err)
	}
}
