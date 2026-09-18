package memory

import (
	"context"
	"testing"
	"time"
)

func TestMemoryKVStore_SetGetDelete(t *testing.T) {
	store := NewMemoryKVStore()
	defer store.Close()

	ctx := context.Background()

	// 1. Get inexistente
	val, found, err := store.Get(ctx, "clave_no_existe")
	if err != nil {
		t.Fatalf("error inesperado en Get: %v", err)
	}
	if found || val != "" {
		t.Fatalf("se esperaba no encontrado, obtuvo: %v, %s", found, val)
	}

	// 2. Set sin TTL
	err = store.Set(ctx, "perfil", "Juan Desarrollador Go", 0)
	if err != nil {
		t.Fatalf("error en Set: %v", err)
	}

	val, found, err = store.Get(ctx, "perfil")
	if err != nil || !found || val != "Juan Desarrollador Go" {
		t.Fatalf("error recuperando valor guardado: %v, %v, %s", err, found, val)
	}

	// 3. Delete
	err = store.Delete(ctx, "perfil")
	if err != nil {
		t.Fatalf("error en Delete: %v", err)
	}

	val, found, err = store.Get(ctx, "perfil")
	if found {
		t.Fatalf("la clave debió ser eliminada")
	}

	// 4. Flush
	_ = store.Set(ctx, "a", "1", 0)
	_ = store.Set(ctx, "b", "2", 0)
	_ = store.Flush(ctx)

	_, foundA, _ := store.Get(ctx, "a")
	_, foundB, _ := store.Get(ctx, "b")
	if foundA || foundB {
		t.Fatalf("flush no limpió todas las claves")
	}
}

func TestMemoryKVStore_TTLExpiration(t *testing.T) {
	store := NewMemoryKVStore()
	defer store.Close()

	ctx := context.Background()

	// Guardar con TTL de 50 milisegundos
	err := store.Set(ctx, "temp_token", "secret123", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("error en Set: %v", err)
	}

	// Inmediatamente debe existir
	val, found, err := store.Get(ctx, "temp_token")
	if err != nil || !found || val != "secret123" {
		t.Fatalf("el valor temporal debía existir inmediatamente")
	}

	// Esperar a que expire
	time.Sleep(70 * time.Millisecond)

	val, found, err = store.Get(ctx, "temp_token")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if found {
		t.Fatalf("el valor debía haber expirado tras 70ms")
	}
}

func TestParseRedisTarget(t *testing.T) {
	addr, pass, db, err := parseRedisTarget("127.0.0.1:6379")
	if err != nil || addr != "127.0.0.1:6379" || pass != "" || db != 0 {
		t.Fatalf("fallo parseando host:port simple: %s, %s, %d, %v", addr, pass, db, err)
	}

	addr, pass, db, err = parseRedisTarget("redis://:mypassword@localhost:6380/2")
	if err != nil || addr != "localhost:6380" || pass != "mypassword" || db != 2 {
		t.Fatalf("fallo parseando redis:// URL: %s, %s, %d, %v", addr, pass, db, err)
	}
}

func TestHybridStore_Fallback(t *testing.T) {
	mem1 := NewMemoryKVStore()
	mem2 := NewMemoryKVStore()
	defer mem1.Close()
	defer mem2.Close()

	// Simulamos HybridStore pasando mem1 como "redis" y mem2 como "mem"
	hybrid := NewHybridStore(mem1, mem2)
	ctx := context.Background()

	err := hybrid.Set(ctx, "config", "go_pure", 0)
	if err != nil {
		t.Fatalf("error en HybridStore.Set: %v", err)
	}

	val, found, err := hybrid.Get(ctx, "config")
	if err != nil || !found || val != "go_pure" {
		t.Fatalf("fallo recuperando valor híbrido: %v, %v, %s", err, found, val)
	}
}
