package deploy

import (
	"context"
	"encoding/json"
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
	Secret         string
	StatePath      string
}

type Inspection struct {
	RemoteFiles int `json:"remoteFiles"`
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

type stateFile struct {
	Version int                   `json:"version"`
	Files   map[string]stateEntry `json:"files"`
}

type stateEntry struct {
	Size            int64 `json:"size"`
	ModTimeUnixNano int64 `json:"mtimeUnixNano"`
}

type plan struct {
	summary Summary
	uploads []fileMeta
	updates []fileMeta
	deletes []remoteMeta
}

type client struct {
	conn net.Conn
	ssh  *ssh.Client
	sftp *sftp.Client
}

func Status(_ context.Context, localRoot string, cfg Config) (Summary, error) {
	localRoot, cfg, err := prepare(localRoot, cfg)
	if err != nil {
		return Summary{}, err
	}
	local, err := localFiles(localRoot)
	if err != nil {
		return Summary{}, err
	}
	previous, err := loadState(cfg.StatePath)
	if err != nil {
		return Summary{}, err
	}
	return buildPlan(local, previous).summary, nil
}

func Sync(ctx context.Context, localRoot string, cfg Config) (Summary, error) {
	localRoot, cfg, err := prepare(localRoot, cfg)
	if err != nil {
		return Summary{}, err
	}
	local, err := localFiles(localRoot)
	if err != nil {
		return Summary{}, err
	}
	previous, err := loadState(cfg.StatePath)
	if err != nil {
		return Summary{}, err
	}
	next := buildPlan(local, previous)

	conn, err := connect(ctx, cfg)
	if err != nil {
		return Summary{}, err
	}
	defer conn.close()

	if err := ensureRemoteRoot(conn.sftp, cfg.RemotePath); err != nil {
		return Summary{}, err
	}
	for _, file := range append(next.uploads, next.updates...) {
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		if err := uploadFile(conn.sftp, cfg.RemotePath, file); err != nil {
			return Summary{}, err
		}
	}
	remote, err := remoteFiles(conn.sftp, cfg.RemotePath)
	if err != nil {
		return Summary{}, err
	}
	next.deletes = remoteOnlyFiles(local, remote)
	next.summary.Deleted = len(next.deletes)
	next.summary.RemoteOnly = len(next.deletes)
	for _, file := range next.deletes {
		if err := ctx.Err(); err != nil {
			return Summary{}, err
		}
		remotePath, err := remoteFilePath(cfg.RemotePath, file.rel)
		if err != nil {
			return Summary{}, err
		}
		if err := conn.sftp.Remove(remotePath); err != nil && !isNotExist(err) {
			return Summary{}, fmt.Errorf("sftp delete remote file %s: %w", remotePath, err)
		}
	}
	if err := removeEmptyDirs(conn.sftp, cfg.RemotePath); err != nil {
		return Summary{}, err
	}
	if err := saveState(cfg.StatePath, nextState(local)); err != nil {
		return Summary{}, err
	}
	next.summary.OutOfSync = false
	return next.summary, nil
}

func Inspect(ctx context.Context, cfg Config) (Inspection, error) {
	cfg, err := prepareConnectionConfig(cfg)
	if err != nil {
		return Inspection{}, err
	}
	conn, err := connect(ctx, cfg)
	if err != nil {
		return Inspection{}, err
	}
	defer conn.close()
	if err := ensureRemoteRoot(conn.sftp, cfg.RemotePath); err != nil {
		return Inspection{}, err
	}
	files, err := remoteFiles(conn.sftp, cfg.RemotePath)
	if err != nil {
		return Inspection{}, err
	}
	return Inspection{RemoteFiles: len(files)}, nil
}

func VerifySecret(ctx context.Context, cfg Config) error {
	cfg, err := prepareConnectionConfig(cfg)
	if err != nil {
		return err
	}
	methods, cleanup, err := secretAuthMethods(cfg)
	if err != nil {
		return err
	}
	conn, err := connectWithMethods(ctx, cfg, methods, cleanup)
	if err != nil {
		return err
	}
	defer conn.close()
	return ensureRemoteRoot(conn.sftp, cfg.RemotePath)
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
	cfg, err = prepareConnectionConfig(cfg)
	if err != nil {
		return "", Config{}, err
	}
	cfg.StatePath = strings.TrimSpace(cfg.StatePath)
	if cfg.StatePath == "" {
		return "", Config{}, fmt.Errorf("%w: deploy state path is required", ErrInvalidConfig)
	}
	statePath, err := filepath.Abs(cfg.StatePath)
	if err != nil {
		return "", Config{}, err
	}
	cfg.StatePath = filepath.Clean(statePath)
	return filepath.Clean(absLocalRoot), cfg, nil
}

func prepareConnectionConfig(cfg Config) (Config, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.User = strings.TrimSpace(cfg.User)
	cfg.RemotePath = strings.TrimSpace(cfg.RemotePath)
	cfg.KeyPath = strings.TrimSpace(cfg.KeyPath)
	cfg.KnownHostsPath = strings.TrimSpace(cfg.KnownHostsPath)
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Host == "" {
		return Config{}, fmt.Errorf("%w: SFTP host is required", ErrInvalidConfig)
	}
	if cfg.User == "" {
		return Config{}, fmt.Errorf("%w: SFTP user is required", ErrInvalidConfig)
	}
	remotePath, err := cleanRemotePath(cfg.RemotePath)
	if err != nil {
		return Config{}, err
	}
	cfg.RemotePath = remotePath
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("%w: SFTP port must be between 1 and 65535", ErrInvalidConfig)
	}
	return cfg, nil
}

