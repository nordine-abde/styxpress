package publishing

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

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

func writeTestFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
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

func (c *fakeClient) Close() error {
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
