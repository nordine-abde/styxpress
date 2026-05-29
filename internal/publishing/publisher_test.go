package publishing

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/nordine-abde/styxpress/internal/config"
)

func TestPublishUploadsPublicOnlyForLocalContent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "index.html"), "home")
	writeTestFile(t, filepath.Join(root, "public", "posts", "first", "index.html"), "post")
	writeTestFile(t, filepath.Join(root, "content", "posts", "first", "source.md"), "# post")

	client := newFakeClient()
	publisher := New(config.Config{
		PublicDir:          filepath.Join(root, "public"),
		ContentDir:         filepath.Join(root, "content"),
		ContentStorageMode: config.ContentStorageLocal,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "/srv/site/public",
		RemoteContentDir:   "/srv/site/content",
	}, fakeDialer{client: client})

	result, err := publisher.Publish(context.Background(), Options{Passphrase: "secret"})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	wantUploaded := []string{
		"/srv/site/public/index.html",
		"/srv/site/public/posts/first/index.html",
	}
	if !reflect.DeepEqual(result.UploadedPaths, wantUploaded) {
		t.Fatalf("UploadedPaths = %#v, want %#v", result.UploadedPaths, wantUploaded)
	}
	if _, ok := client.files["/srv/site/content/posts/first/source.md"]; ok {
		t.Fatal("content file was uploaded in local content storage mode")
	}
	if !reflect.DeepEqual(result.CleanupPaths, wantUploaded) {
		t.Fatalf("CleanupPaths = %#v, want %#v", result.CleanupPaths, wantUploaded)
	}
}

func TestPublishUploadsContentForServerContent(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "index.html"), "home")
	writeTestFile(t, filepath.Join(root, "content", "featured.txt"), "first\n")
	writeTestFile(t, filepath.Join(root, "content", "posts", "first", "source.md"), "# post")

	client := newFakeClient()
	publisher := New(config.Config{
		PublicDir:          filepath.Join(root, "public"),
		ContentDir:         filepath.Join(root, "content"),
		ContentStorageMode: config.ContentStorageServer,
		RemoteHost:         "example.com:2222",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "/srv/site/public",
		RemoteContentDir:   "/srv/site/content",
	}, fakeDialer{client: client})

	result, err := publisher.Publish(context.Background(), Options{})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	wantUploaded := []string{
		"/srv/site/public/index.html",
		"/srv/site/content/featured.txt",
		"/srv/site/content/posts/first/source.md",
	}
	sort.Strings(result.UploadedPaths)
	sort.Strings(wantUploaded)
	if !reflect.DeepEqual(result.UploadedPaths, wantUploaded) {
		t.Fatalf("UploadedPaths = %#v, want %#v", result.UploadedPaths, wantUploaded)
	}
}

func TestPublishReturnsCleanupPathsOnFailedUpload(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "a.txt"), "a")
	writeTestFile(t, filepath.Join(root, "public", "b.txt"), "b")

	client := newFakeClient()
	client.failCreate = "/remote/public/b.txt"
	publisher := New(config.Config{
		PublicDir:          filepath.Join(root, "public"),
		ContentStorageMode: config.ContentStorageLocal,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "/remote/public",
	}, fakeDialer{client: client})

	result, err := publisher.Publish(context.Background(), Options{})
	if err == nil {
		t.Fatal("Publish returned nil error")
	}
	var uploadErr *UploadError
	if !errors.As(err, &uploadErr) {
		t.Fatalf("error type = %T, want *UploadError", err)
	}
	wantCleanup := []string{"/remote/public/a.txt", "/remote/public/b.txt"}
	if !reflect.DeepEqual(uploadErr.CleanupPaths, wantCleanup) {
		t.Fatalf("UploadError.CleanupPaths = %#v, want %#v", uploadErr.CleanupPaths, wantCleanup)
	}
	if !reflect.DeepEqual(result.CleanupPaths, []string{"/remote/public/a.txt"}) {
		t.Fatalf("Result.CleanupPaths = %#v, want uploaded cleanup path only", result.CleanupPaths)
	}
}

func TestPublishRejectsRemoteTraversal(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "index.html"), "home")

	publisher := New(config.Config{
		PublicDir:          filepath.Join(root, "public"),
		ContentStorageMode: config.ContentStorageLocal,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "../public",
	}, fakeDialer{client: newFakeClient()})

	if _, err := publisher.Publish(context.Background(), Options{}); !errors.Is(err, ErrInvalidPublishConfig) {
		t.Fatalf("Publish error = %v, want ErrInvalidPublishConfig", err)
	}
}

