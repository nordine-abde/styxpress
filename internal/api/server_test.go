package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nordine-abde/styxpress/internal/config"
	"github.com/nordine-abde/styxpress/internal/content"
	"github.com/nordine-abde/styxpress/internal/siteconfig"
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

func TestConfigEndpointSavesLocalPathsOnly(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)

	request := authedRequest(t, server, http.MethodPost, "/api/config", `{
		"name":"Client Live",
		"contentDir":"`+escapeJSON(contentDir)+`",
		"publicDir":"`+escapeJSON(publicDir)+`",
		"deploy":{
			"enabled":false,
			"sftp":{
				"port":2222
			}
		}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var saved config.Config
	if err := json.Unmarshal(recorder.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if saved.Name != "Client Live" || saved.ContentDir != contentDir || saved.PublicDir != publicDir {
		t.Fatalf("config = %#v, want local paths", saved)
	}
	if saved.Deploy.Enabled || saved.Deploy.SFTP.Port != 2222 {
		t.Fatalf("deploy config = %#v, want disabled SFTP config without secrets", saved.Deploy)
	}

	for name, body := range map[string]string{
		"removed remote key": `{"remoteHost":"example.com","deploy":{"sftp":{"password":"secret"}}}`,
		"removed deploy mode": `{
			"contentDir":"` + escapeJSON(contentDir) + `",
			"publicDir":"` + escapeJSON(publicDir) + `",
			"deploy":{"mode":"auto"}
		}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := authedRequest(t, server, http.MethodPost, "/api/config", body)
			recorder := httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("removed field status = %d, body = %s; want 400", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestConfigEndpointRequiresSetupForFirstSFTPConfig(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)

	request := authedRequest(t, server, http.MethodPost, "/api/config", `{
		"contentDir":"`+escapeJSON(contentDir)+`",
		"publicDir":"`+escapeJSON(publicDir)+`",
		"deploy":{
			"enabled":true,
			"sftp":{
				"host":"example.com",
				"port":22,
				"user":"deploy",
				"remotePath":"/public_html"
			}
		}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("save status = %d, body = %s; want 409", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "deploy_setup_required") {
		t.Fatalf("body = %s, want deploy_setup_required", recorder.Body.String())
	}
}

func TestDeployEndpointsRequireEnabledSFTPConfig(t *testing.T) {
	server, _, _ := newTestServer(t)

	status := authedRequest(t, server, http.MethodGet, "/api/deploy/status", "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, status)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s; want 200", recorder.Code, recorder.Body.String())
	}
	var statusBody deployStatusResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if statusBody.Enabled || statusBody.Configured || statusBody.OutOfSync {
		t.Fatalf("status = %#v, want disabled deploy status", statusBody)
	}

	deploy := authedRequest(t, server, http.MethodPost, "/api/deploy", `{}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, deploy)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("deploy code = %d, body = %s; want 400", recorder.Code, recorder.Body.String())
	}
}

func TestDeployStatusUsesLocalStateWithoutSFTPConnection(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatalf("create public dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "index.html"), []byte("<h1>Hello</h1>"), 0o644); err != nil {
		t.Fatalf("write public file: %v", err)
	}
	if err := config.Save(server.configPath, config.Config{
		ContentDir: contentDir,
		PublicDir:  publicDir,
		Deploy: config.DeployConfig{
			Enabled: true,
			SFTP: config.SFTPConfig{
				Host:       "example.invalid",
				Port:       22,
				User:       "deploy",
				RemotePath: "/public_html",
			},
		},
	}); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	status := authedRequest(t, server, http.MethodGet, "/api/deploy/status", "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, status)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s; want 200", recorder.Code, recorder.Body.String())
	}
	var body deployStatusResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !body.OutOfSync || body.Summary == nil || body.Summary.Uploaded != 1 {
		t.Fatalf("status = %#v, want one local file pending upload", body)
	}
}

