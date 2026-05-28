package rendering

import (
	"strings"

	"github.com/nordine-abde/styxpress/internal/siteconfig"
)

type StyleCSS struct {
	CurrentCSS        string            `json:"currentCss"`
	ThemeCSS          string            `json:"themeCss"`
	BlankThemeCSS     string            `json:"blankThemeCss"`
	CustomCSSIncluded bool              `json:"customCssIncluded"`
	BodyClasses       []string          `json:"bodyClasses"`
	Theme             StyleTheme        `json:"theme"`
	Header            StyleVariantState `json:"header"`
	Footer            StyleVariantState `json:"footer"`
}

type StyleTheme struct {
	Palette string `json:"palette"`
	Font    string `json:"font"`
	Layout  string `json:"layout"`
	Radius  string `json:"radius"`
}

type StyleVariantState struct {
	Variant   string `json:"variant"`
	ClassName string `json:"className"`
	Rendered  bool   `json:"rendered"`
}

func NewStyleCSS(cfg siteconfig.Config) (StyleCSS, error) {
	cfg = siteconfig.WithDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return StyleCSS{}, err
	}

	themeCSS := themeStyleSheet(cfg)

	return StyleCSS{
		CurrentCSS:        styleSheet(cfg),
		ThemeCSS:          themeCSS,
		BlankThemeCSS:     blankThemeStyleSheet(cfg),
		CustomCSSIncluded: strings.TrimSpace(cfg.Theme.CustomCSS) != "",
		BodyClasses:       siteBodyClasses(cfg),
		Theme: StyleTheme{
			Palette: cfg.Theme.Palette,
			Font:    cfg.Theme.Font,
			Layout:  cfg.Theme.Layout,
			Radius:  cfg.Theme.Radius,
		},
		Header: StyleVariantState{
			Variant:   cfg.Header.Variant,
			ClassName: siteHeaderVariantClass(cfg.Header.Variant),
			Rendered:  cfg.Header.Variant != siteconfig.HeaderHidden,
		},
		Footer: StyleVariantState{
			Variant:   cfg.Footer.Variant,
			ClassName: siteFooterVariantClass(cfg.Footer.Variant),
			Rendered:  cfg.Footer.Variant != siteconfig.FooterHidden,
		},
	}, nil
}

func siteBodyClasses(cfg siteconfig.Config) []string {
	cfg = siteconfig.WithDefaults(cfg)
	return []string{
		themeClass(cfg.Theme.Palette),
		fontClass(cfg.Theme.Font),
		layoutClass(cfg.Theme.Layout),
		radiusClass(cfg.Theme.Radius),
	}
}

func siteHeaderVariantClass(variant string) string {
	return "site-header-" + variant
}

func siteFooterVariantClass(variant string) string {
	return "site-footer-" + variant
}

func themeClass(palette string) string {
	return "theme-" + palette
}

func fontClass(font string) string {
	return "font-" + font
}

func layoutClass(layout string) string {
	return "layout-" + layout
}

func radiusClass(radius string) string {
	return "radius-" + radius
}

func themeStyleSheet(cfg siteconfig.Config) string {
	var builder strings.Builder
	builder.WriteString("/* Styxpress theme CSS.\n")
	builder.WriteString("   Generated from the renderer stylesheet for the current draft.\n")
	builder.WriteString("   theme.customCss and savedThemes[].customCss are not included. */\n\n")

	rules := themeCSSRules(siteStyleSheet, cfg)
	for i, rule := range rules {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(rule)
	}
	builder.WriteByte('\n')
	return builder.String()
}

func blankThemeStyleSheet(cfg siteconfig.Config) string {
	var builder strings.Builder
	builder.WriteString("/* Styxpress blank theme CSS.\n")
	builder.WriteString("   Fill the selectors you want to override, then save as custom CSS. */\n\n")

	seen := make(map[string]struct{})
	rules := make([]string, 0)
	for _, selector := range blankThemeClassSelectors(cfg) {
		if addSeenSelector(seen, selector) {
			rules = append(rules, blankCSSRule([]string{selector}))
		}
	}
	rules = append(rules, blankCSSRules(siteStyleSheet, cfg, seen)...)
	for i, rule := range rules {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(rule)
	}
	builder.WriteByte('\n')
	return builder.String()
}

