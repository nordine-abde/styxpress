package rendering

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nordine-abde/styxpress/internal/siteconfig"
)

func TestNewStyleCSSReturnsCurrentThemeCSSAndBlankTemplate(t *testing.T) {
	cfg := siteconfig.Default()
	cfg.Theme = siteconfig.ThemeConfig{
		Palette:   siteconfig.PaletteMidnight,
		Font:      siteconfig.FontMono,
		Layout:    siteconfig.LayoutWide,
		Radius:    siteconfig.RadiusNone,
		CustomCSS: ".site-main { outline: 3px solid lime; }",
	}
	cfg.Header.Variant = siteconfig.HeaderCentered
	cfg.Footer.Variant = siteconfig.FooterLinks
	cfg.SavedThemes = []siteconfig.SavedThemeConfig{{
		ID:        "saved",
		Name:      "Saved",
		Palette:   siteconfig.PaletteSage,
		Font:      siteconfig.FontSerif,
		Layout:    siteconfig.LayoutClassic,
		Radius:    siteconfig.RadiusSoft,
		CustomCSS: ".saved-theme { color: red; }",
	}}

	css, err := NewStyleCSS(cfg)
	if err != nil {
		t.Fatalf("NewStyleCSS returned error: %v", err)
	}

	if !css.CustomCSSIncluded {
		t.Fatalf("CustomCSSIncluded = false, want true")
	}
	if !reflect.DeepEqual(css.BodyClasses, []string{"theme-midnight", "font-mono", "layout-wide", "radius-none"}) {
		t.Fatalf("BodyClasses = %#v, want current theme classes", css.BodyClasses)
	}
	if css.Theme.Palette != siteconfig.PaletteMidnight || css.Theme.Font != siteconfig.FontMono || css.Theme.Layout != siteconfig.LayoutWide || css.Theme.Radius != siteconfig.RadiusNone {
		t.Fatalf("Theme = %#v, want current theme state", css.Theme)
	}
	if css.Header.ClassName != "site-header-centered" || !css.Header.Rendered {
		t.Fatalf("Header = %#v, want centered rendered header", css.Header)
	}
	if css.Footer.ClassName != "site-footer-links" || !css.Footer.Rendered {
		t.Fatalf("Footer = %#v, want links rendered footer", css.Footer)
	}
	for _, expected := range []string{
		":root {",
		".theme-midnight {",
		".font-mono {",
		".layout-wide {",
		".radius-none {",
		".site-header-centered .site-header-inner",
		".site-footer-links .site-footer-inner",
		".site-main {",
		".post-card {",
		"@media (min-width: 760px)",
	} {
		if !strings.Contains(css.ThemeCSS, expected) {
			t.Fatalf("expected %q in theme CSS:\n%s", expected, css.ThemeCSS)
		}
	}
	for _, excluded := range []string{
		".site-main { outline: 3px solid lime; }",
		".saved-theme { color: red; }",
		".theme-warm {",
		".theme-ink {",
		".theme-sage {",
		".theme-clay {",
		".font-serif {",
	} {
		if strings.Contains(css.ThemeCSS, excluded) {
			t.Fatalf("theme CSS should not contain %q:\n%s", excluded, css.ThemeCSS)
		}
	}
	if !strings.Contains(css.CurrentCSS, ".site-main { outline: 3px solid lime; }") {
		t.Fatalf("current CSS should include draft custom CSS:\n%s", css.CurrentCSS)
	}
	for _, expected := range []string{
		"*::before",
		".theme-warm {",
		".theme-ink {",
		".theme-sage {",
		".font-serif {",
	} {
		if !strings.Contains(css.CurrentCSS, expected) {
			t.Fatalf("current CSS should contain full renderer stylesheet selector %q:\n%s", expected, css.CurrentCSS)
		}
	}
	if strings.Contains(css.CurrentCSS, ".saved-theme { color: red; }") {
		t.Fatalf("current CSS should not include saved theme custom CSS:\n%s", css.CurrentCSS)
	}
	for _, expected := range []string{
		"body.theme-midnight.font-mono.layout-wide.radius-none {\n}",
		".theme-midnight {\n}",
		".font-mono {\n}",
		".layout-wide {\n}",
		".radius-none {\n}",
		".site-header-centered {\n}",
		".site-footer-links {\n}",
		".site-main {\n}",
		"@media (min-width: 760px)",
	} {
		if !strings.Contains(css.BlankThemeCSS, expected) {
			t.Fatalf("expected %q in blank theme CSS:\n%s", expected, css.BlankThemeCSS)
		}
	}
	if strings.Contains(css.BlankThemeCSS, "outline: 3px solid lime") ||
		strings.Contains(css.BlankThemeCSS, "--site-bg:") ||
		strings.Contains(css.BlankThemeCSS, "grid-template-columns:") {
		t.Fatalf("blank theme CSS should contain empty rules only:\n%s", css.BlankThemeCSS)
	}
}

func TestNewStyleCSSReturnsDefaultClassBlocksForBlankTheme(t *testing.T) {
	css, err := NewStyleCSS(siteconfig.Default())
	if err != nil {
		t.Fatalf("NewStyleCSS returned error: %v", err)
	}

	for _, expected := range []string{
		"body.theme-warm.font-system.layout-classic.radius-soft {\n}",
		".theme-warm {\n}",
		".font-system {\n}",
		".layout-classic {\n}",
		".radius-soft {\n}",
	} {
		if !strings.Contains(css.BlankThemeCSS, expected) {
			t.Fatalf("expected %q in blank theme CSS:\n%s", expected, css.BlankThemeCSS)
		}
	}
	if css.CustomCSSIncluded {
		t.Fatalf("CustomCSSIncluded = true, want false")
	}
	if css.CurrentCSS != siteStyleSheet {
		t.Fatalf("CurrentCSS should match renderer stylesheet when no custom CSS is configured")
	}
	if css.CurrentCSS == css.ThemeCSS {
		t.Fatalf("ThemeCSS should be filtered for the selected theme")
	}
}

func TestNewStyleCSSRejectsInvalidSiteConfig(t *testing.T) {
	cfg := siteconfig.Default()
	cfg.Theme.Palette = "unknown"

	_, err := NewStyleCSS(cfg)
	if !errors.Is(err, siteconfig.ErrInvalidConfig) {
		t.Fatalf("NewStyleCSS error = %v, want ErrInvalidConfig", err)
	}
}
