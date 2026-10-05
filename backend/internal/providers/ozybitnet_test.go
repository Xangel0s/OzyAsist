package providers

import (
	"testing"
)

func TestOzyBitNetProvider(t *testing.T) {
	p := NewOzyBitNet("http://localhost:8766/v1")
	if p.Name() != "ozytalk" {
		t.Errorf("expected name 'ozytalk', got '%s'", p.Name())
	}
	if !p.SupportsTools() {
		t.Errorf("expected SupportsTools() to be true")
	}
	models := p.Models()
	if len(models) < 2 {
		t.Errorf("expected at least 2 models, got %d", len(models))
	}
	if models[0] != "ozytalk-1.58b" || models[1] != "ozy-bitcoconut-1.58b" {
		t.Errorf("unexpected models: %v", models)
	}

	pCustom := NewOzyBitNetWithName("ozybitnet", "http://localhost:8766/v1")
	if pCustom.Name() != "ozybitnet" {
		t.Errorf("expected name 'ozybitnet', got '%s'", pCustom.Name())
	}

	talkAlive, coconutAlive := p.HealthCheck()
	t.Logf("HealthCheck results: talkAlive=%v, coconutAlive=%v", talkAlive, coconutAlive)
}
