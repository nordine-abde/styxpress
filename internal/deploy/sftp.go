package deploy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

var ErrInvalidConfig = errors.New("invalid deploy config")

type Config struct {
	Host           string
	Port           int
	User           string
	RemotePath     string
	KeyPath        string
	KnownHostsPath string
	DeleteExtra    bool
}

type Summary struct {
	OutOfSync  bool `json:"outOfSync"`
	Uploaded   int  `json:"uploaded"`
	Updated    int  `json:"updated"`
	Deleted    int  `json:"deleted"`
	Unchanged  int  `json:"unchanged"`
	LocalFiles int  `json:"localFiles"`
	RemoteOnly int  `json:"remoteOnly"`
}

type fileMeta struct {
	rel     string
	local   string
	size    int64
	modTime time.Time
}

type remoteMeta struct {
	rel     string
	size    int64
	modTime time.Time
}

type plan struct {
	summary Summary
	uploads []fileMeta
	updates []fileMeta
	deletes []remoteMeta
}

type client struct {
	ssh  *ssh.Client
	sftp *sftp.Client
}

func Status(ctx context.Context, localRoot string, cfg Config) (Summary, error) {
	localRoot, cfg, err := prepare(localRoot, cfg)
	if err != nil {
		return Summary{}, err
	}
	conn, err := connect(ctx, cfg)
	if err != nil {
		return Summary{}, err
	}
	defer conn.close()

	local, err := localFiles(localRoot)
	if err != nil {
		return Summary{}, err
	}
	remote, err := remoteFiles(conn.sftp, cfg.RemotePath)
	if err != nil {
		return Summary{}, err
	}
	return buildPlan(local, remote, cfg.DeleteExtra).summary, nil
}

func Sync(ctx context.Context, localRoot string, cfg Config) (Summary, error) {
	localRoot, cfg, err := prepare(localRoot, cfg)
	if err != nil {
		return Summary{}, err
	}
	conn, err := connect(ctx, cfg)
	if err != nil {
		return Summary{}, err
	}
	defer conn.close()

	if err := mkdirAll(conn.sftp, cfg.RemotePath); err != nil {
		return Summary{}, err
	}
	local, err := localFiles(localRoot)
	if err != nil {
		return Summary{}, err
	}
	remote, err := remoteFiles(conn.sftp, cfg.RemotePath)
	if err != nil {
		return Summary{}, err
	}
	next := buildPlan(local, remote, cfg.DeleteExtra)
	for _, file := range append(next.uploads, next.updates...) {
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		if err := uploadFile(conn.sftp, file, path.Join(cfg.RemotePath, file.rel)); err != nil {
			return Summary{}, err
		}
	}
	if cfg.DeleteExtra {
		for _, file := range next.deletes {
			if err := ctx.Err(); err != nil {
				return Summary{}, err
			}
			if err := conn.sftp.Remove(path.Join(cfg.RemotePath, file.rel)); err != nil && !isNotExist(err) {
				return Summary{}, err
			}
		}
		if err := removeEmptyDirs(conn.sftp, cfg.RemotePath); err != nil {
			return Summary{}, err
		}
	}
	next.summary.OutOfSync = false
	return next.summary, nil
}

func prepare(localRoot string, cfg Config) (string, Config, error) {
	localRoot = strings.TrimSpace(localRoot)
	if localRoot == "" {
		return "", Config{}, fmt.Errorf("%w: public folder is required", ErrInvalidConfig)
	}
	absLocalRoot, err := filepath.Abs(localRoot)
	if err != nil {
		return "", Config{}, err
	}
	info, err := os.Stat(absLocalRoot)
	if err != nil {
		return "", Config{}, err
	}
	if !info.IsDir() {
		return "", Config{}, fmt.Errorf("%w: public folder must be a directory", ErrInvalidConfig)
	}
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.User = strings.TrimSpace(cfg.User)
	cfg.RemotePath = strings.TrimSpace(cfg.RemotePath)
	cfg.KeyPath = strings.TrimSpace(cfg.KeyPath)
	cfg.KnownHostsPath = strings.TrimSpace(cfg.KnownHostsPath)
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Host == "" {
		return "", Config{}, fmt.Errorf("%w: SFTP host is required", ErrInvalidConfig)
	}
	if cfg.User == "" {
		return "", Config{}, fmt.Errorf("%w: SFTP user is required", ErrInvalidConfig)
	}
	remotePath, err := cleanRemotePath(cfg.RemotePath, cfg.DeleteExtra)
	if err != nil {
		return "", Config{}, err
	}
	cfg.RemotePath = remotePath
	if cfg.Port < 1 || cfg.Port > 65535 {
		return "", Config{}, fmt.Errorf("%w: SFTP port must be between 1 and 65535", ErrInvalidConfig)
	}
	return filepath.Clean(absLocalRoot), cfg, nil
}

