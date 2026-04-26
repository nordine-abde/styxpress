package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/nordine-abde/styxpress/internal/config"
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

func TestTestSSHUsesOptionalPassphrase(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	cfg := config.Config{
		RemoteHost:      "example.com",
		RemoteUser:      "deploy",
		SSHKeyPath:      "/tmp/id_ed25519",
		RemotePublicDir: "/srv/site/public",
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	server, err := New(configPath, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	var gotPassphrase string
	server.sshTester = func(_ *http.Request, got config.Config, passphrase string) error {
		gotPassphrase = passphrase
		if got.RemoteHost != cfg.RemoteHost {
			t.Fatalf("RemoteHost = %q, want %q", got.RemoteHost, cfg.RemoteHost)
		}
		return nil
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/test-ssh", bytes.NewBufferString(`{"passphrase":"secret"}`))
	request.Header.Set(SessionHeader, server.Token())

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if gotPassphrase != "secret" {
		t.Fatalf("passphrase = %q, want secret", gotPassphrase)
	}
}
