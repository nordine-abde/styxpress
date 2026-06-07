package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadOrDefaultMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")

	cfg, err := LoadOrDefault(path)
	if err != nil {
		t.Fatalf("LoadOrDefault returned error: %v", err)
	}
	if cfg.ContentDir != "content" || cfg.PublicDir != "public" {
		t.Fatalf("LoadOrDefault() = %#v, want default local paths", cfg)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "styxpress", "config.toml")
	cfg := Config{
		Name:       "My Blog",
		ContentDir: "/tmp/content",
		PublicDir:  "/tmp/public",
		Deploy: DeployConfig{
			Enabled: true,
			SFTP: SFTPConfig{
				Host:           "example.com",
				Port:           2222,
				User:           "deploy",
				RemotePath:     "/public_html",
				KeyPath:        "~/.ssh/id_ed25519",
				KnownHostsPath: "~/.ssh/known_hosts",
				DeleteExtra:    true,
			},
		},
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
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(data), "deploy_mode") {
		t.Fatalf("saved config contains removed deploy_mode key:\n%s", data)
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
	if got.ContentDir != "content" || got.PublicDir != "public" {
		t.Fatalf("Load() = %#v, want default paths", got)
	}
	if got.Deploy.SFTP.Port != 22 {
		t.Fatalf("Load() deploy = %#v, want default SFTP port", got.Deploy)
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

func TestValidateRejectsNULBytes(t *testing.T) {
	cfg := Default()
	cfg.Deploy.SFTP.Host = "example.com\x00"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate returned nil, want error")
	}
}

func TestValidateRejectsOverlappingContentAndPublicDirs(t *testing.T) {
	root := t.TempDir()
	contentDir := filepath.Join(root, "content")
	publicDir := filepath.Join(contentDir, "public")

	cases := []struct {
		name       string
		contentDir string
		publicDir  string
	}{
		{
			name:       "same directory",
			contentDir: contentDir,
			publicDir:  contentDir,
		},
		{
			name:       "public nested in content",
			contentDir: contentDir,
			publicDir:  publicDir,
		},
		{
			name:       "content nested in public",
			contentDir: publicDir,
			publicDir:  contentDir,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			cfg.ContentDir = tt.contentDir
			cfg.PublicDir = tt.publicDir
			if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("Validate error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func TestValidateRejectsSymlinkOverlappingContentAndPublicDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows setups")
	}

	root := t.TempDir()
	contentDir := filepath.Join(root, "content")
	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		t.Fatalf("MkdirAll content: %v", err)
	}
	publicLink := filepath.Join(root, "public-link")
	if err := os.Symlink(contentDir, publicLink); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	cfg := Default()
	cfg.ContentDir = contentDir
	cfg.PublicDir = publicLink
	if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Validate error = %v, want ErrInvalidConfig", err)
	}
}

func TestValidateRequiresSFTPFieldsWhenDeployEnabled(t *testing.T) {
	cfg := Default()
	cfg.Deploy.Enabled = true

	if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Validate error = %v, want ErrInvalidConfig", err)
	}
}

func TestLoadRejectsRemovedRemoteKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "styxpress", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), directoryPermission); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("remote_host = \"example.com\"\n"), filePermission); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Load(path); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load error = %v, want ErrInvalidConfig", err)
	}
}

func TestLoadRejectsRemovedDeployMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "styxpress", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), directoryPermission); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("deploy_mode = \"auto\"\n"), filePermission); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Load(path); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load error = %v, want ErrInvalidConfig", err)
	}
}

func TestSiteStoreMigratesLegacyConfig(t *testing.T) {
	root := t.TempDir()
	legacy := Config{
		Name:       "Legacy site",
		ContentDir: "/tmp/legacy-content",
		PublicDir:  "/tmp/legacy-public",
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
	if sites[0].Config.ContentDir != legacy.ContentDir || sites[0].Config.PublicDir != legacy.PublicDir {
		t.Fatalf("site config = %#v, want legacy local paths", sites[0].Config)
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

func TestSiteStoreSuggestsAndCreatesDefaultHomePaths(t *testing.T) {
	home := t.TempDir()
	previousUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) {
		return home, nil
	}
	t.Cleanup(func() {
		userHomeDir = previousUserHomeDir
	})

	store := NewSiteStore(t.TempDir())
	suggestion, err := store.Suggest("My Blog")
	if err != nil {
		t.Fatalf("Suggest returned error: %v", err)
	}
	if suggestion.ID != "my-blog" || suggestion.Name != "My Blog" {
		t.Fatalf("suggestion = %#v, want normalized my-blog site", suggestion)
	}
	expectedContentDir := filepath.Join(home, "Styxpress", "my-blog", "content")
	expectedPublicDir := filepath.Join(home, "Styxpress", "my-blog", "public")
	if suggestion.Config.ContentDir != expectedContentDir || suggestion.Config.PublicDir != expectedPublicDir {
		t.Fatalf("suggestion config = %#v, want home workspace paths", suggestion.Config)
	}

	created, err := store.Create(Config{Name: "My Blog"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "my-blog" || created.Config.ContentDir != expectedContentDir || created.Config.PublicDir != expectedPublicDir {
		t.Fatalf("created site = %#v, want suggested home paths", created)
	}
	assertDirExists(t, expectedContentDir)
	assertDirExists(t, expectedPublicDir)

	nextSuggestion, err := store.Suggest("My Blog")
	if err != nil {
		t.Fatalf("second Suggest returned error: %v", err)
	}
	if nextSuggestion.ID != "my-blog-2" {
		t.Fatalf("second suggestion ID = %q, want my-blog-2", nextSuggestion.ID)
	}
	if nextSuggestion.Config.PublicDir != filepath.Join(home, "Styxpress", "my-blog-2", "public") {
		t.Fatalf("second suggestion config = %#v, want unique public path", nextSuggestion.Config)
	}
}

func TestSiteStoreCreateWithExplicitPathsDoesNotNeedHome(t *testing.T) {
	previousUserHomeDir := userHomeDir
	userHomeDir = func() (string, error) {
		return "", os.ErrNotExist
	}
	t.Cleanup(func() {
		userHomeDir = previousUserHomeDir
	})

	root := t.TempDir()
	contentDir := filepath.Join(root, "content")
	publicDir := filepath.Join(root, "public")
	store := NewSiteStore(t.TempDir())
	site, err := store.Create(Config{
		Name:       "Explicit Paths",
		ContentDir: contentDir,
		PublicDir:  publicDir,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if site.Config.ContentDir != contentDir || site.Config.PublicDir != publicDir {
		t.Fatalf("site config = %#v, want explicit paths", site.Config)
	}
	assertDirExists(t, contentDir)
	assertDirExists(t, publicDir)
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

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) returned error: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", path)
	}
}