func cleanRemotePath(value string, deleteExtra bool) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%w: SFTP remote path is required", ErrInvalidConfig)
	}
	if strings.Contains(value, "\x00") {
		return "", fmt.Errorf("%w: SFTP remote path contains NUL byte", ErrInvalidConfig)
	}
	if !strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("%w: SFTP remote path must be absolute", ErrInvalidConfig)
	}
	cleaned := path.Clean(value)
	if cleaned == "." {
		cleaned = "/"
	}
	if cleaned == "/" && deleteExtra {
		return "", fmt.Errorf("%w: refusing to delete extra files at remote root", ErrInvalidConfig)
	}
	return cleaned, nil
}

func connect(ctx context.Context, cfg Config) (*client, error) {
	methods, cleanup, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	callback, err := hostKeyCallback(cfg.KnownHostsPath)
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	dialer := net.Dialer{Timeout: 20 * time.Second}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(rawConn, addr, &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            methods,
		HostKeyCallback: callback,
		Timeout:         20 * time.Second,
	})
	if err != nil {
		_ = rawConn.Close()
		return nil, err
	}
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}
	return &client{ssh: sshClient, sftp: sftpClient}, nil
}

func (c *client) close() {
	_ = c.sftp.Close()
	_ = c.ssh.Close()
}

func authMethods(cfg Config) ([]ssh.AuthMethod, func(), error) {
	var methods []ssh.AuthMethod
	var cleanup []io.Closer
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		conn, err := net.Dial("unix", sock)
		if err == nil {
			cleanup = append(cleanup, conn)
			methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
		}
	}
	keyPaths := candidateKeyPaths(cfg.KeyPath)
	var encryptedKey bool
	for _, keyPath := range keyPaths {
		signer, err := signerFromKeyFile(keyPath)
		if err == nil {
			methods = append(methods, ssh.PublicKeys(signer))
			continue
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if isEncryptedKeyError(err) {
			encryptedKey = true
			if cfg.KeyPath == "" {
				continue
			}
			return nil, closeAll(cleanup), fmt.Errorf("%w: encrypted private keys require ssh-agent", ErrInvalidConfig)
		}
		if cfg.KeyPath != "" {
			return nil, closeAll(cleanup), err
		}
	}
	if len(methods) == 0 {
		message := "no SSH auth methods available; load a key into ssh-agent"
		if encryptedKey {
			message += " or add the encrypted key to ssh-agent"
		} else {
			message += " or configure an unencrypted key path"
		}
		return nil, closeAll(cleanup), fmt.Errorf("%w: %s", ErrInvalidConfig, message)
	}
	return methods, closeAll(cleanup), nil
}

func closeAll(closers []io.Closer) func() {
	return func() {
		for _, closer := range closers {
			_ = closer.Close()
		}
	}
}

func candidateKeyPaths(configured string) []string {
	if configured != "" {
		expanded, err := expandHome(configured)
		if err != nil {
			return []string{configured}
		}
		return []string{expanded}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
}

func signerFromKeyFile(keyPath string) (ssh.Signer, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(data)
}

func isEncryptedKeyError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "passphrase") || strings.Contains(message, "encrypted")
}

func hostKeyCallback(configured string) (ssh.HostKeyCallback, error) {
	paths := knownHostsPaths(configured)
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: known_hosts file not found", ErrInvalidConfig)
	}
	return knownhosts.New(paths...)
}

func knownHostsPaths(configured string) []string {
	if configured != "" {
		expanded, err := expandHome(configured)
		if err != nil {
			return []string{configured}
		}
		if _, err := os.Stat(expanded); err == nil {
			return []string{expanded}
		}
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	candidates := []string{
		filepath.Join(home, ".ssh", "known_hosts"),
		filepath.Join(home, ".ssh", "known_hosts2"),
	}
	var existing []string
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			existing = append(existing, candidate)
		}
	}
	return existing
}

func expandHome(value string) (string, error) {
	if value == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(value, "~/")), nil
	}
	return value, nil
}

func localFiles(root string) (map[string]fileMeta, error) {
	files := map[string]fileMeta{}
	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: public folder contains symlink %s", ErrInvalidConfig, current)
		}
		if entry.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: public folder contains non-regular file %s", ErrInvalidConfig, current)
		}
		rel, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		files[rel] = fileMeta{
			rel:     rel,
			local:   current,
			size:    info.Size(),
			modTime: info.ModTime(),
		}
		return nil
	})
	return files, err
}

func remoteFiles(client *sftp.Client, root string) (map[string]remoteMeta, error) {
	files := map[string]remoteMeta{}
	if _, err := client.Stat(root); err != nil {
		if isNotExist(err) {
			return files, nil
		}
		return nil, err
	}
	err := walkRemote(client, root, "", files)
	return files, err
}