func TestDeploySecretRequiresVerifiedSFTPConfig(t *testing.T) {
	server, _, _ := newTestServer(t)

	save := authedRequest(t, server, http.MethodPost, "/api/deploy/secret", `{"secret":"session-passphrase"}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, save)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("save secret code = %d, body = %s; want 400", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "session-passphrase") {
		t.Fatalf("save secret response exposed secret: %s", recorder.Body.String())
	}

	status := authedRequest(t, server, http.MethodGet, "/api/deploy/status", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, status)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, body = %s; want 200", recorder.Code, recorder.Body.String())
	}
	var statusBody deployStatusResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &statusBody); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if statusBody.SecretSet {
		t.Fatalf("SecretSet = true, want false after rejected secret")
	}
	if strings.Contains(recorder.Body.String(), "session-passphrase") {
		t.Fatalf("status response exposed secret: %s", recorder.Body.String())
	}
}

func TestSiteRegistryUsesLocalConfig(t *testing.T) {
	store := config.NewSiteStore(t.TempDir())
	server, err := newServer("", store, nil)
	if err != nil {
		t.Fatalf("newServer returned error: %v", err)
	}
	root := t.TempDir()
	contentDir := filepath.Join(root, "blog-content")
	publicDir := filepath.Join(root, "blog-public")

	create := authedRequest(t, server, http.MethodPost, "/api/sites", `{
		"name":"My Blog",
		"config":{"contentDir":"`+escapeJSON(contentDir)+`","publicDir":"`+escapeJSON(publicDir)+`"}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, create)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	list := authedRequest(t, server, http.MethodGet, "/api/sites", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, list)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body sitesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode sites: %v", err)
	}
	if len(body.Sites) != 1 || body.Sites[0].Config.ContentDir != contentDir {
		t.Fatalf("sites response = %#v, want created local site", body)
	}
	assertFileExists(t, filepath.Join(contentDir, siteconfig.FileName))
	assertFileExists(t, filepath.Join(publicDir, "index.html"))
}

func TestSiteSuggestionUsesUniqueHomePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store := config.NewSiteStore(t.TempDir())
	server, err := newServer("", store, nil)
	if err != nil {
		t.Fatalf("newServer returned error: %v", err)
	}

	suggestion := authedRequest(t, server, http.MethodGet, "/api/sites/suggestion?name=My%20Blog", "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, suggestion)
	if recorder.Code != http.StatusOK {
		t.Fatalf("suggestion status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body config.Site
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode suggestion: %v", err)
	}
	if body.ID != "my-blog" || body.Config.PublicDir != filepath.Join(home, "Styxpress", "my-blog", "public") {
		t.Fatalf("suggestion = %#v, want normalized home public dir", body)
	}

	create := authedRequest(t, server, http.MethodPost, "/api/sites", `{"name":"My Blog"}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, create)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	nextSuggestion := authedRequest(t, server, http.MethodGet, "/api/sites/suggestion?name=My%20Blog", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, nextSuggestion)
	if recorder.Code != http.StatusOK {
		t.Fatalf("next suggestion status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode next suggestion: %v", err)
	}
	if body.ID != "my-blog-2" || body.Config.ContentDir != filepath.Join(home, "Styxpress", "my-blog-2", "content") {
		t.Fatalf("next suggestion = %#v, want unique home content dir", body)
	}
}

func TestPostWorkflowPreviewAndMediaEndpoints(t *testing.T) {
	server, contentDir, _ := newTestServer(t)

	save := authedRequest(t, server, http.MethodPost, "/api/posts", `{
		"slug":"hello-world",
		"title":"Hello World",
		"description":"Intro",
		"source":"# Hello\n\nBody",
		"publishedAt":"2026-04-01T09:30:00Z",
		"updatedAt":"2026-04-01T09:30:00Z",
		"publishStatus":"published"
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
	if listBody.Posts[0].PublishedAt != "" || listBody.Posts[0].PublishStatus != "draft" {
		t.Fatalf("list publish state = publishedAt %q status %q, want draft without publishedAt", listBody.Posts[0].PublishedAt, listBody.Posts[0].PublishStatus)
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
	if !strings.Contains(detailBody.Source, "# Hello") || detailBody.PublishStatus != "draft" {
		t.Fatalf("detail body = %#v, want draft with markdown source", detailBody)
	}

	uploadCover(t, server, "/api/posts/hello-world/cover", "cover.jpg", "cover image", "")
	if _, err := os.Stat(filepath.Join(contentDir, "posts", "hello-world", "cover.jpg")); err != nil {
		t.Fatalf("cover was not written: %v", err)
	}
	getCover := authedRequest(t, server, http.MethodGet, "/api/posts/hello-world/cover", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, getCover)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get cover status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "cover image" {
		t.Fatalf("cover body = %q, want cover image", recorder.Body.String())
	}
	uploadCover(t, server, "/api/posts/hello-world/assets", "asset.png", "asset body", "docs/asset.png")
	if _, err := os.Stat(filepath.Join(contentDir, "posts", "hello-world", "assets", "docs", "asset.png")); err != nil {
		t.Fatalf("asset was not written: %v", err)
	}
	getAsset := authedRequest(t, server, http.MethodGet, "/api/posts/hello-world/assets/docs/asset.png", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, getAsset)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get asset status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Body.String() != "asset body" {
		t.Fatalf("asset body = %q, want asset body", recorder.Body.String())
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
}

func TestMediaEndpointsRejectSymlinkedPostDirectory(t *testing.T) {
	server, contentDir, _ := newTestServer(t)
	outsideRoot := t.TempDir()
	outsideRepo := content.NewRepository(outsideRoot)
	writePost(t, outsideRepo, content.Post{Slug: "hello-world", Title: "Hello", Source: "Body"})
	if err := outsideRepo.WriteCover("hello-world", "cover.jpg", strings.NewReader("secret cover")); err != nil {
		t.Fatalf("WriteCover returned error: %v", err)
	}
	if err := outsideRepo.WriteAsset("hello-world", "leak.png", strings.NewReader("secret asset")); err != nil {
		t.Fatalf("WriteAsset returned error: %v", err)
	}

	postsDir := filepath.Join(contentDir, "posts")
	if err := os.MkdirAll(postsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll posts returned error: %v", err)
	}
	if err := os.Symlink(filepath.Join(outsideRoot, "posts", "hello-world"), filepath.Join(postsDir, "hello-world")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	assertMediaRequestBlocked(t, server, "/api/posts/hello-world/cover", "secret cover")
	assertMediaRequestBlocked(t, server, "/api/posts/hello-world/assets/leak.png", "secret asset")
}

func TestCoverEndpointRejectsSymlinkCoverFile(t *testing.T) {
	server, contentDir, _ := newTestServer(t)
	repo := content.NewRepository(contentDir)
	writePost(t, repo, content.Post{Slug: "hello-world", Title: "Hello", Source: "Body"})

	secretPath := filepath.Join(t.TempDir(), "secret-cover.jpg")
	if err := os.WriteFile(secretPath, []byte("secret cover"), 0o644); err != nil {
		t.Fatalf("WriteFile secret cover returned error: %v", err)
	}
	coverPath := filepath.Join(contentDir, "posts", "hello-world", "cover.jpg")
	if err := os.Symlink(secretPath, coverPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	assertMediaRequestBlocked(t, server, "/api/posts/hello-world/cover", "secret cover")
}

func TestSiteConfigEndpointSavesUnderConfiguredContentDir(t *testing.T) {
	server, contentDir, _ := newTestServer(t)

	request := authedRequest(t, server, http.MethodPost, "/api/site-config", `{
		"title":"My Blog",
		"description":"Local notes",
		"header":{"links":[{"label":"Home","href":"/"}]},
		"footer":{"text":"Local files","showWatermark":false,"links":[{"label":"RSS","href":"/feed.xml"}]}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("save site config status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	cfg, err := siteconfig.Load(contentDir)
	if err != nil {
		t.Fatalf("load saved site config: %v", err)
	}
	if cfg.Title != "My Blog" || cfg.Footer.ShowWatermark {
		t.Fatalf("saved site config = %#v, want title and disabled watermark", cfg)
	}

	preview := authedRequest(t, server, http.MethodPost, "/api/site-config/preview", `{
		"title":"Preview Blog",
		"description":"",
		"header":{"links":[]},
		"footer":{"text":"","showWatermark":true,"links":[]}
	}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, preview)
	if recorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body previewResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if !strings.Contains(body.HTML, "Preview Blog") || !strings.Contains(body.HTML, `Published with <a href="https://styxpress.anordine.com">Styxpress</a>`) {
		t.Fatalf("preview html = %s, want title and watermark", body.HTML)
	}
}

func TestSiteFaviconUploadAndReset(t *testing.T) {
	server, contentDir, _ := newTestServer(t)

	uploadCover(t, server, "/api/site-config/favicon", "brand.ico", "ico body", "")
	cfg, err := siteconfig.Load(contentDir)
	if err != nil {
		t.Fatalf("load site config: %v", err)
	}
	if cfg.Favicon != "assets/brand.ico" {
		t.Fatalf("Favicon = %q, want assets/brand.ico", cfg.Favicon)
	}
	data, err := os.ReadFile(filepath.Join(contentDir, "assets", "brand.ico"))
	if err != nil {
		t.Fatalf("read uploaded favicon: %v", err)
	}
	if string(data) != "ico body" {
		t.Fatalf("uploaded favicon = %q, want ico body", data)
	}

	request := authedRequest(t, server, http.MethodDelete, "/api/site-config/favicon", "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("reset favicon status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	cfg, err = siteconfig.Load(contentDir)
	if err != nil {
		t.Fatalf("load reset site config: %v", err)
	}
	if cfg.Favicon != siteconfig.DefaultFaviconPath {
		t.Fatalf("Favicon after reset = %q, want %q", cfg.Favicon, siteconfig.DefaultFaviconPath)
	}
}

func TestRenderAndPublishEndpointsUseLocalOutput(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)
	repo := content.NewRepository(contentDir)
	publishedAt := mustTime(t, "2026-04-01T09:30:00Z")
	writePost(t, repo, content.Post{
		Slug:        "alpha",
		Title:       "Alpha",
		Source:      "# Alpha",
		PublishedAt: publishedAt,
		UpdatedAt:   publishedAt,
	})
	writePost(t, repo, content.Post{
		Slug:   "publish-me",
		Title:  "Publish Me",
		Source: "# Publish Me",
	})

	renderSite := authedRequest(t, server, http.MethodPost, "/api/site/render", `{}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, renderSite)
	if recorder.Code != http.StatusOK {
		t.Fatalf("render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var renderBody renderSiteResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &renderBody); err != nil {
		t.Fatalf("decode render response: %v", err)
	}
	if len(renderBody.Posts) != 1 || renderBody.Posts[0].Slug != "alpha" {
		t.Fatalf("render response = %#v, want one published post", renderBody)
	}
	if _, err := os.Stat(filepath.Join(publicDir, "posts", "alpha", "index.html")); err != nil {
		t.Fatalf("alpha output missing: %v", err)
	}

	renderDraft := authedRequest(t, server, http.MethodPost, "/api/posts/publish-me/render", `{}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, renderDraft)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("render draft status = %d, body = %s; want %d", recorder.Code, recorder.Body.String(), http.StatusBadRequest)
	}
	if _, err := os.Stat(filepath.Join(contentDir, "posts", "publish-me", "published_at.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("published_at.txt stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(publicDir, "posts", "publish-me", "index.html")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("draft output stat error = %v, want not exist", err)
	}

	renderPost := authedRequest(t, server, http.MethodPost, "/api/posts/alpha/render", `{}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, renderPost)
	if recorder.Code != http.StatusOK {
		t.Fatalf("render post status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	publishPost := authedRequest(t, server, http.MethodPost, "/api/posts/publish-me/publish", `{}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, publishPost)
	if recorder.Code != http.StatusOK {
		t.Fatalf("publish status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Stat(filepath.Join(publicDir, "posts", "publish-me", "index.html")); err != nil {
		t.Fatalf("publish-me output missing: %v", err)
	}
	detail := authedRequest(t, server, http.MethodGet, "/api/posts/publish-me", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, detail)
	if recorder.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var post postResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &post); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	if post.PublishedAt == "" || post.PublishStatus != "published" {
		t.Fatalf("post = %#v, want local published status", post)
	}
}

func TestInvalidConfiguredPathReturnsBadRequest(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	if err := config.Save(configPath, config.Config{ContentDir: "", PublicDir: "public"}); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("content_dir = \"bad\x00\"\npublic_dir = \"public\"\nname = \"\"\n"), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	server, err := New(configPath, nil)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}

	request := authedRequest(t, server, http.MethodGet, "/api/posts", "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want 400", recorder.Code, recorder.Body.String())
	}
}

func TestSaveConfigRejectsOverlappingContentAndPublicDirs(t *testing.T) {
	server, contentDir, _ := newTestServer(t)

	request := authedRequest(t, server, http.MethodPost, "/api/config", fmt.Sprintf(`{
		"name": "Unsafe",
		"contentDir": %q,
		"publicDir": %q,
		"deploy": {"sftp": {"port": 22}}
	}`, contentDir, contentDir))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want %d", recorder.Code, recorder.Body.String(), http.StatusBadRequest)
	}
}

func TestUploadCoverNormalizesUploadedFilename(t *testing.T) {
	server, contentDir, _ := newTestServer(t)
	repo, err := server.repository()
	if err != nil {
		t.Fatalf("repository: %v", err)
	}
	writePost(t, repo, content.Post{Slug: "hello-world", Title: "Hello", Source: "Body"})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "hero-photo.JPG")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("jpg")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/posts/hello-world/cover", &body)
	request.Header.Set(SessionHeader, server.Token())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s; want %d", recorder.Code, recorder.Body.String(), http.StatusOK)
	}
	var response map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["cover"] != "cover.jpg" {
		t.Fatalf("cover response = %q, want cover.jpg", response["cover"])
	}
	data, err := os.ReadFile(filepath.Join(contentDir, "posts", "hello-world", "cover.jpg"))
	if err != nil {
		t.Fatalf("ReadFile cover.jpg returned error: %v", err)
	}
	if string(data) != "jpg" {
		t.Fatalf("cover.jpg = %q, want jpg", data)
	}
}

func TestUploadRejectsTraversalFilename(t *testing.T) {
	server, _, _ := newTestServer(t)
	repo, err := server.repository()
	if err != nil {
		t.Fatalf("repository: %v", err)
	}
	writePost(t, repo, content.Post{Slug: "hello-world", Title: "Hello", Source: "Body"})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "../secret.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("secret")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/posts/hello-world/cover", &body)
	request.Header.Set(SessionHeader, server.Token())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want %d", recorder.Code, recorder.Body.String(), http.StatusBadRequest)
	}
}

