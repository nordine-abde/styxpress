package rendering

import (
	"strings"

	"github.com/nordine-abde/styxpress/internal/siteconfig"
)

type StyleGuide struct {
	StarterCSS        string          `json:"starterCss"`
	CustomCSSIncluded bool            `json:"customCssIncluded"`
	BodyClasses       []string        `json:"bodyClasses"`
	Selectors         []StyleSelector `json:"selectors"`
	HeaderVariants    []StyleVariant  `json:"headerVariants"`
	FooterVariants    []StyleVariant  `json:"footerVariants"`
	Notes             []string        `json:"notes"`
}

type StyleSelector struct {
	Selector    string `json:"selector"`
	ClassName   string `json:"className,omitempty"`
	Kind        string `json:"kind"`
	Current     bool   `json:"current"`
	Description string `json:"description"`
}

type StyleVariant struct {
	Variant   string `json:"variant"`
	Selector  string `json:"selector"`
	ClassName string `json:"className"`
	Current   bool   `json:"current"`
	Rendered  bool   `json:"rendered"`
}

func NewStyleGuide(cfg siteconfig.Config) (StyleGuide, error) {
	cfg = siteconfig.WithDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return StyleGuide{}, err
	}

	bodyClasses := siteBodyClasses(cfg)
	headerVariants := headerStyleVariants(cfg.Header.Variant)
	footerVariants := footerStyleVariants(cfg.Footer.Variant)

	return StyleGuide{
		StarterCSS:        starterStyleSheet(cfg),
		CustomCSSIncluded: false,
		BodyClasses:       bodyClasses,
		Selectors:         styleSelectors(cfg, bodyClasses, headerVariants, footerVariants),
		HeaderVariants:    headerVariants,
		FooterVariants:    footerVariants,
		Notes: []string{
			"starterCss is generated from the renderer base stylesheet and intentionally excludes theme.customCss and savedThemes[].customCss.",
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

func headerStyleVariants(current string) []StyleVariant {
	variants := []string{
		siteconfig.HeaderNav,
		siteconfig.HeaderCentered,
		siteconfig.HeaderMinimal,
		siteconfig.HeaderHidden,
	}
	result := make([]StyleVariant, 0, len(variants))
	for _, variant := range variants {
		className := siteHeaderVariantClass(variant)
		result = append(result, StyleVariant{
			Variant:   variant,
			Selector:  "." + className,
			ClassName: className,
			Current:   variant == current,
			Rendered:  variant != siteconfig.HeaderHidden,
		})
	}
	return result
}

func footerStyleVariants(current string) []StyleVariant {
	variants := []string{
		siteconfig.FooterSimple,
		siteconfig.FooterLinks,
		siteconfig.FooterHidden,
	}
	result := make([]StyleVariant, 0, len(variants))
	for _, variant := range variants {
		className := siteFooterVariantClass(variant)
		result = append(result, StyleVariant{
			Variant:   variant,
			Selector:  "." + className,
			ClassName: className,
			Current:   variant == current,
			Rendered:  variant != siteconfig.FooterHidden,
		})
	}
	return result
}

func starterStyleSheet(cfg siteconfig.Config) string {
	var builder strings.Builder
	builder.WriteString("/* Styxpress custom CSS starter.\n")
	builder.WriteString("   Generated from the renderer base stylesheet for the current draft theme.\n")
	builder.WriteString("   Existing theme.customCss and savedThemes[].customCss are intentionally not included. */\n\n")

	rules := starterCSSRules(siteStyleSheet, cfg)
	for i, rule := range rules {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(rule)
	}
	builder.WriteByte('\n')
	return builder.String()
}

func starterCSSRules(css string, cfg siteconfig.Config) []string {
	var rules []string
	for _, rule := range parseCSSRules(css) {
		if strings.HasPrefix(rule.prelude, "@") {
			if len(starterCSSRules(rule.body, cfg)) > 0 {
				rules = append(rules, rule.raw)
			}
			continue
		}
		if starterRuleIncluded(rule.selectors, cfg) {
			rules = append(rules, rule.raw)
		}
	}
	return rules
}

func starterRuleIncluded(selectors []string, cfg siteconfig.Config) bool {
	for _, selector := range selectors {
		if selector == ":root" {
			return true
		}
		if isUniversalSelector(selector) {
			continue
		}
		if optionSelector(selector, themeClasses()) {
			if selectorHasClass(selector, themeClass(cfg.Theme.Palette)) {
				return true
			}
			continue
		}
		if optionSelector(selector, fontClasses()) {
			if selectorHasClass(selector, fontClass(cfg.Theme.Font)) {
				return true
			}
			continue
		}
		if optionSelector(selector, layoutClasses()) {
			if selectorHasClass(selector, layoutClass(cfg.Theme.Layout)) {
				return true
			}
			continue
		}
		if optionSelector(selector, radiusClasses()) {
			if selectorHasClass(selector, radiusClass(cfg.Theme.Radius)) {
				return true
			}
			continue
		}
		if optionSelector(selector, headerVariantClasses()) {
			if selectorHasClass(selector, siteHeaderVariantClass(cfg.Header.Variant)) {
				return true
			}
			continue
		}
		if optionSelector(selector, footerVariantClasses()) {
			if selectorHasClass(selector, siteFooterVariantClass(cfg.Footer.Variant)) {
				return true
			}
			continue
		}
		return true
	}
	return false
}

func styleSelectors(cfg siteconfig.Config, bodyClasses []string, headerVariants []StyleVariant, footerVariants []StyleVariant) []StyleSelector {
	var selectors []StyleSelector
	seen := make(map[string]struct{})
	add := func(selector string, className string, kind string, current bool, description string) {
		if selector == "" {
			return
		}
		if _, ok := seen[selector]; ok {
			return
		}
		seen[selector] = struct{}{}
		selectors = append(selectors, StyleSelector{
			Selector:    selector,
			ClassName:   className,
			Kind:        kind,
			Current:     current,
			Description: description,
		})
	}

	add("body."+strings.Join(bodyClasses, "."), "", "body", true, "Current body class combination rendered on every page.")
	for _, className := range bodyClasses {
		add("."+className, className, bodyClassKind(className), true, bodyClassDescription(className))
	}
	for _, variant := range headerVariants {
		add(variant.Selector, variant.ClassName, "headerVariant", variant.Current, "Header variant class.")
	}
	for _, variant := range footerVariants {
		add(variant.Selector, variant.ClassName, "footerVariant", variant.Current, "Footer variant class.")
	}
	for _, selector := range starterRuleSelectors(siteStyleSheet, cfg) {
		add(selector, simpleSelectorClass(selector), styleSelectorKind(selector), styleSelectorCurrent(selector, cfg, bodyClasses), styleSelectorDescription(selector))
	}
	return selectors
}

func starterRuleSelectors(css string, cfg siteconfig.Config) []string {
	var selectors []string
	for _, rule := range parseCSSRules(css) {
		if strings.HasPrefix(rule.prelude, "@") {
			selectors = append(selectors, starterRuleSelectors(rule.body, cfg)...)
			continue
		}
		if !starterRuleIncluded(rule.selectors, cfg) {
			continue
		}
		for _, selector := range rule.selectors {
			if isUniversalSelector(selector) {
				continue
			}
			selectors = append(selectors, selector)
		}
	}
	return selectors
}

func bodyClassKind(className string) string {
	switch {
	case strings.HasPrefix(className, "theme-"):
		return "theme"
	case strings.HasPrefix(className, "font-"):
		return "font"
	case strings.HasPrefix(className, "layout-"):
		return "layout"
	case strings.HasPrefix(className, "radius-"):
		return "radius"
	default:
		return "body"
	}
}

func bodyClassDescription(className string) string {
	switch bodyClassKind(className) {
	case "theme":
		return "Current palette class on body."
	case "font":
		return "Current font class on body."
	case "layout":
		return "Current layout width class on body."
	case "radius":
		return "Current border radius class on body."
	default:
		return "Current body class."
	}
}

func styleSelectorKind(selector string) string {
	switch {
	case selector == ":root":
		return "variables"
	case selector == "body" || strings.HasPrefix(selector, "body."):
		return "body"
	case optionSelector(selector, themeClasses()):
		return "theme"
	case optionSelector(selector, fontClasses()):
		return "font"
	case optionSelector(selector, layoutClasses()):
		return "layout"
	case optionSelector(selector, radiusClasses()):
		return "radius"
	case optionSelector(selector, headerVariantClasses()):
		return "headerVariant"
	case optionSelector(selector, footerVariantClasses()):
		return "footerVariant"
	case strings.Contains(selector, "site-header") || strings.Contains(selector, "site-footer") || strings.Contains(selector, "site-nav") || strings.Contains(selector, "site-branding") || strings.Contains(selector, "site-title") || strings.Contains(selector, "skip-link"):
		return "siteChrome"
	case strings.Contains(selector, "post-"):
		return "content"
	default:
		return "base"
	}
}

func styleSelectorCurrent(selector string, cfg siteconfig.Config, bodyClasses []string) bool {
	if selector == ":root" || selector == "body" || selector == "body."+strings.Join(bodyClasses, ".") {
		return true
	}
	currentClasses := []string{
		themeClass(cfg.Theme.Palette),
		fontClass(cfg.Theme.Font),
		layoutClass(cfg.Theme.Layout),
		radiusClass(cfg.Theme.Radius),
		siteHeaderVariantClass(cfg.Header.Variant),
		siteFooterVariantClass(cfg.Footer.Variant),
	}
	for _, className := range currentClasses {
		if selectorHasClass(selector, className) {
			return true
		}
	}
	return false
}

func styleSelectorDescription(selector string) string {
	switch styleSelectorKind(selector) {
	case "variables":
		return "Default CSS variables for colors, width, and radius."
	case "theme":
		return "Current palette override."
	case "font":
		return "Current font override."
	case "layout":
		return "Current layout override."
	case "radius":
		return "Current radius override."
	case "headerVariant":
		return "Header variant selector."
	case "footerVariant":
		return "Footer variant selector."
	case "siteChrome":
		return "Header, footer, navigation, or site shell selector."
	case "content":
		return "Post list or post content selector."
	case "body":
		return "Page body selector."
	default:
		return "Base element selector."
	}
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

func simpleSelectorClass(selector string) string {
	if !strings.HasPrefix(selector, ".") {
		return ""
	}
	tokenEnd := 1
	for tokenEnd < len(selector) && isCSSClassByte(selector[tokenEnd]) {
		tokenEnd++
	}
	if tokenEnd == len(selector) {
		return selector[1:]
	}
	return ""
}

func themeClasses() []string {
	return []string{
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