func walkRemote(client *sftp.Client, dir string, relDir string, files map[string]remoteMeta) error {
	entries, err := client.ReadDir(dir)
	if err != nil {
		if isNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		remotePath := path.Join(dir, name)
		rel := path.Join(relDir, name)
		if entry.IsDir() {
			if err := walkRemote(client, remotePath, rel, files); err != nil {
				return err
			}
			continue
		}
		if !entry.Mode().IsRegular() {
			continue
		}
		files[rel] = remoteMeta{
			rel:     rel,
			size:    entry.Size(),
			modTime: entry.ModTime(),
		}
	}
	return nil
}

func buildPlan(local map[string]fileMeta, remote map[string]remoteMeta, deleteExtra bool) plan {
	next := plan{}
	localKeys := make([]string, 0, len(local))
	for rel := range local {
		localKeys = append(localKeys, rel)
	}
	sort.Strings(localKeys)
	for _, rel := range localKeys {
		localFile := local[rel]
		remoteFile, ok := remote[rel]
		switch {
		case !ok:
			next.uploads = append(next.uploads, localFile)
		case sameFile(localFile, remoteFile):
			next.summary.Unchanged++
		default:
			next.updates = append(next.updates, localFile)
		}
	}
	remoteKeys := make([]string, 0, len(remote))
	for rel := range remote {
		remoteKeys = append(remoteKeys, rel)
	}
	sort.Strings(remoteKeys)
	for _, rel := range remoteKeys {
		if _, ok := local[rel]; ok {
			continue
		}
		next.summary.RemoteOnly++
		if deleteExtra {
			next.deletes = append(next.deletes, remote[rel])
		}
	}
	next.summary.Uploaded = len(next.uploads)
	next.summary.Updated = len(next.updates)
	next.summary.Deleted = len(next.deletes)
	next.summary.LocalFiles = len(local)
	next.summary.OutOfSync = next.summary.Uploaded > 0 || next.summary.Updated > 0 || next.summary.Deleted > 0
	return next
}

func sameFile(local fileMeta, remote remoteMeta) bool {
	if local.size != remote.size {
		return false
	}
	delta := local.modTime.Unix() - remote.modTime.Unix()
	return delta >= -1 && delta <= 1
}

func uploadFile(client *sftp.Client, local fileMeta, remotePath string) error {
	if err := mkdirAll(client, path.Dir(remotePath)); err != nil {
		return err
	}
	source, err := os.Open(local.local)
	if err != nil {
		return err
	}
	defer source.Close()

	tmpPath := remotePath + ".styxpress-tmp-" + randomSuffix()
	target, err := client.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		_ = client.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = client.Remove(tmpPath)
		return closeErr
	}
	if err := client.Chmod(tmpPath, 0o644); err != nil {
		_ = client.Remove(tmpPath)
		return err
	}
	if err := client.Rename(tmpPath, remotePath); err != nil {
		if removeErr := client.Remove(remotePath); removeErr != nil && !isNotExist(removeErr) {
			_ = client.Remove(tmpPath)
			return err
		}
		if renameErr := client.Rename(tmpPath, remotePath); renameErr != nil {
			_ = client.Remove(tmpPath)
			return renameErr
		}
	}
	return client.Chtimes(remotePath, local.modTime, local.modTime)
}

func mkdirAll(client *sftp.Client, dir string) error {
	dir = path.Clean(dir)
	if dir == "." || dir == "/" {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(dir, "/"), "/")
	current := ""
	if strings.HasPrefix(dir, "/") {
		current = "/"
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = path.Join(current, part)
		if err := client.Mkdir(current); err != nil && !isExist(err) {
			return err
		}
	}
	return nil
}

func removeEmptyDirs(client *sftp.Client, root string) error {
	dirs, err := remoteDirs(client, root)
	if err != nil {
		return err
	}
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, dir := range dirs {
		if dir == root {
			continue
		}
		if err := client.RemoveDirectory(dir); err != nil && !isNotExist(err) {
			continue
		}
	}
	return nil
}

func remoteDirs(client *sftp.Client, root string) ([]string, error) {
	var dirs []string
	var walk func(string) error
	walk = func(dir string) error {
		dirs = append(dirs, dir)
		entries, err := client.ReadDir(dir)
		if err != nil {
			if isNotExist(err) {
				return nil
			}
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				if err := walk(path.Join(dir, entry.Name())); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return dirs, walk(root)
}

func randomSuffix() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

func isExist(err error) bool {
	return errors.Is(err, os.ErrExist) || strings.Contains(strings.ToLower(err.Error()), "file exists")
}

func isNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "no such file")
}
