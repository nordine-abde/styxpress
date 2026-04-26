package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nordine-abde/styxpress/internal/config"
	"github.com/nordine-abde/styxpress/internal/publishing"
)

func TestAPIRequiresSessionToken(t *testing.T) {
	server, err := New(filepath.Join(t.TempDir(), "config.toml"), nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestAPIAcceptsSessionHeader(t *testing.T) {
	server, err := New(filepath.Join(t.TempDir(), "config.toml"), nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	request.Header.Set(SessionHeader, server.Token())

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestTestSSHUsesOptionalPassphrase(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	cfg := config.Config{
		RemoteHost:      "example.com",
		RemoteUser:      "deploy",
		SSHKeyPath:      "/tmp/id_ed25519",
		RemotePublicDir: "/srv/site/public",
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	server, err := New(configPath, nil)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	var gotPassphrase string
	server.sshTester = func(_ *http.Request, got config.Config, passphrase string) error {
		gotPassphrase = passphrase
		if got.RemoteHost != cfg.RemoteHost {
			t.Fatalf("RemoteHost = %q, want %q", got.RemoteHost, cfg.RemoteHost)
		}
		return nil
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/test-ssh", bytes.NewBufferString(`{"passphrase":"secret"}`))
	request.Header.Set(SessionHeader, server.Token())

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if gotPassphrase != "secret" {
		t.Fatalf("passphrase = %q, want secret", gotPassphrase)
	}
}

func TestPostWorkflowPreviewAndFeaturedEndpoints(t *testing.T) {
	server, contentDir, _ := newTestServer(t)

	save := authedRequest(t, server, http.MethodPost, "/api/posts", `{
		"slug":"hello-world",
		"title":"Hello World",
		"description":"Intro",
		"source":"# Hello\n\nBody"
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, save)
	if recorder.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	list := authedRequest(t, server, http.MethodGet, "/api/posts", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, list)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var listBody postListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listBody.Posts) != 1 || listBody.Posts[0].Slug != "hello-world" || listBody.Posts[0].Source != "" {
		t.Fatalf("list body = %#v, want summary without source", listBody)
	}

	detail := authedRequest(t, server, http.MethodGet, "/api/posts/hello-world", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, detail)
	if recorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var detailBody postResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &detailBody); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if !strings.Contains(detailBody.Source, "# Hello") {
		t.Fatalf("detail source = %q, want markdown source", detailBody.Source)
	}

	preview := authedRequest(t, server, http.MethodPost, "/api/render-preview", `{
		"slug":"hello-world",
		"title":"Hello World",
		"source":"# Preview"
	}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, preview)
	if recorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var previewBody previewResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &previewBody); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if !strings.Contains(previewBody.HTML, "<h1>Preview</h1>") {
		t.Fatalf("preview body = %s, want rendered markdown", previewBody.HTML)
	}

	featured := authedRequest(t, server, http.MethodPost, "/api/featured", `{"slugs":["hello-world"]}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, featured)
	if recorder.Code != http.StatusOK {
		t.Fatalf("featured status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Stat(filepath.Join(contentDir, "featured.txt")); err != nil {
		t.Fatalf("featured.txt was not written: %v", err)
	}
}

func TestPublishEndpointRendersAndPublishesConfiguredPaths(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)
	var gotPassphrase string
	var gotConfig config.Config
	server.publishRunner = func(_ *http.Request, cfg config.Config, passphrase string) (publishing.Result, error) {
		gotConfig = cfg
		gotPassphrase = passphrase
		return publishing.Result{UploadedPaths: []string{"/srv/site/public/index.html"}}, nil
	}

	save := authedRequest(t, server, http.MethodPost, "/api/posts", `{
		"slug":"publish-me",
		"title":"Publish Me",
		"source":"# Published"
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, save)
	if recorder.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	publish := authedRequest(t, server, http.MethodPost, "/api/posts/publish-me/publish", `{"passphrase":"secret"}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, publish)
	if recorder.Code != http.StatusOK {
		t.Fatalf("publish status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if gotPassphrase != "secret" {
		t.Fatalf("passphrase = %q, want secret", gotPassphrase)
	}
	if gotConfig.ContentDir != contentDir || gotConfig.PublicDir != publicDir {
		t.Fatalf("publish config paths = %q %q, want %q %q", gotConfig.ContentDir, gotConfig.PublicDir, contentDir, publicDir)
	}
	if _, err := os.Stat(filepath.Join(publicDir, "posts", "publish-me", "index.html")); err != nil {
		t.Fatalf("post index was not rendered: %v", err)
	}
	if _, err := os.Stat(filepath.Join(publicDir, "feed.xml")); err != nil {
		t.Fatalf("feed was not rendered: %v", err)
	}
}

func TestAssetUploadRejectsTraversalPath(t *testing.T) {
	server, _, _ := newTestServer(t)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("path", "../secret.txt"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	part, err := writer.CreateFormFile("file", "secret.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("secret")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/posts/hello-world/assets", &body)
	request.Header.Set(SessionHeader, server.Token())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want %d", recorder.Code, recorder.Body.String(), http.StatusBadRequest)
	}
}

func newTestServer(t *testing.T) (*Server, string, string) {
	t.Helper()
	root := t.TempDir()
	contentDir := filepath.Join(root, "content")
	publicDir := filepath.Join(root, "public")
	configPath := filepath.Join(root, "config.toml")
	cfg := config.Config{
		SiteBaseURL:        "https://blog.example.com",
		ContentDir:         contentDir,
		PublicDir:          publicDir,
		ContentStorageMode: config.ContentStorageLocal,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "/srv/site/public",
	}
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	server, err := New(configPath, nil)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}
	return server, contentDir, publicDir
}

func authedRequest(t *testing.T, server *Server, method string, path string, body string) *http.Request {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set(SessionHeader, server.Token())
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}
