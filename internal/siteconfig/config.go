package siteconfig

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	FileName            = "site.toml"
	filePermission      = 0o644
	directoryPermission = 0o755

	PaletteInk      = "ink"
	PaletteSage     = "sage"
	PaletteClay     = "clay"
	PaletteMidnight = "midnight"

	FontSystem = "system"
	FontSerif  = "serif"
	FontMono   = "mono"

	LayoutClassic = "classic"
	LayoutWide    = "wide"

	RadiusNone = "none"
	RadiusSoft = "soft"

	HeaderNav      = "nav"
	HeaderCentered = "centered"
	HeaderMinimal  = "minimal"
	HeaderHidden   = "hidden"

	FooterSimple = "simple"
	FooterLinks  = "links"
	FooterHidden = "hidden"
)

var ErrInvalidConfig = errors.New("invalid site config")

type Config struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Theme       ThemeConfig  `json:"theme"`
	Header      HeaderConfig `json:"header"`
	Footer      FooterConfig `json:"footer"`
}

type ThemeConfig struct {
	Palette string `json:"palette"`
	Font    string `json:"font"`
	Layout  string `json:"layout"`
	Radius  string `json:"radius"`
}

type HeaderConfig struct {
	Variant string `json:"variant"`
	Title   string `json:"title"`
	Tagline string `json:"tagline"`
	Links   []Link `json:"links"`
}

type FooterConfig struct {
	Variant string `json:"variant"`
	Text    string `json:"text"`
	Links   []Link `json:"links"`
}

type Link struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

func Default() Config {
	return Config{
		Title:       "Styxpress",
		Description: "Latest posts",
		Theme: ThemeConfig{
			Palette: PaletteInk,
			Font:    FontSystem,
			Layout:  LayoutClassic,
			Radius:  RadiusSoft,
		},
		Header: HeaderConfig{
			Variant: HeaderNav,
			Links: []Link{
				{Label: "Home", Href: "/"},
				{Label: "RSS", Href: "/feed.xml"},
			},
		},
		Footer: FooterConfig{
			Variant: FooterSimple,
			Text:    "Published with Styxpress",
			Links: []Link{
				{Label: "RSS", Href: "/feed.xml"},
			},
		},
	}
}

func Path(contentRoot string) string {
	return filepath.Join(contentRoot, FileName)
}