func TestVerifyRemoteReportsMatchedMissingAndDifferentFiles(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "index.html"), "same")
	writeTestFile(t, filepath.Join(root, "public", "feed.xml"), "feed")
	writeTestFile(t, filepath.Join(root, "public", "sitemap.xml"), "abc")
	writeTestFile(t, filepath.Join(root, "public", "assets", "styxpress.css"), "short")

	client := newFakeClient()
	client.files["/srv/site/public/index.html"] = "same"
	client.files["/srv/site/public/sitemap.xml"] = "abd"
	client.files["/srv/site/public/assets/styxpress.css"] = "longer"
	client.files["/srv/site/public/posts/draft/index.html"] = "stale draft"
	publisher := New(config.Config{
		PublicDir:          filepath.Join(root, "public"),
		ContentStorageMode: config.ContentStorageLocal,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "/srv/site/public",
	}, fakeDialer{client: client})

	result, err := publisher.VerifyRemote(context.Background(), Options{
		RemoteOnlyPaths: []string{
			"posts/draft/index.html",
			"posts/missing-draft/index.html",
		},
	})
	if err != nil {
		t.Fatalf("VerifyRemote returned error: %v", err)
	}

	byPath := verificationByRelativePath(result.Files)
	if got := byPath["index.html"]; got.Status != VerificationStatusPublished || got.LocalSHA256 == "" || got.RemoteSHA256 == "" {
		t.Fatalf("index.html verification = %#v, want published with hashes", got)
	}
	if got := byPath["feed.xml"]; got.Status != VerificationStatusNotOnRemote {
		t.Fatalf("feed.xml verification = %#v, want not on remote", got)
	}
	if got := byPath["sitemap.xml"]; got.Status != VerificationStatusChangesPending || got.LocalSHA256 == "" || got.RemoteSHA256 == "" || got.LocalSHA256 == got.RemoteSHA256 {
		t.Fatalf("sitemap.xml verification = %#v, want hash mismatch", got)
	}
	if got := byPath["assets/styxpress.css"]; got.Status != VerificationStatusChangesPending || got.LocalSize == got.RemoteSize || got.LocalSHA256 != "" || got.RemoteSHA256 != "" {
		t.Fatalf("stylesheet verification = %#v, want size mismatch without hashes", got)
	}
	if got := byPath["posts/draft/index.html"]; got.Status != VerificationStatusStillOnRemote || got.RemoteSize == 0 {
		t.Fatalf("draft verification = %#v, want still on remote", got)
	}
	if _, ok := byPath["posts/missing-draft/index.html"]; ok {
		t.Fatalf("missing draft verification was reported; want absent remote-only paths skipped")
	}

	wantSummary := VerificationSummary{
		Total:          5,
		Published:      1,
		ChangesPending: 2,
		NotOnRemote:    1,
		StillOnRemote:  1,
	}
	if result.Summary != wantSummary {
		t.Fatalf("summary = %#v, want %#v", result.Summary, wantSummary)
	}
}

func TestVerifyRemoteReportsUnknownOnRemoteReadError(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "public", "index.html"), "same")

	client := newFakeClient()
	client.files["/srv/site/public/index.html"] = "same"
	client.failOpen = "/srv/site/public/index.html"
	publisher := New(config.Config{
		PublicDir:          filepath.Join(root, "public"),
		ContentStorageMode: config.ContentStorageLocal,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "/srv/site/public",
	}, fakeDialer{client: client})

	result, err := publisher.VerifyRemote(context.Background(), Options{})
	if err != nil {
		t.Fatalf("VerifyRemote returned error: %v", err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("files = %#v, want one verification", result.Files)
	}
	if result.Files[0].Status != VerificationStatusUnknown || result.Files[0].Error == "" {
		t.Fatalf("verification = %#v, want unknown with error", result.Files[0])
	}
	if result.Summary.Unknown != 1 {
		t.Fatalf("summary = %#v, want one unknown", result.Summary)
	}
}

func writeTestFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func verificationByRelativePath(files []FileVerification) map[string]FileVerification {
	byPath := make(map[string]FileVerification, len(files))
	for _, file := range files {
		byPath[file.RelativePath] = file
	}
	return byPath
}

type fakeDialer struct {
	client *fakeClient
	cfg    SSHConfig
}

func (d fakeDialer) Dial(_ context.Context, cfg SSHConfig) (Client, error) {
	d.cfg = cfg
	return d.client, nil
}

type fakeClient struct {
	dirs       map[string]bool
	files      map[string]string
	failCreate string
	failOpen   string
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		dirs:  make(map[string]bool),
		files: make(map[string]string),
	}
}

func (c *fakeClient) MkdirAll(path string) error {
	c.dirs[path] = true
	return nil
}

func (c *fakeClient) Create(path string) (RemoteFile, error) {
	if path == c.failCreate {
		return nil, errors.New("create failed")
	}
	return &fakeFile{
		close: func(data string) {
			c.files[path] = data
		},
	}, nil
}

func (c *fakeClient) Stat(path string) (os.FileInfo, error) {
	if _, ok := c.dirs[path]; ok {
		return fakeFileInfo{name: filepath.Base(path), mode: fs.ModeDir | 0o755}, nil
	}
	data, ok := c.files[path]
	if !ok {
		return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}
	return fakeFileInfo{name: filepath.Base(path), size: int64(len(data)), mode: 0o644}, nil
}

func (c *fakeClient) Open(path string) (RemoteReader, error) {
	if path == c.failOpen {
		return nil, errors.New("open failed")
	}
	data, ok := c.files[path]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}
	return io.NopCloser(bytes.NewBufferString(data)), nil
}

func (c *fakeClient) Close() error {
	return nil
}

type fakeFileInfo struct {
	name string
	size int64
	mode fs.FileMode
}

func (i fakeFileInfo) Name() string {
	return i.name
}

func (i fakeFileInfo) Size() int64 {
	return i.size
}

func (i fakeFileInfo) Mode() fs.FileMode {
	return i.mode
}

func (i fakeFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (i fakeFileInfo) IsDir() bool {
	return i.mode.IsDir()
}

func (i fakeFileInfo) Sys() any {
	return nil
}

type fakeFile struct {
	bytes.Buffer
	close func(data string)
}

func (f *fakeFile) Close() error {
	f.close(f.String())
	return nil
}

var _ io.Writer = (*fakeFile)(nil)
