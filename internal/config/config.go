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

	ContentStorageLocal  = "local"
	ContentStorageServer = "server"
)

var ErrInvalidConfig = errors.New("invalid config")

type Config struct {
	SiteBaseURL        string `json:"siteBaseUrl"`
	ContentDir         string `json:"contentDir"`
	PublicDir          string `json:"publicDir"`
	ContentStorageMode string `json:"contentStorageMode"`
	RemoteHost         string `json:"remoteHost"`
	RemoteUser         string `json:"remoteUser"`
	SSHKeyPath         string `json:"sshKeyPath"`
	RemotePublicDir    string `json:"remotePublicDir"`
	RemoteContentDir   string `json:"remoteContentDir"`
}

func Default() Config {
	return Config{
		ContentDir:         "content",
		PublicDir:          "public",
		ContentStorageMode: ContentStorageLocal,
	}
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDirName, configFileName), nil
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
	switch c.ContentStorageMode {
	case "", ContentStorageLocal, ContentStorageServer:
	default:
		return fmt.Errorf("%w: content_storage_mode must be %q or %q", ErrInvalidConfig, ContentStorageLocal, ContentStorageServer)
	}
	return nil
}

func WithDefaults(cfg Config) Config {
	defaults := Default()
	if cfg.ContentDir == "" {
		cfg.ContentDir = defaults.ContentDir
	}
	if cfg.PublicDir == "" {
		cfg.PublicDir = defaults.PublicDir
	}
	if cfg.ContentStorageMode == "" {
		cfg.ContentStorageMode = defaults.ContentStorageMode
	}
	return cfg
}

func encode(w io.Writer, cfg Config) error {
	values := map[string]string{
		"site_base_url":        cfg.SiteBaseURL,
		"content_dir":          cfg.ContentDir,
		"public_dir":           cfg.PublicDir,
		"content_storage_mode": cfg.ContentStorageMode,
		"remote_host":          cfg.RemoteHost,
		"remote_user":          cfg.RemoteUser,
		"ssh_key_path":         cfg.SSHKeyPath,
		"remote_public_dir":    cfg.RemotePublicDir,
		"remote_content_dir":   cfg.RemoteContentDir,
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if _, err := fmt.Fprintf(w, "%s = %s\n", key, strconv.Quote(values[key])); err != nil {
			return err
		}
	}
	return nil
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
		value, err := strconv.Unquote(strings.TrimSpace(rawValue))
		if err != nil {
			return fmt.Errorf("%w: line %d value must be a quoted string", ErrInvalidConfig, lineNumber)
		}

		switch key {
		case "site_base_url":
			cfg.SiteBaseURL = value
		case "content_dir":
			cfg.ContentDir = value
		case "public_dir":
			cfg.PublicDir = value
		case "content_storage_mode":
			cfg.ContentStorageMode = value
		case "remote_host":
			cfg.RemoteHost = value
		case "remote_user":
			cfg.RemoteUser = value
		case "ssh_key_path":
			cfg.SSHKeyPath = value
		case "remote_public_dir":
			cfg.RemotePublicDir = value
		case "remote_content_dir":
			cfg.RemoteContentDir = value
		default:
			return fmt.Errorf("%w: unknown key %q", ErrInvalidConfig, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
