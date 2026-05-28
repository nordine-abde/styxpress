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
	if cfg.Theme.Palette != PaletteWarm || cfg.Header.Variant != HeaderNav || cfg.Footer.Variant != FooterSimple {
		t.Fatalf("LoadOrDefault() = %#v, want default presets", cfg)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Title:       "Anordine",
		Description: "Software notes",
		Theme: ThemeConfig{
			Palette:   PaletteSage,
			Font:      FontSerif,
			Layout:    LayoutWide,
			Radius:    RadiusNone,
			CustomCSS: ".post-card { border-width: 2px; }\n",
		},
		SavedThemes: []SavedThemeConfig{
			{
				ID:        "sage-wide",
				Name:      "Sage Wide",
				Palette:   PaletteSage,
				Font:      FontSerif,
				Layout:    LayoutWide,
				Radius:    RadiusSoft,
				CustomCSS: ".site-title { letter-spacing: 0.02em; }",
			},
			{
				ID:        "warm_mono",
				Name:      "Warm Mono",
				Palette:   PaletteWarm,
				Font:      FontMono,
				Layout:    LayoutClassic,
				Radius:    RadiusNone,
				CustomCSS: "",
			},
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
	if len(got.SavedThemes) != 2 || got.SavedThemes[0] != cfg.SavedThemes[0] || got.SavedThemes[1] != cfg.SavedThemes[1] {
		t.Fatalf("SavedThemes = %#v, want %#v", got.SavedThemes, cfg.SavedThemes)
	}
	if len(got.Header.Links) != 2 || got.Header.Links[1].Href != "https://github.com/nordine-abde" {
		t.Fatalf("Header links = %#v, want round trip links", got.Header.Links)
	}
	if len(got.Footer.Links) != 2 || got.Footer.Links[1].Href != "mailto:hello@example.com" {
		t.Fatalf("Footer links = %#v, want round trip links", got.Footer.Links)
	}
}

func TestLoadLegacyConfigWithoutSavedThemeFields(t *testing.T) {
	root := t.TempDir()
	data := []byte(`title = "Legacy"
description = "Existing site"

[theme]
palette = "ink"
font = "system"
layout = "classic"
radius = "soft"

[header]
variant = "nav"
title = ""
tagline = ""

[footer]
variant = "simple"
text = "Published with Styxpress"
`)
	if err := os.WriteFile(filepath.Join(root, FileName), data, 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Theme.CustomCSS != "" {
		t.Fatalf("CustomCSS = %q, want empty", cfg.Theme.CustomCSS)
	}
	if len(cfg.SavedThemes) != 0 {
		t.Fatalf("SavedThemes = %#v, want none", cfg.SavedThemes)
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

func TestValidateRejectsInvalidCustomCSSAndSavedThemes(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "active custom CSS NUL",
			mutate: func(cfg *Config) {
				cfg.Theme.CustomCSS = "body {}\x00"
			},
		},
		{
			name: "saved custom CSS NUL",
			mutate: func(cfg *Config) {
				cfg.SavedThemes = []SavedThemeConfig{validSavedTheme()}
				cfg.SavedThemes[0].CustomCSS = "body {}\x00"
			},
		},
		{
			name: "unknown saved palette",
			mutate: func(cfg *Config) {
				cfg.SavedThemes = []SavedThemeConfig{validSavedTheme()}
				cfg.SavedThemes[0].Palette = "custom"
			},
		},
		{
			name: "missing saved id",
			mutate: func(cfg *Config) {
				cfg.SavedThemes = []SavedThemeConfig{validSavedTheme()}
				cfg.SavedThemes[0].ID = ""
			},
		},
		{
			name: "missing saved name",
			mutate: func(cfg *Config) {
				cfg.SavedThemes = []SavedThemeConfig{validSavedTheme()}
				cfg.SavedThemes[0].Name = ""
			},
		},
		{
			name: "unsafe saved id",
			mutate: func(cfg *Config) {
				cfg.SavedThemes = []SavedThemeConfig{validSavedTheme()}
				cfg.SavedThemes[0].ID = "../theme"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.mutate(&cfg)
			if err := cfg.Validate(); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("Validate error = %v, want ErrInvalidConfig", err)
			}
		})
	}
}

func validSavedTheme() SavedThemeConfig {
	return SavedThemeConfig{
		ID:      "sage-wide",
		Name:    "Sage Wide",
		Palette: PaletteSage,
		Font:    FontSerif,
		Layout:  LayoutWide,
		Radius:  RadiusSoft,
	}
}