func blankThemeClassSelectors(cfg siteconfig.Config) []string {
	bodyClasses := siteBodyClasses(cfg)
	selectors := []string{
		"body." + strings.Join(bodyClasses, "."),
	}
	for _, className := range bodyClasses {
		selectors = append(selectors, "."+className)
	}
	selectors = append(selectors,
		"."+siteHeaderVariantClass(cfg.Header.Variant),
		"."+siteFooterVariantClass(cfg.Footer.Variant),
	)
	return selectors
}

func themeCSSRules(css string, cfg siteconfig.Config) []string {
	var rules []string
	for _, rule := range parseCSSRules(css) {
		if strings.HasPrefix(rule.prelude, "@") {
			nested := themeCSSRules(rule.body, cfg)
			if len(nested) > 0 {
				rules = append(rules, nestedCSSRule(rule.prelude, nested))
			}
			continue
		}
		selectors := includedRuleSelectors(rule.selectors, cfg)
		if len(selectors) > 0 {
			rules = append(rules, cssRuleWithSelectors(selectors, rule.body))
		}
	}
	return rules
}

func blankCSSRules(css string, cfg siteconfig.Config, seen map[string]struct{}) []string {
	var rules []string
	for _, rule := range parseCSSRules(css) {
		if strings.HasPrefix(rule.prelude, "@") {
			nested := blankCSSRules(rule.body, cfg, seen)
			if len(nested) > 0 {
				rules = append(rules, nestedCSSRule(rule.prelude, nested))
			}
			continue
		}
		selectors := make([]string, 0, len(rule.selectors))
		for _, selector := range includedRuleSelectors(rule.selectors, cfg) {
			if addSeenSelector(seen, selector) {
				selectors = append(selectors, selector)
			}
		}
		if len(selectors) > 0 {
			rules = append(rules, blankCSSRule(selectors))
		}
	}
	return rules
}

func blankCSSRule(selectors []string) string {
	return strings.Join(selectors, ",\n") + " {\n}"
}

func cssRuleWithSelectors(selectors []string, body string) string {
	return strings.Join(selectors, ",\n") + " {" + body + "}"
}

func addSeenSelector(seen map[string]struct{}, selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return false
	}
	if _, ok := seen[selector]; ok {
		return false
	}
	seen[selector] = struct{}{}
	return true
}

func nestedCSSRule(prelude string, rules []string) string {
	var builder strings.Builder
	builder.WriteString(prelude)
	builder.WriteString(" {\n")
	for i, rule := range rules {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(indentCSS(rule))
		builder.WriteByte('\n')
	}
	builder.WriteByte('}')
	return builder.String()
}