func Load(contentRoot string) (Config, error) {
	file, err := os.Open(Path(contentRoot))
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	if err := decode(file, &cfg); err != nil {
		return Config{}, err
	}
	cfg = WithDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadOrDefault(contentRoot string) (Config, error) {
	cfg, err := Load(contentRoot)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	return Config{}, err
}

func Save(contentRoot string, cfg Config) error {
	cfg = WithDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(contentRoot, directoryPermission); err != nil {
		return err
	}

	file, err := os.OpenFile(Path(contentRoot), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermission)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := file.Chmod(filePermission); err != nil {
		return err
	}
	return encode(file, cfg)
}

func WithDefaults(cfg Config) Config {
	defaults := Default()
	cfg.Title = strings.TrimSpace(cfg.Title)
	cfg.Description = strings.TrimSpace(cfg.Description)
	if cfg.Title == "" {
		cfg.Title = defaults.Title
	}
	if cfg.Theme.Palette == "" {
		cfg.Theme.Palette = defaults.Theme.Palette
	}
	if cfg.Theme.Font == "" {
		cfg.Theme.Font = defaults.Theme.Font
	}
	if cfg.Theme.Layout == "" {
		cfg.Theme.Layout = defaults.Theme.Layout
	}
	if cfg.Theme.Radius == "" {
		cfg.Theme.Radius = defaults.Theme.Radius
	}
	if cfg.Header.Variant == "" {
		cfg.Header.Variant = defaults.Header.Variant
	}
	cfg.Header.Title = strings.TrimSpace(cfg.Header.Title)
	cfg.Header.Tagline = strings.TrimSpace(cfg.Header.Tagline)
	if cfg.Footer.Variant == "" {
		cfg.Footer.Variant = defaults.Footer.Variant
	}
	cfg.Footer.Text = strings.TrimSpace(cfg.Footer.Text)
	cfg.Header.Links = cleanLinks(cfg.Header.Links)
	cfg.Footer.Links = cleanLinks(cfg.Footer.Links)
	return cfg
}

func (c Config) Validate() error {
	if strings.Contains(c.Title, "\x00") || strings.Contains(c.Description, "\x00") {
		return fmt.Errorf("%w: text fields must not contain NUL bytes", ErrInvalidConfig)
	}
	if !allowed(c.Theme.Palette, PaletteInk, PaletteSage, PaletteClay, PaletteMidnight) {
		return fmt.Errorf("%w: unknown theme palette %q", ErrInvalidConfig, c.Theme.Palette)
	}
	if !allowed(c.Theme.Font, FontSystem, FontSerif, FontMono) {
		return fmt.Errorf("%w: unknown theme font %q", ErrInvalidConfig, c.Theme.Font)
	}
	if !allowed(c.Theme.Layout, LayoutClassic, LayoutWide) {
		return fmt.Errorf("%w: unknown theme layout %q", ErrInvalidConfig, c.Theme.Layout)
	}
	if !allowed(c.Theme.Radius, RadiusNone, RadiusSoft) {
		return fmt.Errorf("%w: unknown theme radius %q", ErrInvalidConfig, c.Theme.Radius)
	}
	if !allowed(c.Header.Variant, HeaderNav, HeaderCentered, HeaderMinimal, HeaderHidden) {
		return fmt.Errorf("%w: unknown header variant %q", ErrInvalidConfig, c.Header.Variant)
	}
	if strings.Contains(c.Header.Title, "\x00") || strings.Contains(c.Header.Tagline, "\x00") {
		return fmt.Errorf("%w: header text must not contain NUL bytes", ErrInvalidConfig)
	}
	if !allowed(c.Footer.Variant, FooterSimple, FooterLinks, FooterHidden) {
		return fmt.Errorf("%w: unknown footer variant %q", ErrInvalidConfig, c.Footer.Variant)
	}
	if strings.Contains(c.Footer.Text, "\x00") {
		return fmt.Errorf("%w: footer text must not contain NUL bytes", ErrInvalidConfig)
	}
	if err := validateLinks("header", c.Header.Links); err != nil {
		return err
	}
	if err := validateLinks("footer", c.Footer.Links); err != nil {
		return err
	}
	return nil
}

func encode(w io.Writer, cfg Config) error {
	lines := []string{
		fmt.Sprintf("title = %s\n", strconv.Quote(cfg.Title)),
		fmt.Sprintf("description = %s\n", strconv.Quote(cfg.Description)),
		"\n[theme]\n",
		fmt.Sprintf("palette = %s\n", strconv.Quote(cfg.Theme.Palette)),
		fmt.Sprintf("font = %s\n", strconv.Quote(cfg.Theme.Font)),
		fmt.Sprintf("layout = %s\n", strconv.Quote(cfg.Theme.Layout)),
		fmt.Sprintf("radius = %s\n", strconv.Quote(cfg.Theme.Radius)),
		"\n[header]\n",
		fmt.Sprintf("variant = %s\n", strconv.Quote(cfg.Header.Variant)),
		fmt.Sprintf("title = %s\n", strconv.Quote(cfg.Header.Title)),
		fmt.Sprintf("tagline = %s\n", strconv.Quote(cfg.Header.Tagline)),
	}
	for _, line := range lines {
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	for _, link := range cfg.Header.Links {
		if _, err := fmt.Fprintf(w, "\n[[header.links]]\nlabel = %s\nhref = %s\n", strconv.Quote(link.Label), strconv.Quote(link.Href)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "\n[footer]\nvariant = %s\ntext = %s\n", strconv.Quote(cfg.Footer.Variant), strconv.Quote(cfg.Footer.Text)); err != nil {
		return err
	}
	for _, link := range cfg.Footer.Links {
		if _, err := fmt.Fprintf(w, "\n[[footer.links]]\nlabel = %s\nhref = %s\n", strconv.Quote(link.Label), strconv.Quote(link.Href)); err != nil {
			return err
		}
	}
	return nil
}

func decode(r io.Reader, cfg *Config) error {
	scanner := bufio.NewScanner(r)
	var section string
	var headerLink *Link
	var footerLink *Link
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[[") && strings.HasSuffix(line, "]]") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "[["), "]]"))
			switch name {
			case "header.links":
				cfg.Header.Links = append(cfg.Header.Links, Link{})
				headerLink = &cfg.Header.Links[len(cfg.Header.Links)-1]
				footerLink = nil
			case "footer.links":
				cfg.Footer.Links = append(cfg.Footer.Links, Link{})
				footerLink = &cfg.Footer.Links[len(cfg.Footer.Links)-1]
				headerLink = nil
			default:
				return fmt.Errorf("%w: line %d unknown array section %q", ErrInvalidConfig, lineNumber, name)
			}
			section = name
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if !allowed(name, "theme", "header", "footer") {
				return fmt.Errorf("%w: line %d unknown section %q", ErrInvalidConfig, lineNumber, name)
			}
			section = name
			headerLink = nil
			footerLink = nil
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%w: line %d must be key = value", ErrInvalidConfig, lineNumber)
		}
		key = strings.TrimSpace(key)
		value, err := strconv.Unquote(strings.TrimSpace(rawValue))
		if err != nil {
			return fmt.Errorf("%w: line %d value must be a quoted string", ErrInvalidConfig, lineNumber)
		}
		if err := assignValue(cfg, section, key, value, headerLink, footerLink); err != nil {
			return fmt.Errorf("%w: line %d %v", ErrInvalidConfig, lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func assignValue(cfg *Config, section string, key string, value string, headerLink *Link, footerLink *Link) error {
	switch section {
	case "":
		switch key {
		case "title":
			cfg.Title = value
		case "description":
			cfg.Description = value
		default:
			return fmt.Errorf("unknown key %q", key)
		}
	case "theme":
		switch key {
		case "palette":
			cfg.Theme.Palette = value
		case "font":
			cfg.Theme.Font = value
		case "layout":
			cfg.Theme.Layout = value
		case "radius":
			cfg.Theme.Radius = value
		default:
			return fmt.Errorf("unknown theme key %q", key)
		}
	case "header":
		switch key {
		case "variant":
			cfg.Header.Variant = value
		case "title":
			cfg.Header.Title = value
		case "tagline":
			cfg.Header.Tagline = value
		default:
			return fmt.Errorf("unknown header key %q", key)
		}
	case "footer":
		switch key {
		case "variant":
			cfg.Footer.Variant = value
		case "text":
			cfg.Footer.Text = value
		default:
			return fmt.Errorf("unknown footer key %q", key)
		}
	case "header.links":
		if headerLink == nil {
			return errors.New("header link entry is missing")
		}
		switch key {
		case "label":
			headerLink.Label = value
		case "href":
			headerLink.Href = value
		default:
			return fmt.Errorf("unknown header link key %q", key)
		}
	case "footer.links":
		if footerLink == nil {
			return errors.New("footer link entry is missing")
		}
		switch key {
		case "label":
			footerLink.Label = value
		case "href":
			footerLink.Href = value
		default:
			return fmt.Errorf("unknown footer link key %q", key)
		}
	default:
		return fmt.Errorf("unknown section %q", section)
	}
	return nil
}

func cleanLinks(links []Link) []Link {
	if len(links) == 0 {
		return nil
	}
	cleaned := make([]Link, 0, len(links))
	for _, link := range links {
		link.Label = strings.TrimSpace(link.Label)
		link.Href = strings.TrimSpace(link.Href)
		if link.Label == "" && link.Href == "" {
			continue
		}
		cleaned = append(cleaned, link)
	}
	return cleaned
}

func validateLinks(owner string, links []Link) error {
	for _, link := range links {
		if link.Label == "" || link.Href == "" {
			return fmt.Errorf("%w: %s links require label and href", ErrInvalidConfig, owner)
		}
		if strings.Contains(link.Label, "\x00") || strings.Contains(link.Href, "\x00") {
			return fmt.Errorf("%w: %s links must not contain NUL bytes", ErrInvalidConfig, owner)
		}
		if !safeHref(link.Href) {
			return fmt.Errorf("%w: %s link href %q must be root-relative, http(s), mailto, or anchor", ErrInvalidConfig, owner, link.Href)
		}
	}
	return nil
}

func safeHref(href string) bool {
	return (strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//")) ||
		strings.HasPrefix(href, "#") ||
		strings.HasPrefix(href, "https://") ||
		strings.HasPrefix(href, "http://") ||
		strings.HasPrefix(href, "mailto:")
}

func allowed(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
