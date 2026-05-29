package publishing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	Stat(path string) (os.FileInfo, error)
	Open(path string) (RemoteReader, error)
	Close() error
}

type RemoteFile interface {
	io.Writer
	Close() error
}

type RemoteReader interface {
	io.Reader
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
	Passphrase      string
	RemoteOnlyPaths []string
}

type Result struct {
	UploadedPaths []string `json:"uploadedPaths"`
	CleanupPaths  []string `json:"cleanupPaths,omitempty"`
}

type VerificationStatus string

const (
	VerificationStatusNotOnRemote    VerificationStatus = "not_on_remote"
	VerificationStatusChangesPending VerificationStatus = "changes_pending"
	VerificationStatusPublished      VerificationStatus = "published"
	VerificationStatusUnknown        VerificationStatus = "unknown"
	VerificationStatusStillOnRemote  VerificationStatus = "still_on_remote"
)

type VerificationResult struct {
	Files   []FileVerification  `json:"files"`
	Summary VerificationSummary `json:"summary"`
}

type VerificationSummary struct {
	Total          int `json:"total"`
	Published      int `json:"published"`
	ChangesPending int `json:"changesPending"`
	NotOnRemote    int `json:"notOnRemote"`
	Unknown        int `json:"unknown"`
	StillOnRemote  int `json:"stillOnRemote"`
}

type FileVerification struct {
	RelativePath string             `json:"relativePath"`
	LocalPath    string             `json:"localPath"`
	RemotePath   string             `json:"remotePath"`
	Status       VerificationStatus `json:"status"`
	LocalSize    int64              `json:"localSize"`
	RemoteSize   int64              `json:"remoteSize"`
	LocalSHA256  string             `json:"localSha256,omitempty"`
	RemoteSHA256 string             `json:"remoteSha256,omitempty"`
	Error        string             `json:"error,omitempty"`
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

func (p *Publisher) VerifyRemote(ctx context.Context, opts Options) (VerificationResult, error) {
	if err := validatePublishConfig(p.cfg); err != nil {
		return VerificationResult{}, err
	}

	client, err := p.dial(ctx, opts.Passphrase)
	if err != nil {
		return VerificationResult{}, err
	}
	defer client.Close()

	result := VerificationResult{Files: []FileVerification{}}
	if err := p.verifyTree(client, p.cfg.PublicDir, p.cfg.RemotePublicDir, &result); err != nil {
		return result, err
	}
	remoteRoot, err := cleanRemoteDir(p.cfg.RemotePublicDir)
	if err != nil {
		return result, err
	}
	for _, relativePath := range opts.RemoteOnlyPaths {
		verification := verifyRemoteOnlyFile(client, remoteRoot, relativePath)
		if verification.Status == VerificationStatusNotOnRemote {
			continue
		}
		result.Files = append(result.Files, verification)
		result.Summary.add(verification.Status)
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

func (p *Publisher) verifyTree(client Client, localRoot string, remoteRoot string, result *VerificationResult) error {
	localRoot = filepath.Clean(localRoot)
	remoteRoot, err := cleanRemoteDir(remoteRoot)
	if err != nil {
		return err
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
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(localRoot, localPath)
		if err != nil {
			return err
		}
		relativePath := filepath.ToSlash(rel)
		remotePath, err := joinRemotePath(remoteRoot, relativePath)
		if err != nil {
			return err
		}

		verification := verifyFile(client, localPath, remotePath, relativePath)
		result.Files = append(result.Files, verification)
		result.Summary.add(verification.Status)
		return nil
	})
}

func verifyFile(client Client, localPath string, remotePath string, relativePath string) FileVerification {
	verification := FileVerification{
		RelativePath: relativePath,
		LocalPath:    localPath,
		RemotePath:   remotePath,
		Status:       VerificationStatusUnknown,
	}

	localInfo, err := os.Stat(localPath)
	if err != nil {
		verification.Error = err.Error()
		return verification
	}
	verification.LocalSize = localInfo.Size()

	remoteInfo, err := client.Stat(remotePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			verification.Status = VerificationStatusNotOnRemote
			return verification
		}
		verification.Error = err.Error()
		return verification
	}
	if remoteInfo.IsDir() {
		verification.Error = "remote path is a directory"
		return verification
	}
	verification.RemoteSize = remoteInfo.Size()

	if verification.LocalSize != verification.RemoteSize {
		verification.Status = VerificationStatusChangesPending
		return verification
	}

	localHash, err := hashLocalFile(localPath)
	if err != nil {
		verification.Error = err.Error()
		return verification
	}
	remoteHash, err := hashRemoteFile(client, remotePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			verification.Status = VerificationStatusNotOnRemote
			return verification
		}
		verification.Error = err.Error()
		return verification
	}
	verification.LocalSHA256 = localHash
	verification.RemoteSHA256 = remoteHash

	if localHash != remoteHash {
		verification.Status = VerificationStatusChangesPending
		return verification
	}

	verification.Status = VerificationStatusPublished
	return verification
}

func verifyRemoteOnlyFile(client Client, remoteRoot string, relativePath string) FileVerification {
	relativePath = filepath.ToSlash(relativePath)
	remotePath, err := joinRemotePath(remoteRoot, relativePath)
	verification := FileVerification{
		RelativePath: relativePath,
		RemotePath:   remotePath,
		Status:       VerificationStatusNotOnRemote,
	}
	if err != nil {
		verification.Status = VerificationStatusUnknown
		verification.Error = err.Error()
		return verification
	}

	remoteInfo, err := client.Stat(remotePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return verification
		}
		verification.Status = VerificationStatusUnknown
		verification.Error = err.Error()
		return verification
	}
	if remoteInfo.IsDir() {
		verification.Status = VerificationStatusUnknown
		verification.Error = "remote path is a directory"
		return verification
	}

	verification.RemoteSize = remoteInfo.Size()
	verification.Status = VerificationStatusStillOnRemote
	return verification
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

func hashLocalFile(localPath string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return hashReader(file)
}

func hashRemoteFile(client Client, remotePath string) (string, error) {
	file, err := client.Open(remotePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return hashReader(file)
}

func hashReader(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
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

func (s *VerificationSummary) add(status VerificationStatus) {
	s.Total++
	switch status {
	case VerificationStatusPublished:
		s.Published++
	case VerificationStatusChangesPending:
		s.ChangesPending++
	case VerificationStatusNotOnRemote:
		s.NotOnRemote++
	case VerificationStatusStillOnRemote:
		s.StillOnRemote++
	default:
		s.Unknown++
	}
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
