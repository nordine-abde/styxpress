package rendering

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/nordine-abde/styxpress/internal/siteconfig"
)

func TestNewStyleGuideReturnsCurrentThemeStarterWithoutCustomCSS(t *testing.T) {
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

	guide, err := NewStyleGuide(cfg)
	if err != nil {
		t.Fatalf("NewStyleGuide returned error: %v", err)
	}

	if guide.CustomCSSIncluded {
		t.Fatalf("CustomCSSIncluded = true, want false")
	}
	if !reflect.DeepEqual(guide.BodyClasses, []string{"theme-midnight", "font-mono", "layout-wide", "radius-none"}) {
		t.Fatalf("BodyClasses = %#v, want current theme classes", guide.BodyClasses)
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
		if !strings.Contains(guide.StarterCSS, expected) {
			t.Fatalf("expected %q in starter CSS:\n%s", expected, guide.StarterCSS)
		}
	}
	for _, excluded := range []string{
		".site-main { outline: 3px solid lime; }",
		".saved-theme { color: red; }",
		".theme-sage {",
		".theme-clay {",
		".font-serif {",
	} {
		if strings.Contains(guide.StarterCSS, excluded) {
			t.Fatalf("starter CSS should not contain %q:\n%s", excluded, guide.StarterCSS)
		}
	}
	assertStyleSelector(t, guide.Selectors, "body.theme-midnight.font-mono.layout-wide.radius-none", "", "body", true)
	assertStyleSelector(t, guide.Selectors, ".theme-midnight", "theme-midnight", "theme", true)
	assertStyleSelector(t, guide.Selectors, ".site-header-minimal", "site-header-minimal", "headerVariant", false)
	assertStyleSelector(t, guide.Selectors, ".site-footer-links", "site-footer-links", "footerVariant", true)
	assertStyleVariant(t, guide.HeaderVariants, siteconfig.HeaderCentered, "site-header-centered", true, true)
	assertStyleVariant(t, guide.HeaderVariants, siteconfig.HeaderHidden, "site-header-hidden", false, false)
	assertStyleVariant(t, guide.FooterVariants, siteconfig.FooterLinks, "site-footer-links", true, true)
}

func TestNewStyleGuideRejectsInvalidSiteConfig(t *testing.T) {
	cfg := siteconfig.Default()
	cfg.Theme.Palette = "unknown"

	_, err := NewStyleGuide(cfg)
	if !errors.Is(err, siteconfig.ErrInvalidConfig) {
		t.Fatalf("NewStyleGuide error = %v, want ErrInvalidConfig", err)
	}
}

func assertStyleSelector(t *testing.T, selectors []StyleSelector, selector string, className string, kind string, current bool) {
	t.Helper()
	for _, candidate := range selectors {
		if candidate.Selector != selector {
			continue
		}
		if candidate.ClassName != className || candidate.Kind != kind || candidate.Current != current {
			t.Fatalf("selector %q = %#v, want class=%q kind=%q current=%v", selector, candidate, className, kind, current)
		}
		return
	}
	t.Fatalf("selector %q not found in %#v", selector, selectors)
}

func assertStyleVariant(t *testing.T, variants []StyleVariant, variant string, className string, current bool, rendered bool) {
	t.Helper()
	for _, candidate := range variants {
		if candidate.Variant != variant {
			continue
		}
		if candidate.ClassName != className || candidate.Selector != "."+className || candidate.Current != current || candidate.Rendered != rendered {
			t.Fatalf("variant %q = %#v, want class=%q current=%v rendered=%v", variant, candidate, className, current, rendered)
		}
		return
	}
	t.Fatalf("variant %q not found in %#v", variant, variants)
}
