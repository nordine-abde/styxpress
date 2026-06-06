package siteconfig

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrDefaultMissingFile(t *testing.T) {
	cfg, err := LoadOrDefault(t.TempDir())
	if err != nil {
		t.Fatalf("LoadOrDefault returned error: %v", err)
	}
	if cfg.Title != "Styxpress" || !cfg.Footer.ShowWatermark {
		t.Fatalf("LoadOrDefault() = %#v, want default title and watermark", cfg)
	}
	if len(cfg.Header.Links) == 0 || cfg.Header.Links[0].Href != "/" {
		t.Fatalf("Header links = %#v, want default home link", cfg.Header.Links)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Title:       "Anordine",
		Description: "Software notes",
		Header: HeaderConfig{
			Links: []Link{
				{Label: "Home", Href: "/"},
				{Label: "GitHub", Href: "https://github.com/nordine-abde"},
			},
		},
		Footer: FooterConfig{
			Text:          "All notes are local files.",
			ShowWatermark: false,
			Links: []Link{
				{Label: "RSS", Href: "/feed.xml"},
				{Label: "Email", Href: "mailto:hello@example.com"},
			},
		},
	}

	if err := Save(root, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Title != cfg.Title || got.Description != cfg.Description || got.Footer.Text != cfg.Footer.Text || got.Footer.ShowWatermark != cfg.Footer.ShowWatermark {
		t.Fatalf("Load() = %#v, want %#v", got, cfg)
	}
	if len(got.Header.Links) != 2 || got.Header.Links[1].Href != "https://github.com/nordine-abde" {
		t.Fatalf("Header links = %#v, want round trip links", got.Header.Links)
	}
	if len(got.Footer.Links) != 2 || got.Footer.Links[1].Href != "mailto:hello@example.com" {
		t.Fatalf("Footer links = %#v, want round trip links", got.Footer.Links)
	}
}

func TestLoadNormalizesCommonSitemapTypo(t *testing.T) {
	root := t.TempDir()
	data := []byte(`title = "Typo"

[footer]

[[footer.links]]
label = "sitemap"
href = "/sitmap.xml"
`)
	if err := os.WriteFile(filepath.Join(root, FileName), data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	got, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got.Footer.Links) != 1 || got.Footer.Links[0].Href != "/sitemap.xml" {
		t.Fatalf("Footer links = %#v, want normalized sitemap href", got.Footer.Links)
	}
}

func TestLoadRejectsRemovedThemeSection(t *testing.T) {
	root := t.TempDir()
	data := []byte(`title = "Legacy"

[theme]
palette = "ink"
`)
	if err := os.WriteFile(filepath.Join(root, FileName), data, 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	if _, err := Load(root); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load error = %v, want ErrInvalidConfig", err)
	}
}

func TestSaveUsesPublicContentPermissions(t *testing.T) {
	root := t.TempDir()
	if err := Save(root, Default()); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	info, err := os.Stat(filepath.Join(root, FileName))
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if got := info.Mode().Perm(); got != filePermission {
		t.Fatalf("site config mode = %v, want %v", got, os.FileMode(filePermission))
	}
}

func TestValidateRejectsUnsafeLinksAndNULText(t *testing.T) {
	cfg := Default()
	cfg.Header.Links = []Link{{Label: "Unsafe", Href: "javascript:alert(1)"}}
	if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Validate unsafe link = %v, want ErrInvalidConfig", err)
	}

	cfg = Default()
	cfg.Footer.Text = "bad\x00"
	if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Validate NUL text = %v, want ErrInvalidConfig", err)
	}
}

func TestLoadRejectsInvalidWatermarkValue(t *testing.T) {
	root := t.TempDir()
	data := []byte(`title = "Bad"

[footer]
showWatermark = "nope"
`)
	if err := os.WriteFile(filepath.Join(root, FileName), data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(root); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load error = %v, want ErrInvalidConfig", err)
	}
}