func cleanRemotePath(value string) (string, error) {
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
	if cleaned == "/" {
		return "", fmt.Errorf("%w: refusing to manage the remote root folder", ErrInvalidConfig)
	}
	return cleaned, nil
}

func connect(ctx context.Context, cfg Config) (*client, error) {
	methods, cleanup, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	return connectWithMethods(ctx, cfg, methods, cleanup)
}

func connectWithMethods(ctx context.Context, cfg Config, methods []ssh.AuthMethod, cleanup func()) (*client, error) {
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
	if deadline, ok := ctx.Deadline(); ok {
		_ = rawConn.SetDeadline(deadline)
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
	return &client{conn: rawConn, ssh: sshClient, sftp: sftpClient}, nil
}

func (c *client) close() {
	_ = c.sftp.Close()
	_ = c.ssh.Close()
	_ = c.conn.Close()
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
		signer, err := signerFromKeyFile(keyPath, cfg.Secret)
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
			return nil, closeAll(cleanup), fmt.Errorf("%w: encrypted private keys require a session passphrase or ssh-agent", ErrInvalidConfig)
		}
		if cfg.KeyPath != "" {
			return nil, closeAll(cleanup), err
		}
	}
	if cfg.Secret != "" {
		methods = append(methods, ssh.Password(cfg.Secret))
		methods = append(methods, ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = cfg.Secret
			}
			return answers, nil
		}))
	}
	if len(methods) == 0 {
		message := "no SSH auth methods available; load a key into ssh-agent"
		if encryptedKey {
			message += ", add the encrypted key to ssh-agent, or set a session passphrase"
		} else {
			message += ", configure an unencrypted key path, or set a session password"
		}
		return nil, closeAll(cleanup), fmt.Errorf("%w: %s", ErrInvalidConfig, message)
	}
	return methods, closeAll(cleanup), nil
}

func secretAuthMethods(cfg Config) ([]ssh.AuthMethod, func(), error) {
	if cfg.Secret == "" {
		return nil, func() {}, fmt.Errorf("%w: password or passphrase is required", ErrInvalidConfig)
	}
	if cfg.KeyPath != "" {
		signer, err := encryptedSignerFromConfiguredKey(cfg.KeyPath, cfg.Secret)
		if err != nil {
			return nil, func() {}, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, func() {}, nil
	}
	return []ssh.AuthMethod{
		ssh.Password(cfg.Secret),
		ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = cfg.Secret
			}
			return answers, nil
		}),
	}, func() {}, nil
}

func encryptedSignerFromConfiguredKey(keyPath string, secret string) (ssh.Signer, error) {
	keyPaths := candidateKeyPaths(keyPath)
	if len(keyPaths) != 1 {
		return nil, fmt.Errorf("%w: configured private key path is required", ErrInvalidConfig)
	}
	data, err := os.ReadFile(keyPaths[0])
	if err != nil {
		return nil, err
	}
	if _, err := ssh.ParsePrivateKey(data); err == nil {
		return nil, fmt.Errorf("%w: configured private key is not encrypted; no deploy secret is required", ErrInvalidConfig)
	} else if !isEncryptedKeyError(err) {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKeyWithPassphrase(data, []byte(secret))
	if err != nil {
		return nil, fmt.Errorf("%w: private key passphrase was rejected", ErrInvalidConfig)
	}
	return signer, nil
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

func signerFromKeyFile(keyPath string, secret string) (ssh.Signer, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err == nil {
		return signer, nil
	}
	if secret != "" && isEncryptedKeyError(err) {
		return ssh.ParsePrivateKeyWithPassphrase(data, []byte(secret))
	}
	return nil, err
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
		return nil, fmt.Errorf("sftp stat remote folder %s: %w", root, err)
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
		return fmt.Errorf("sftp read remote directory %s: %w", dir, err)
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

func loadState(statePath string) (map[string]remoteMeta, error) {
	files := map[string]remoteMeta{}
	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return files, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read deploy state %s: %w", statePath, err)
	}
	var state stateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode deploy state %s: %w", statePath, err)
	}
	for rel, entry := range state.Files {
		if !validRelativePath(rel) {
			continue
		}
		files[rel] = remoteMeta{
			rel:     rel,
			size:    entry.Size,
			modTime: time.Unix(0, entry.ModTimeUnixNano).UTC(),
		}
	}
	return files, nil
}

func saveState(statePath string, files map[string]remoteMeta) error {
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return fmt.Errorf("create deploy state folder %s: %w", filepath.Dir(statePath), err)
	}
	state := stateFile{
		Version: 1,
		Files:   make(map[string]stateEntry, len(files)),
	}
	for rel, file := range files {
		if !validRelativePath(rel) {
			continue
		}
		state.Files[rel] = stateEntry{
			Size:            file.size,
			ModTimeUnixNano: file.modTime.UnixNano(),
		}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode deploy state %s: %w", statePath, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(statePath, data, 0o600); err != nil {
		return fmt.Errorf("write deploy state %s: %w", statePath, err)
	}
	return nil
}

