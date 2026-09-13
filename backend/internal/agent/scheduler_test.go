package agent

import (
	"testing"
	"time"
)

func TestParseAlarmDuration(t *testing.T) {
	cases := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"10m", 10 * time.Minute, false},
		{"30s", 30 * time.Second, false},
		{"1h", 1 * time.Hour, false},
		{"invalid", 0, true},
	}

	for _, c := range cases {
		d, err := ParseAlarmDuration(c.input)
		if c.hasError && err == nil {
			t.Errorf("ParseAlarmDuration(%q) esperaba error pero no lo dio", c.input)
		}
		if !c.hasError && err != nil {
			t.Errorf("ParseAlarmDuration(%q) error inesperado: %v", c.input, err)
		}
		if !c.hasError && d != c.expected {
			t.Errorf("ParseAlarmDuration(%q) = %v, want %v", c.input, d, c.expected)
		}
	}
}

func TestScheduleAlarm_Creation(t *testing.T) {
	alarm, err := ScheduleAlarm("5s", "Test de alarma")
	if err != nil {
		t.Fatalf("ScheduleAlarm falló: %v", err)
	}

	if alarm.ID == "" {
		t.Errorf("ID de alarma no debe ser vacío")
	}

	alarmsMu.Lock()
	_, exists := activeAlarms[alarm.ID]
	alarmsMu.Unlock()

	if !exists {
		t.Errorf("Alarma no registrada en activeAlarms")
	}

	// Limpiar
	alarm.Timer.Stop()
	alarmsMu.Lock()
	delete(activeAlarms, alarm.ID)
	alarmsMu.Unlock()
}
