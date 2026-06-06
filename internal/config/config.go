package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	appDirName          = "styxpress"
	configFileName      = "config.toml"
	filePermission      = 0o600
	directoryPermission = 0o700
)

var ErrInvalidConfig = errors.New("invalid config")

type Config struct {
	Name       string `json:"name"`
	ContentDir string `json:"contentDir"`
	PublicDir  string `json:"publicDir"`
}

func Default() Config {
	return Config{
		ContentDir: "content",
		PublicDir:  "public",
	}
}

func DefaultPath() (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

func DefaultDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDirName), nil
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
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

func LoadOrDefault(path string) (Config, error) {
	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	return Config{}, err
}

func Save(path string, cfg Config) error {
	cfg = WithDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), directoryPermission); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermission)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := file.Chmod(filePermission); err != nil {
		return err
	}
	return encode(file, cfg)
}

func (c Config) Validate() error {
	if strings.Contains(c.Name, "\x00") ||
		strings.Contains(c.ContentDir, "\x00") ||
		strings.Contains(c.PublicDir, "\x00") {
		return fmt.Errorf("%w: fields must not contain NUL bytes", ErrInvalidConfig)
	}
	return nil
}

func WithDefaults(cfg Config) Config {
	defaults := Default()
	cfg.Name = strings.TrimSpace(cfg.Name)
	if cfg.ContentDir == "" {
		cfg.ContentDir = defaults.ContentDir
	}
	if cfg.PublicDir == "" {
		cfg.PublicDir = defaults.PublicDir
	}
	return cfg
}

func encode(w io.Writer, cfg Config) error {
	values := map[string]configValue{
		"name":        stringConfigValue(cfg.Name),
		"content_dir": stringConfigValue(cfg.ContentDir),
		"public_dir":  stringConfigValue(cfg.PublicDir),
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]
		encoded := value.Value
		if value.Quoted {
			encoded = strconv.Quote(value.Value)
		}
		if _, err := fmt.Fprintf(w, "%s = %s\n", key, encoded); err != nil {
			return err
		}
	}
	return nil
}

type configValue struct {
	Value  string
	Quoted bool
}

func stringConfigValue(value string) configValue {
	return configValue{Value: value, Quoted: true}
}

func decode(r io.Reader, cfg *Config) error {
	scanner := bufio.NewScanner(r)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%w: line %d must be key = value", ErrInvalidConfig, lineNumber)
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)

		value, err := strconv.Unquote(rawValue)
		if err != nil {
			return fmt.Errorf("%w: line %d value must be a quoted string", ErrInvalidConfig, lineNumber)
		}

		switch key {
		case "name":
			cfg.Name = value
		case "content_dir":
			cfg.ContentDir = value
		case "public_dir":
			cfg.PublicDir = value
		default:
			return fmt.Errorf("%w: unknown key %q", ErrInvalidConfig, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
