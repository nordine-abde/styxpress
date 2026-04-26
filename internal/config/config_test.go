package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOrDefaultMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")

	cfg, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("LoadOrDefault returned error: %v", err)
	}
	if cfg.ContentStorageMode != ContentStorageLocal {
		t.Fatalf("ContentStorageMode = %q, want %q", cfg.ContentStorageMode, ContentStorageLocal)
	}
	if cfg.ContentDir != "content" {
		t.Fatalf("ContentDir = %q, want content", cfg.ContentDir)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "styxpress", "config.toml")
	cfg := Config{
		SiteBaseURL:        "https://example.com",
		ContentDir:         "/tmp/content",
		PublicDir:          "/tmp/public",
		ContentStorageMode: ContentStorageServer,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         "/home/me/.ssh/id_ed25519",
		RemotePublicDir:    "/srv/site/public",
		RemoteContentDir:   "/srv/site/content",
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got != cfg {
		t.Fatalf("Load() = %#v, want %#v", got, cfg)
	}
}

func TestSaveAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "styxpress", "config.toml")

	if err := Save(path, Config{}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.ContentDir != "content" || got.PublicDir != "public" || got.ContentStorageMode != ContentStorageLocal {
		t.Fatalf("Load() = %#v, want default paths and storage mode", got)
	}
}

func TestSaveUsesRestrictiveFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file mode bits are not enforced on Windows")
	}

	path := filepath.Join(t.TempDir(), "styxpress", "config.toml")
	if err := Save(path, Default()); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if got := info.Mode().Perm(); got != filePermission {
		t.Fatalf("config mode = %v, want %v", got, os.FileMode(filePermission))
	}
}

func TestValidateRejectsUnknownStorageMode(t *testing.T) {
	cfg := Default()
	cfg.ContentStorageMode = "shared"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate returned nil, want error")
	}
}
