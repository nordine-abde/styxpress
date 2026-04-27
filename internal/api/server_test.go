package api

import (
	"bytes"
	"encoding/json"
	"errors"
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

	missingFeatured := authedRequest(t, server, http.MethodPost, "/api/featured", `{"slugs":["missing-post"]}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, missingFeatured)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing featured status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestEndToEndFixtureSiteLocalAndServerContentModes(t *testing.T) {
	tests := []struct {
		name        string
		storageMode string
	}{
		{name: "local content", storageMode: config.ContentStorageLocal},
		{name: "server content", storageMode: config.ContentStorageServer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, contentDir, publicDir := newTestServerWithMode(t, tt.storageMode)
			copyDir(t, filepath.Join("..", "..", "fixtures", "local-site", "content"), contentDir)

			detail := authedRequest(t, server, http.MethodGet, "/api/posts/hello-world", "")
			recorder := httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, detail)
			if recorder.Code != http.StatusOK {
				t.Fatalf("detail status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			var original postResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &original); err != nil {
				t.Fatalf("decode detail: %v", err)
			}

			update := authedRequest(t, server, http.MethodPost, "/api/posts/hello-world", `{
				"title":"Hello World Revised",
				"description":"Updated intro",
				"source":"# Hello World Revised\n\nEdited body.",
				"assets":["diagram.txt"]
			}`)
			recorder = httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, update)
			if recorder.Code != http.StatusOK {
				t.Fatalf("update status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			var edited postResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &edited); err != nil {
				t.Fatalf("decode edited post: %v", err)
			}
			if edited.PublishedAt != original.PublishedAt {
				t.Fatalf("PublishedAt = %q, want existing %q", edited.PublishedAt, original.PublishedAt)
			}
			if !strings.Contains(edited.Source, "Edited body") {
				t.Fatalf("edited source = %q, want updated body", edited.Source)
			}

			var gotConfig config.Config
			server.publishRunner = func(_ *http.Request, cfg config.Config, _ string) (publishing.Result, error) {
				gotConfig = cfg
				return publishing.Result{
					UploadedPaths: []string{
						"/srv/styxpress/public/index.html",
						"/srv/styxpress/public/posts/hello-world/index.html",
					},
				}, nil
			}

			publish := authedRequest(t, server, http.MethodPost, "/api/posts/hello-world/publish", `{}`)
			recorder = httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, publish)
			if recorder.Code != http.StatusOK {
				t.Fatalf("publish status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if gotConfig.ContentStorageMode != tt.storageMode {
				t.Fatalf("ContentStorageMode = %q, want %q", gotConfig.ContentStorageMode, tt.storageMode)
			}
			if gotConfig.ContentDir != contentDir || gotConfig.PublicDir != publicDir {
				t.Fatalf("publish config paths = %q %q, want %q %q", gotConfig.ContentDir, gotConfig.PublicDir, contentDir, publicDir)
			}

			for _, path := range []string{
				filepath.Join(publicDir, "index.html"),
				filepath.Join(publicDir, "feed.xml"),
				filepath.Join(publicDir, "sitemap.xml"),
				filepath.Join(publicDir, "posts", "hello-world", "index.html"),
				filepath.Join(publicDir, "posts", "hello-world", "assets", "diagram.txt"),
			} {
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("expected rendered file %s: %v", path, err)
				}
			}
		})
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

func TestPublishEndpointReportsCleanupPathsOnUploadFailure(t *testing.T) {
	server, _, _ := newTestServer(t)
	server.publishRunner = func(_ *http.Request, _ config.Config, _ string) (publishing.Result, error) {
		return publishing.Result{CleanupPaths: []string{"/srv/site/public/index.html"}}, &publishing.UploadError{
			Path:         "/srv/site/public/feed.xml",
			CleanupPaths: []string{"/srv/site/public/index.html", "/srv/site/public/feed.xml"},
			Err:          errors.New("create failed"),
		}
	}

	save := authedRequest(t, server, http.MethodPost, "/api/posts", `{
		"slug":"cleanup-message",
		"title":"Cleanup Message",
		"source":"# Cleanup"
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, save)
	if recorder.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	publish := authedRequest(t, server, http.MethodPost, "/api/posts/cleanup-message/publish", `{}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, publish)
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("publish status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != "publish_upload_failed" {
		t.Fatalf("error code = %q, want publish_upload_failed", response.Error.Code)
	}
	if !strings.Contains(response.Error.Message, "/srv/site/public/index.html") || !strings.Contains(response.Error.Message, "/srv/site/public/feed.xml") {
		t.Fatalf("error message = %q, want cleanup paths", response.Error.Message)
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
	return newTestServerWithMode(t, config.ContentStorageLocal)
}

func newTestServerWithMode(t *testing.T, storageMode string) (*Server, string, string) {
	t.Helper()
	root := t.TempDir()
	contentDir := filepath.Join(root, "content")
	publicDir := filepath.Join(root, "public")
	configPath := filepath.Join(root, "config.toml")
	cfg := config.Config{
		SiteBaseURL:        "https://blog.example.com",
		ContentDir:         contentDir,
		PublicDir:          publicDir,
		ContentStorageMode: storageMode,
		RemoteHost:         "example.com",
		RemoteUser:         "deploy",
		SSHKeyPath:         filepath.Join(root, "id_ed25519"),
		RemotePublicDir:    "/srv/site/public",
		RemoteContentDir:   "/srv/site/content",
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

func copyDir(t *testing.T, source string, destination string) {
	t.Helper()
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
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