func indentCSS(css string) string {
	lines := strings.Split(css, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func includedRuleSelectors(selectors []string, cfg siteconfig.Config) []string {
	included := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		if themeSelectorIncluded(selector, cfg) {
			included = append(included, selector)
		}
	}
	return included
}

func themeSelectorIncluded(selector string, cfg siteconfig.Config) bool {
	if selector == ":root" {
		return true
	}
	if isUniversalSelector(selector) {
		return false
	}
	if optionSelector(selector, themeClasses()) {
		return selectorHasClass(selector, themeClass(cfg.Theme.Palette))
	}
	if optionSelector(selector, fontClasses()) {
		return selectorHasClass(selector, fontClass(cfg.Theme.Font))
	}
	if optionSelector(selector, layoutClasses()) {
		return selectorHasClass(selector, layoutClass(cfg.Theme.Layout))
	}
	if optionSelector(selector, radiusClasses()) {
		return selectorHasClass(selector, radiusClass(cfg.Theme.Radius))
	}
	if optionSelector(selector, headerVariantClasses()) {
		return selectorHasClass(selector, siteHeaderVariantClass(cfg.Header.Variant))
	}
	if optionSelector(selector, footerVariantClasses()) {
		return selectorHasClass(selector, siteFooterVariantClass(cfg.Footer.Variant))
	}
	return true
}

type cssRule struct {
	prelude   string
	body      string
	raw       string
	selectors []string
}

func parseCSSRules(css string) []cssRule {
	var rules []cssRule
	for i := 0; i < len(css); {
		for i < len(css) && isCSSWhitespace(css[i]) {
			i++
		}
		if i >= len(css) {
			break
		}

		start := i
		brace := strings.IndexByte(css[start:], '{')
		if brace == -1 {
			break
		}
		brace += start
		end := matchingCSSBrace(css, brace)
		if end == -1 {
			break
		}

		prelude := strings.TrimSpace(css[start:brace])
		body := css[brace+1 : end]
		rules = append(rules, cssRule{
			prelude:   prelude,
			body:      body,
			raw:       strings.TrimSpace(css[start : end+1]),
			selectors: splitCSSSelectors(prelude),
		})
		i = end + 1
	}
	return rules
}

func matchingCSSBrace(css string, open int) int {
	depth := 0
	for i := open; i < len(css); i++ {
		switch css[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitCSSSelectors(prelude string) []string {
	if strings.HasPrefix(prelude, "@") {
		return nil
	}
	parts := strings.Split(prelude, ",")
	selectors := make([]string, 0, len(parts))
	for _, part := range parts {
		selector := strings.TrimSpace(part)
		if selector != "" {
			selectors = append(selectors, selector)
		}
	}
	return selectors
}

func isCSSWhitespace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}

func isUniversalSelector(selector string) bool {
	return selector == "*" || strings.HasPrefix(selector, "*::")
}

func optionSelector(selector string, classNames []string) bool {
	for _, className := range classNames {
		if selectorHasClass(selector, className) {
			return true
		}
	}
	return false
}

func selectorHasClass(selector string, className string) bool {
	for _, token := range cssClassTokens(selector) {
		if token == className {
			return true
		}
	}
	return false
}

func cssClassTokens(selector string) []string {
	var tokens []string
	for i := 0; i < len(selector); i++ {
		if selector[i] != '.' {
			continue
		}
		start := i + 1
		end := start
		for end < len(selector) && isCSSClassByte(selector[end]) {
			end++
		}
		if end > start {
			tokens = append(tokens, selector[start:end])
			i = end - 1
		}
	}
	return tokens
}

func isCSSClassByte(b byte) bool {
	return b == '-' || b == '_' || b >= '0' && b <= '9' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
}

func themeClasses() []string {
	return []string{
		themeClass(siteconfig.PaletteWarm),
		themeClass(siteconfig.PaletteInk),
		themeClass(siteconfig.PaletteSage),
		themeClass(siteconfig.PaletteClay),
		themeClass(siteconfig.PaletteMidnight),
	}
}

func fontClasses() []string {
	return []string{
		fontClass(siteconfig.FontSystem),
		fontClass(siteconfig.FontSerif),
		fontClass(siteconfig.FontMono),
	}
}

func layoutClasses() []string {
	return []string{
		layoutClass(siteconfig.LayoutClassic),
		layoutClass(siteconfig.LayoutWide),
	}
}

func radiusClasses() []string {
	return []string{
		radiusClass(siteconfig.RadiusNone),
		radiusClass(siteconfig.RadiusSoft),
	}
}

func headerVariantClasses() []string {
	return []string{
		siteHeaderVariantClass(siteconfig.HeaderNav),
		siteHeaderVariantClass(siteconfig.HeaderCentered),
		siteHeaderVariantClass(siteconfig.HeaderMinimal),
		siteHeaderVariantClass(siteconfig.HeaderHidden),
	}
}

func footerVariantClasses() []string {
	return []string{
		siteFooterVariantClass(siteconfig.FooterSimple),
		siteFooterVariantClass(siteconfig.FooterLinks),
		siteFooterVariantClass(siteconfig.FooterHidden),
	}
}
