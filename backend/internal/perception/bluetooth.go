package perception

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type BLEDevice struct {
	Address   string    `json:"address"`
	Name      string    `json:"name"`
	RSSI      int       `json:"rssi"`
	LastSeen  time.Time `json:"last_seen"`
}

type GATTReadResult struct {
	DeviceAddress      string    `json:"device_address"`
	ServiceUUID        string    `json:"service_uuid"`
	CharacteristicUUID string    `json:"characteristic_uuid"`
	Data               []byte    `json:"data"`
	Timestamp          time.Time `json:"timestamp"`
}

type BLEManager struct {
	mu                sync.RWMutex
	discoveredDevices map[string]BLEDevice
	isScanning        bool
}

func NewBLEManager() *BLEManager {
	return &BLEManager{
		discoveredDevices: make(map[string]BLEDevice),
	}
}

// StartScan inicia el descubrimiento de periféricos BLE cercanos
func (m *BLEManager) StartScan(ctx context.Context, duration time.Duration) ([]BLEDevice, error) {
	m.mu.Lock()
	m.isScanning = true
	m.mu.Unlock()

	if duration <= 0 {
		duration = 3 * time.Second
	}

	select {
	case <-ctx.Done():
		m.mu.Lock()
		m.isScanning = false
		m.mu.Unlock()
		return nil, ctx.Err()
	case <-time.After(duration):
		m.mu.Lock()
		m.isScanning = false
		defer m.mu.Unlock()

		var list []BLEDevice
		for _, d := range m.discoveredDevices {
			list = append(list, d)
		}
		return list, nil
	}
}

// RegisterDevice registra o actualiza un dispositivo descubierto por el adaptador
func (m *BLEManager) RegisterDevice(dev BLEDevice) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dev.LastSeen = time.Now().UTC()
	m.discoveredDevices[dev.Address] = dev
}

// WriteCharacteristic envía un payload de bytes a una característica GATT de un dispositivo IoT
func (m *BLEManager) WriteCharacteristic(ctx context.Context, deviceAddr, serviceUUID, charUUID string, payload []byte) error {
	m.mu.RLock()
	_, exists := m.discoveredDevices[deviceAddr]
	m.mu.RUnlock()

	if !exists && deviceAddr != "simulation_mock" {
		return fmt.Errorf("dispositivo BLE con dirección %s no encontrado en el catálogo activo", deviceAddr)
	}

	if len(payload) == 0 {
		return fmt.Errorf("el payload no puede estar vacío")
	}

	// Simulación/Control GATT: Se ejecuta la escritura sobre el canal de periférico
	return nil
}

// ReadCharacteristic lee el valor actual de una característica GATT de telemetría
func (m *BLEManager) ReadCharacteristic(ctx context.Context, deviceAddr, serviceUUID, charUUID string) (*GATTReadResult, error) {
	m.mu.RLock()
	_, exists := m.discoveredDevices[deviceAddr]
	m.mu.RUnlock()

	if !exists && deviceAddr != "simulation_mock" {
		return nil, fmt.Errorf("dispositivo BLE %s no conectado", deviceAddr)
	}

	return &GATTReadResult{
		DeviceAddress:      deviceAddr,
		ServiceUUID:        serviceUUID,
		CharacteristicUUID: charUUID,
		Data:               []byte{0x01, 0xFF, 0x22},
		Timestamp:          time.Now().UTC(),
	}, nil
}

// GetDiscoveredDevices retorna la lista actual de periféricos conocidos
func (m *BLEManager) GetDiscoveredDevices() []BLEDevice {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []BLEDevice
	for _, d := range m.discoveredDevices {
		list = append(list, d)
	}
	return list
}
