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
	if cfg.Title != "Styxpress" {
		t.Fatalf("Title = %q, want default title", cfg.Title)
	}
	if cfg.Theme.Palette != PaletteInk || cfg.Header.Variant != HeaderNav || cfg.Footer.Variant != FooterSimple {
		t.Fatalf("LoadOrDefault() = %#v, want default presets", cfg)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Title:       "Anordine",
		Description: "Software notes",
		Theme: ThemeConfig{
			Palette: PaletteSage,
			Font:    FontSerif,
			Layout:  LayoutWide,
			Radius:  RadiusNone,
		},
		Header: HeaderConfig{
			Variant: HeaderCentered,
			Title:   "Anordine Lab",
			Tagline: "Quiet notes",
			Links: []Link{
				{Label: "Home", Href: "/"},
				{Label: "GitHub", Href: "https://github.com/nordine-abde"},
			},
		},
		Footer: FooterConfig{
			Variant: FooterLinks,
			Text:    "All notes are local files.",
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
	if got.Title != cfg.Title || got.Theme != cfg.Theme || got.Header.Variant != cfg.Header.Variant || got.Footer.Text != cfg.Footer.Text {
		t.Fatalf("Load() = %#v, want %#v", got, cfg)
	}
	if len(got.Header.Links) != 2 || got.Header.Links[1].Href != "https://github.com/nordine-abde" {
		t.Fatalf("Header links = %#v, want round trip links", got.Header.Links)
	}
	if len(got.Footer.Links) != 2 || got.Footer.Links[1].Href != "mailto:hello@example.com" {
		t.Fatalf("Footer links = %#v, want round trip links", got.Footer.Links)
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

func TestValidateRejectsUnknownPresetsAndUnsafeLinks(t *testing.T) {
	cfg := Default()
	cfg.Theme.Palette = "custom"
	if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Validate unknown palette = %v, want ErrInvalidConfig", err)
	}

	cfg = Default()
	cfg.Header.Links = []Link{{Label: "Unsafe", Href: "javascript:alert(1)"}}
	if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Validate unsafe link = %v, want ErrInvalidConfig", err)
	}
}