func validRelativePath(rel string) bool {
	return rel != "" &&
		!strings.HasPrefix(rel, "/") &&
		!strings.Contains(rel, "\x00") &&
		rel == path.Clean(rel) &&
		rel != "." &&
		!strings.HasPrefix(rel, "../") &&
		rel != ".."
}

func buildPlan(local map[string]fileMeta, remote map[string]remoteMeta) plan {
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
		next.deletes = append(next.deletes, remote[rel])
	}
	next.summary.Uploaded = len(next.uploads)
	next.summary.Updated = len(next.updates)
	next.summary.Deleted = len(next.deletes)
	next.summary.LocalFiles = len(local)
	next.summary.OutOfSync = next.summary.Uploaded > 0 || next.summary.Updated > 0 || next.summary.Deleted > 0
	return next
}

func sameFile(local fileMeta, remote remoteMeta) bool {
	return local.size == remote.size && local.modTime.Equal(remote.modTime)
}

func remoteOnlyFiles(local map[string]fileMeta, remote map[string]remoteMeta) []remoteMeta {
	remoteKeys := make([]string, 0, len(remote))
	for rel := range remote {
		remoteKeys = append(remoteKeys, rel)
	}
	sort.Strings(remoteKeys)
	files := make([]remoteMeta, 0)
	for _, rel := range remoteKeys {
		if _, ok := local[rel]; ok {
			continue
		}
		files = append(files, remote[rel])
	}
	return files
}

func nextState(local map[string]fileMeta) map[string]remoteMeta {
	state := map[string]remoteMeta{}
	for rel, file := range local {
		state[rel] = remoteMeta{
			rel:     rel,
			size:    file.size,
			modTime: file.modTime,
		}
	}
	return state
}

func uploadFile(client *sftp.Client, remoteRoot string, local fileMeta) error {
	remotePath, err := remoteFilePath(remoteRoot, local.rel)
	if err != nil {
		return err
	}
	if err := mkdirRelativeAll(client, remoteRoot, path.Dir(local.rel)); err != nil {
		return err
	}
	source, err := os.Open(local.local)
	if err != nil {
		return fmt.Errorf("open local file %s: %w", local.local, err)
	}
	defer source.Close()

	target, err := client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return fmt.Errorf("sftp open remote file %s: %w", remotePath, err)
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		return fmt.Errorf("sftp upload remote file %s: %w", remotePath, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("sftp close remote file %s: %w", remotePath, closeErr)
	}
	return nil
}

func ensureRemoteRoot(client *sftp.Client, remoteRoot string) error {
	info, err := client.Stat(remoteRoot)
	if err != nil {
		return fmt.Errorf("sftp remote folder must already exist %s: %w", remoteRoot, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: SFTP remote path must be a directory: %s", ErrInvalidConfig, remoteRoot)
	}
	return nil
}

func mkdirRelativeAll(client *sftp.Client, remoteRoot string, relDir string) error {
	relDir = path.Clean(relDir)
	if relDir == "." || relDir == "" {
		return nil
	}
	if !validRelativePath(relDir) {
		return fmt.Errorf("%w: invalid relative public folder %s", ErrInvalidConfig, relDir)
	}
	current := remoteRoot
	for _, part := range strings.Split(relDir, "/") {
		if part == "" {
			continue
		}
		current = path.Join(current, part)
		info, err := client.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("%w: SFTP remote path must be a directory: %s", ErrInvalidConfig, current)
			}
			continue
		}
		if !isNotExist(err) {
			return fmt.Errorf("sftp stat remote directory %s: %w", current, err)
		}
		if err := client.Mkdir(current); err != nil {
			return fmt.Errorf("sftp create remote directory %s: %w", current, err)
		}
	}
	return nil
}

func remoteFilePath(remoteRoot string, rel string) (string, error) {
	if !validRelativePath(rel) {
		return "", fmt.Errorf("%w: invalid relative public file %s", ErrInvalidConfig, rel)
	}
	return path.Join(remoteRoot, rel), nil
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
			return fmt.Errorf("sftp read remote directory %s: %w", dir, err)
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

func isExist(err error) bool {
	return errors.Is(err, os.ErrExist) || strings.Contains(strings.ToLower(err.Error()), "file exists")
}

func isNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "no such file")
}
