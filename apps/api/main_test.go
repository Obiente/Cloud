package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apisrv "api/internal/server"
)

func TestServerServesRoot(t *testing.T) {
	handler := apisrv.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	if body := strings.TrimSpace(rec.Body.String()); body != "obiente-cloud-api" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestServerRegistersConnectHandlers(t *testing.T) {
	handler := apisrv.New()
	req := httptest.NewRequest(http.MethodPost, "/obiente.cloud.auth.v1.AuthService/GetCurrentUser", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("expected RPC handler to be registered, received 404")
	}
}