func TestUploadAssetRejectsNonImage(t *testing.T) {
	server, _, _ := newTestServer(t)
	repo, err := server.repository()
	if err != nil {
		t.Fatalf("repository: %v", err)
	}
	writePost(t, repo, content.Post{Slug: "hello-world", Title: "Hello", Source: "Body"})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "notes.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("notes")); err != nil {
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
		ContentDir: contentDir,
		PublicDir:  publicDir,
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

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) returned error: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want file", path)
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

func uploadCover(t *testing.T, server *Server, path string, filename string, value string, assetPath string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if assetPath != "" {
		if err := writer.WriteField("path", assetPath); err != nil {
			t.Fatalf("WriteField: %v", err)
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte(value)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set(SessionHeader, server.Token())
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("upload status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func assertMediaRequestBlocked(t *testing.T, server *Server, path string, secret string) {
	t.Helper()
	request := authedRequest(t, server, http.MethodGet, path, "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s; want %d", recorder.Code, recorder.Body.String(), http.StatusBadRequest)
	}
	if strings.Contains(recorder.Body.String(), secret) {
		t.Fatalf("response body leaked symlink target content: %s", recorder.Body.String())
	}
}

func writePost(t *testing.T, repo *content.Repository, post content.Post) {
	t.Helper()
	if _, err := repo.WritePost(post, content.WritePostOptions{}); err != nil {
		t.Fatalf("WritePost(%s) returned error: %v", post.Slug, err)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func escapeJSON(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return strings.Trim(string(data), `"`)
}
