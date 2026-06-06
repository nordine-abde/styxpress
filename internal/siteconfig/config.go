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
)

var ErrInvalidConfig = errors.New("invalid site config")

type Config struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Header      HeaderConfig `json:"header"`
	Footer      FooterConfig `json:"footer"`
}

type HeaderConfig struct {
	Links []Link `json:"links"`
}

type FooterConfig struct {
	Text          string `json:"text"`
	ShowWatermark bool   `json:"showWatermark"`
	Links         []Link `json:"links"`
}

type Link struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

func Default() Config {
	return Config{
		Title:       "Styxpress",
		Description: "Latest posts",
		Header: HeaderConfig{
			Links: []Link{
				{Label: "Home", Href: "/"},
				{Label: "RSS", Href: "/feed.xml"},
			},
		},
		Footer: FooterConfig{
			ShowWatermark: true,
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

	cfg := Default()
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
	cfg.Header.Links = cleanLinks(cfg.Header.Links)
	cfg.Footer.Text = strings.TrimSpace(cfg.Footer.Text)
	cfg.Footer.Links = cleanLinks(cfg.Footer.Links)
	return cfg
}

func (c Config) Validate() error {
	if strings.Contains(c.Title, "\x00") || strings.Contains(c.Description, "\x00") {
		return fmt.Errorf("%w: text fields must not contain NUL bytes", ErrInvalidConfig)
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
		"\n[header]\n",
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
	if _, err := fmt.Fprintf(w, "\n[footer]\ntext = %s\nshowWatermark = %t\n", strconv.Quote(cfg.Footer.Text), cfg.Footer.ShowWatermark); err != nil {
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
	scanner.Buffer(make([]byte, 1024), 4*1024*1024)
	var section string
	var headerLink *Link
	var footerLink *Link
	headerLinksCleared := false
	footerLinksCleared := false
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
				if !headerLinksCleared {
					cfg.Header.Links = nil
					headerLinksCleared = true
				}
				cfg.Header.Links = append(cfg.Header.Links, Link{})
				headerLink = &cfg.Header.Links[len(cfg.Header.Links)-1]
				footerLink = nil
			case "footer.links":
				if !footerLinksCleared {
					cfg.Footer.Links = nil
					footerLinksCleared = true
				}
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
			if name != "header" && name != "footer" {
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
		rawValue = strings.TrimSpace(rawValue)
		if err := assignValue(cfg, section, key, rawValue, headerLink, footerLink); err != nil {
			return fmt.Errorf("%w: line %d %v", ErrInvalidConfig, lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func assignValue(cfg *Config, section string, key string, rawValue string, headerLink *Link, footerLink *Link) error {
	switch section {
	case "":
		value, err := quotedValue(rawValue)
		if err != nil {
			return err
		}
		switch key {
		case "title":
			cfg.Title = value
		case "description":
			cfg.Description = value
		default:
			return fmt.Errorf("unknown key %q", key)
		}
	case "header.links":
		if headerLink == nil {
			return errors.New("header link entry is missing")
		}
		value, err := quotedValue(rawValue)
		if err != nil {
			return err
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
		value, err := quotedValue(rawValue)
		if err != nil {
			return err
		}
		switch key {
		case "label":
			footerLink.Label = value
		case "href":
			footerLink.Href = value
		default:
			return fmt.Errorf("unknown footer link key %q", key)
		}
	case "header":
		return fmt.Errorf("unknown header key %q", key)
	case "footer":
		switch key {
		case "text":
			value, err := quotedValue(rawValue)
			if err != nil {
				return err
			}
			cfg.Footer.Text = value
		case "showWatermark":
			value, err := strconv.ParseBool(rawValue)
			if err != nil {
				return errors.New("showWatermark must be true or false")
			}
			cfg.Footer.ShowWatermark = value
		default:
			return fmt.Errorf("unknown footer key %q", key)
		}
	default:
		return fmt.Errorf("unknown section %q", section)
	}
	return nil
}

func quotedValue(raw string) (string, error) {
	value, err := strconv.Unquote(raw)
	if err != nil {
		return "", errors.New("value must be a quoted string")
	}
	return value, nil
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
