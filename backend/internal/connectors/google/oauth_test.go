package google

import (
	"context"
	"crypto/rand"
	"net/http"
	"testing"
	"time"
)

func TestPKCE_Generation(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("Error generando PKCE: %v", err)
	}
	if len(verifier) < 40 || len(challenge) < 40 {
		t.Errorf("Longitud de PKCE insuficiente: verifier=%d, challenge=%d", len(verifier), len(challenge))
	}
}

func TestToken_EncryptionDecryption(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	originalToken := "ya29.a0AfH6SMD_ejemplo_token_google_workspace_oauth2"
	encrypted, err := EncryptToken(originalToken, key)
	if err != nil {
		t.Fatalf("Fallo cifrando token: %v", err)
	}

	decrypted, err := DecryptToken(encrypted, key)
	if err != nil {
		t.Fatalf("Fallo descifrando token: %v", err)
	}

	if decrypted != originalToken {
		t.Fatalf("Token descifrado no coincide. Esperado: %s, Obtenido: %s", originalToken, decrypted)
	}
}

func TestEphemeralCallbackListener(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redirectURI, codeChan, err := StartEphemeralCallbackListener(ctx)
	if err != nil {
		t.Fatalf("Error iniciando callback listener: %v", err)
	}

	if redirectURI == "" {
		t.Fatal("redirectURI vacía")
	}

	// Simular callback de Google
	resp, err := http.Get(redirectURI + "?code=test_auth_code_12345")
	if err != nil {
		t.Fatalf("Error haciendo callback HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status inesperado: %d", resp.StatusCode)
	}

	select {
	case code := <-codeChan:
		if code != "test_auth_code_12345" {
			t.Fatalf("Código inesperado: %s", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout esperando código de autorización en canal")
	}
}
