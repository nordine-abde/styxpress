package publishing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nordine-abde/styxpress/internal/config"
)

var ErrInvalidPublishConfig = errors.New("invalid publish config")

type Client interface {
	MkdirAll(path string) error
	Create(path string) (RemoteFile, error)
	Close() error
}

type RemoteFile interface {
	io.Writer
	Close() error
}

type Dialer interface {
	Dial(ctx context.Context, cfg SSHConfig) (Client, error)
}

type SSHConfig struct {
	Host       string
	User       string
	KeyPath    string
	Passphrase string
}

type Publisher struct {
	cfg    config.Config
	dialer Dialer
}

type Options struct {
	Passphrase string
}

type Result struct {
	UploadedPaths []string `json:"uploadedPaths"`
	CleanupPaths  []string `json:"cleanupPaths,omitempty"`
}

type UploadError struct {
	Path         string
	CleanupPaths []string
	Err          error
}

func (e *UploadError) Error() string {
	if e.Path == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("upload %s: %v", e.Path, e.Err)
}

func (e *UploadError) Unwrap() error {
	return e.Err
}

func New(cfg config.Config, dialer Dialer) *Publisher {
	if dialer == nil {
		dialer = SSHDialer{}
	}
	return &Publisher{
		cfg:    config.WithDefaults(cfg),
		dialer: dialer,
	}
}

func TestSSH(ctx context.Context, cfg config.Config, passphrase string) error {
	client, err := New(cfg, SSHDialer{}).dial(ctx, passphrase)
	if err != nil {
		return err
	}
	return client.Close()
}

func (p *Publisher) Publish(ctx context.Context, opts Options) (Result, error) {
	if err := validatePublishConfig(p.cfg); err != nil {
		return Result{}, err
	}

	client, err := p.dial(ctx, opts.Passphrase)
	if err != nil {
		return Result{}, err
	}
	defer client.Close()

	var result Result
	if err := p.uploadTree(client, p.cfg.PublicDir, p.cfg.RemotePublicDir, &result); err != nil {
		return result, err
	}
	if p.cfg.ContentStorageMode == config.ContentStorageServer {
		if err := p.uploadTree(client, p.cfg.ContentDir, p.cfg.RemoteContentDir, &result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (p *Publisher) dial(ctx context.Context, passphrase string) (Client, error) {
	if err := validateSSHConfig(p.cfg); err != nil {
		return nil, err
	}
	return p.dialer.Dial(ctx, SSHConfig{
		Host:       p.cfg.RemoteHost,
		User:       p.cfg.RemoteUser,
		KeyPath:    p.cfg.SSHKeyPath,
		Passphrase: passphrase,
	})
}

func (p *Publisher) uploadTree(client Client, localRoot string, remoteRoot string, result *Result) error {
	localRoot = filepath.Clean(localRoot)
	remoteRoot, err := cleanRemoteDir(remoteRoot)
	if err != nil {
		return err
	}
	if err := client.MkdirAll(remoteRoot); err != nil {
		return &UploadError{Path: remoteRoot, CleanupPaths: appendCleanup(result.CleanupPaths, remoteRoot), Err: err}
	}

	return filepath.WalkDir(localRoot, func(localPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if localPath == localRoot {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink %s", ErrInvalidPublishConfig, localPath)
		}

		rel, err := filepath.Rel(localRoot, localPath)
		if err != nil {
			return err
		}
		remotePath, err := joinRemotePath(remoteRoot, filepath.ToSlash(rel))
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if err := client.MkdirAll(remotePath); err != nil {
				return &UploadError{Path: remotePath, CleanupPaths: appendCleanup(result.CleanupPaths, remotePath), Err: err}
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}

		if err := uploadFile(client, localPath, remotePath); err != nil {
			return &UploadError{Path: remotePath, CleanupPaths: appendCleanup(result.CleanupPaths, remotePath), Err: err}
		}
		result.UploadedPaths = append(result.UploadedPaths, remotePath)
		result.CleanupPaths = append(result.CleanupPaths, remotePath)
		return nil
	})
}

func uploadFile(client Client, localPath string, remotePath string) error {
	if err := client.MkdirAll(path.Dir(remotePath)); err != nil {
		return err
	}
	source, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer source.Close()

	target, err := client.Create(remotePath)
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	return nil
}

func validatePublishConfig(cfg config.Config) error {
	if err := validateSSHConfig(cfg); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.PublicDir) == "" {
		return fmt.Errorf("%w: public dir is required", ErrInvalidPublishConfig)
	}
	if strings.TrimSpace(cfg.RemotePublicDir) == "" {
		return fmt.Errorf("%w: remote public dir is required", ErrInvalidPublishConfig)
	}
	if cfg.ContentStorageMode == config.ContentStorageServer {
		if strings.TrimSpace(cfg.ContentDir) == "" {
			return fmt.Errorf("%w: content dir is required", ErrInvalidPublishConfig)
		}
		if strings.TrimSpace(cfg.RemoteContentDir) == "" {
			return fmt.Errorf("%w: remote content dir is required", ErrInvalidPublishConfig)
		}
	}
	return nil
}

func validateSSHConfig(cfg config.Config) error {
	if strings.TrimSpace(cfg.RemoteHost) == "" {
		return fmt.Errorf("%w: remote host is required", ErrInvalidPublishConfig)
	}
	if strings.TrimSpace(cfg.RemoteUser) == "" {
		return fmt.Errorf("%w: remote user is required", ErrInvalidPublishConfig)
	}
	if strings.TrimSpace(cfg.SSHKeyPath) == "" {
		return fmt.Errorf("%w: SSH key path is required", ErrInvalidPublishConfig)
	}
	return nil
}

func cleanRemoteDir(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: remote directory is required", ErrInvalidPublishConfig)
	}
	if strings.Contains(value, "\x00") {
		return "", fmt.Errorf("%w: remote directory contains NUL byte", ErrInvalidPublishConfig)
	}
	cleaned := path.Clean(value)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", fmt.Errorf("%w: remote directory must not traverse upward", ErrInvalidPublishConfig)
	}
	return cleaned, nil
}

func joinRemotePath(root string, rel string) (string, error) {
	if strings.Contains(rel, "\x00") {
		return "", fmt.Errorf("%w: remote path contains NUL byte", ErrInvalidPublishConfig)
	}
	cleaned := path.Clean("/" + rel)
	if cleaned == "/" || strings.Contains(cleaned, "/../") {
		return "", fmt.Errorf("%w: remote path must stay within the remote root", ErrInvalidPublishConfig)
	}
	return path.Join(root, strings.TrimPrefix(cleaned, "/")), nil
}

func appendCleanup(paths []string, path string) []string {
	paths = append(paths, path)
	sort.Strings(paths)
	return compactStrings(paths)
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	dst := values[:1]
	for _, value := range values[1:] {
		if value != dst[len(dst)-1] {
			dst = append(dst, value)
		}
	}
	return dst
}
