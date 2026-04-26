package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestAPIRequiresSessionToken(t *testing.T) {
	server, err := New(filepath.Join(t.TempDir(), "config.toml"), nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAPIAcceptsSessionHeader(t *testing.T) {
	server, err := New(filepath.Join(t.TempDir(), "config.toml"), nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.Header.Set(SessionHeader, server.Token())

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
