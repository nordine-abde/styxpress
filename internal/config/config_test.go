package config

import (
	"errors"
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
		SSHUsePassphrase:   true,
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

func TestSiteStoreMigratesLegacyConfig(t *testing.T) {
	root := t.TempDir()
	legacy := Config{
		Name:               "Legacy site",
		ContentDir:         "/tmp/legacy-content",
		PublicDir:          "/tmp/legacy-public",
		ContentStorageMode: ContentStorageServer,
		RemoteHost:         "example.com",
	}
	if err := Save(filepath.Join(root, configFileName), legacy); err != nil {
		t.Fatalf("Save legacy config: %v", err)
	}

	store := NewSiteStore(root)
	sites, activeID, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if activeID != defaultSiteID {
		t.Fatalf("activeID = %q, want %q", activeID, defaultSiteID)
	}
	if len(sites) != 1 {
		t.Fatalf("sites len = %d, want 1", len(sites))
	}
	if sites[0].ID != defaultSiteID || sites[0].Name != legacy.Name {
		t.Fatalf("site = %#v, want migrated default legacy site", sites[0])
	}
	if sites[0].Config.ContentDir != legacy.ContentDir || sites[0].Config.RemoteHost != legacy.RemoteHost {
		t.Fatalf("site config = %#v, want legacy values", sites[0].Config)
	}
}

func TestSiteStoreDoesNotRemigrateLegacyConfigAfterDeletingLastSite(t *testing.T) {
	root := t.TempDir()
	if err := Save(filepath.Join(root, configFileName), Config{Name: "Legacy site"}); err != nil {
		t.Fatalf("Save legacy config: %v", err)
	}
	store := NewSiteStore(root)
	sites, _, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("sites len = %d, want migrated site", len(sites))
	}

	if _, err := store.Delete(sites[0].ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	sites, activeID, err := store.List()
	if err != nil {
		t.Fatalf("List returned error after delete: %v", err)
	}
	if len(sites) != 0 || activeID != "" {
		t.Fatalf("List() = sites %#v, activeID %q; want no remigrated site", sites, activeID)
	}
}

func TestSiteStoreCreatesSelectsSavesAndDeletesSites(t *testing.T) {
	store := NewSiteStore(t.TempDir())

	first, err := store.Create(Config{Name: "My Blog", ContentDir: "/tmp/blog-content", PublicDir: "/tmp/blog-public"})
	if err != nil {
		t.Fatalf("Create first returned error: %v", err)
	}
	if first.ID != "my-blog" {
		t.Fatalf("first ID = %q, want my-blog", first.ID)
	}
	second, err := store.Create(Config{Name: "My Blog", ContentDir: "/tmp/second-content", PublicDir: "/tmp/second-public"})
	if err != nil {
		t.Fatalf("Create second returned error: %v", err)
	}
	if second.ID != "my-blog-2" {
		t.Fatalf("second ID = %q, want my-blog-2", second.ID)
	}

	selected, err := store.Select(first.ID)
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.ID != first.ID {
		t.Fatalf("selected ID = %q, want %q", selected.ID, first.ID)
	}

	saved, err := store.SaveActive(Config{Name: "Renamed", ContentDir: "/tmp/renamed-content", PublicDir: "/tmp/renamed-public"})
	if err != nil {
		t.Fatalf("SaveActive returned error: %v", err)
	}
	if saved.ID != first.ID || saved.Name != "Renamed" || saved.Config.ContentDir != "/tmp/renamed-content" {
		t.Fatalf("saved site = %#v, want updated active site", saved)
	}

	activeID, err := store.Delete(first.ID)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if activeID == first.ID {
		t.Fatalf("activeID = %q, want a remaining site", activeID)
	}
}

func TestSiteStoreAllowsNoSites(t *testing.T) {
	store := NewSiteStore(t.TempDir())

	sites, activeID, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(sites) != 0 || activeID != "" {
		t.Fatalf("List() = sites %#v, activeID %q; want empty registry", sites, activeID)
	}

	if _, err := store.Active(); !errors.Is(err, ErrNoActiveSite) {
		t.Fatalf("Active error = %v, want ErrNoActiveSite", err)
	}
}

func TestSiteStoreAllowsDeletingLastSite(t *testing.T) {
	store := NewSiteStore(t.TempDir())
	site, err := store.Create(Config{Name: "Temporary site"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	activeID, err := store.Delete(site.ID)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if activeID != "" {
		t.Fatalf("activeID = %q, want no active site", activeID)
	}
	sites, activeID, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(sites) != 0 || activeID != "" {
		t.Fatalf("List() = sites %#v, activeID %q; want empty registry", sites, activeID)
	}
}

func TestSiteStoreRecoversInvalidActiveSiteFile(t *testing.T) {
	root := t.TempDir()
	store := NewSiteStore(root)
	if _, err := store.Create(Config{Name: "Recoverable site"}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, activeSiteFileName), []byte("\n"), filePermission); err != nil {
		t.Fatalf("Write active site: %v", err)
	}

	sites, activeID, err := store.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if activeID == "" || !containsSite(sites, activeID) {
		t.Fatalf("activeID = %q, sites = %#v; want recovered active site", activeID, sites)
	}
}
