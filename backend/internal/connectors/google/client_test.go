package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateGmailDraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test_access_token" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := DraftResponse{
			ID: "draft_12345",
		}
		resp.Message.ID = "msg_12345"
		resp.Message.ThreadID = "th_12345"
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Creamos un cliente con RoundTripper apuntando al test server
	customHTTP := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	client := NewClient(customHTTP)
	draft := DraftRequest{
		To:      "colleague@example.com",
		Subject: "Reporte Semanal",
		Body:    "Hola, adjunto el resumen de tareas completadas.",
	}

	res, err := client.CreateGmailDraft(context.Background(), "test_access_token", draft)
	if err != nil {
		t.Fatalf("Error inesperado creando borrador: %v", err)
	}

	if res.ID != "draft_12345" {
		t.Errorf("Draft ID inesperado: %s", res.ID)
	}
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
