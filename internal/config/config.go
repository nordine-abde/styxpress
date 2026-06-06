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

var userHomeDir = os.UserHomeDir

type Config struct {
	Name       string       `json:"name"`
	ContentDir string       `json:"contentDir"`
	PublicDir  string       `json:"publicDir"`
	Deploy     DeployConfig `json:"deploy"`
}

type DeployConfig struct {
	Enabled bool       `json:"enabled"`
	Mode    string     `json:"mode"`
	SFTP    SFTPConfig `json:"sftp"`
}

type SFTPConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	RemotePath     string `json:"remotePath"`
	KeyPath        string `json:"keyPath"`
	KnownHostsPath string `json:"knownHostsPath"`
	DeleteExtra    bool   `json:"deleteExtra"`
}

func Default() Config {
	return Config{
		ContentDir: "content",
		PublicDir:  "public",
		Deploy: DeployConfig{
			Mode: "manual",
			SFTP: SFTPConfig{
				Port: 22,
			},
		},
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
		strings.Contains(c.PublicDir, "\x00") ||
		strings.Contains(c.Deploy.Mode, "\x00") ||
		strings.Contains(c.Deploy.SFTP.Host, "\x00") ||
		strings.Contains(c.Deploy.SFTP.User, "\x00") ||
		strings.Contains(c.Deploy.SFTP.RemotePath, "\x00") ||
		strings.Contains(c.Deploy.SFTP.KeyPath, "\x00") ||
		strings.Contains(c.Deploy.SFTP.KnownHostsPath, "\x00") {
		return fmt.Errorf("%w: fields must not contain NUL bytes", ErrInvalidConfig)
	}
	if c.Deploy.Mode != "manual" && c.Deploy.Mode != "auto" {
		return fmt.Errorf("%w: deploy mode must be manual or auto", ErrInvalidConfig)
	}
	if c.Deploy.SFTP.Port < 1 || c.Deploy.SFTP.Port > 65535 {
		return fmt.Errorf("%w: SFTP port must be between 1 and 65535", ErrInvalidConfig)
	}
	if c.Deploy.Enabled {
		if c.Deploy.SFTP.Host == "" {
			return fmt.Errorf("%w: SFTP host is required when deploy is enabled", ErrInvalidConfig)
		}
		if c.Deploy.SFTP.User == "" {
			return fmt.Errorf("%w: SFTP user is required when deploy is enabled", ErrInvalidConfig)
		}
		if c.Deploy.SFTP.RemotePath == "" {
			return fmt.Errorf("%w: SFTP remote path is required when deploy is enabled", ErrInvalidConfig)
		}
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
	cfg.Deploy.Mode = strings.TrimSpace(cfg.Deploy.Mode)
	if cfg.Deploy.Mode == "" {
		cfg.Deploy.Mode = defaults.Deploy.Mode
	}
	cfg.Deploy.SFTP.Host = strings.TrimSpace(cfg.Deploy.SFTP.Host)
	cfg.Deploy.SFTP.User = strings.TrimSpace(cfg.Deploy.SFTP.User)
	cfg.Deploy.SFTP.RemotePath = strings.TrimSpace(cfg.Deploy.SFTP.RemotePath)
	cfg.Deploy.SFTP.KeyPath = strings.TrimSpace(cfg.Deploy.SFTP.KeyPath)
	cfg.Deploy.SFTP.KnownHostsPath = strings.TrimSpace(cfg.Deploy.SFTP.KnownHostsPath)
	if cfg.Deploy.SFTP.Port == 0 {
		cfg.Deploy.SFTP.Port = defaults.Deploy.SFTP.Port
	}
	return cfg
}

func encode(w io.Writer, cfg Config) error {
	values := map[string]configValue{
		"deploy_enabled":        boolConfigValue(cfg.Deploy.Enabled),
		"deploy_mode":           stringConfigValue(cfg.Deploy.Mode),
		"name":                  stringConfigValue(cfg.Name),
		"content_dir":           stringConfigValue(cfg.ContentDir),
		"public_dir":            stringConfigValue(cfg.PublicDir),
		"sftp_delete_extra":     boolConfigValue(cfg.Deploy.SFTP.DeleteExtra),
		"sftp_host":             stringConfigValue(cfg.Deploy.SFTP.Host),
		"sftp_known_hosts_path": stringConfigValue(cfg.Deploy.SFTP.KnownHostsPath),
		"sftp_key_path":         stringConfigValue(cfg.Deploy.SFTP.KeyPath),
		"sftp_port":             intConfigValue(cfg.Deploy.SFTP.Port),
		"sftp_remote_path":      stringConfigValue(cfg.Deploy.SFTP.RemotePath),
		"sftp_user":             stringConfigValue(cfg.Deploy.SFTP.User),
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

func boolConfigValue(value bool) configValue {
	return configValue{Value: strconv.FormatBool(value)}
}

func intConfigValue(value int) configValue {
	return configValue{Value: strconv.Itoa(value)}
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

		switch key {
		case "name":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.Name = value
		case "content_dir":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.ContentDir = value
		case "public_dir":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.PublicDir = value
		case "deploy_enabled":
			parsed, err := strconv.ParseBool(rawValue)
			if err != nil {
				return fmt.Errorf("%w: line %d value must be true or false", ErrInvalidConfig, lineNumber)
			}
			cfg.Deploy.Enabled = parsed
		case "deploy_mode":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.Deploy.Mode = value
		case "sftp_host":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.Deploy.SFTP.Host = value
		case "sftp_port":
			parsed, err := strconv.Atoi(rawValue)
			if err != nil {
				return fmt.Errorf("%w: line %d value must be an integer", ErrInvalidConfig, lineNumber)
			}
			cfg.Deploy.SFTP.Port = parsed
		case "sftp_user":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.Deploy.SFTP.User = value
		case "sftp_remote_path":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.Deploy.SFTP.RemotePath = value
		case "sftp_key_path":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.Deploy.SFTP.KeyPath = value
		case "sftp_known_hosts_path":
			value, err := decodeStringValue(rawValue, lineNumber)
			if err != nil {
				return err
			}
			cfg.Deploy.SFTP.KnownHostsPath = value
		case "sftp_delete_extra":
			parsed, err := strconv.ParseBool(rawValue)
			if err != nil {
				return fmt.Errorf("%w: line %d value must be true or false", ErrInvalidConfig, lineNumber)
			}
			cfg.Deploy.SFTP.DeleteExtra = parsed
		default:
			return fmt.Errorf("%w: unknown key %q", ErrInvalidConfig, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func decodeStringValue(rawValue string, lineNumber int) (string, error) {
	value, err := strconv.Unquote(rawValue)
	if err != nil {
		return "", fmt.Errorf("%w: line %d value must be a quoted string", ErrInvalidConfig, lineNumber)
	}
	return value, nil
}
