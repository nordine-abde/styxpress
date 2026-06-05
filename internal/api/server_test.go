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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nordine-abde/styxpress/internal/config"
	"github.com/nordine-abde/styxpress/internal/content"
	"github.com/nordine-abde/styxpress/internal/publishing"
	"github.com/nordine-abde/styxpress/internal/rendering"
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

func TestTestSSHUsesActiveSiteConfig(t *testing.T) {
	store := config.NewSiteStore(t.TempDir())
	site, err := store.Create(config.Config{
		Name:            "Remote Site",
		RemoteHost:      "active.example.com",
		RemoteUser:      "deploy",
		SSHKeyPath:      "/tmp/id_ed25519",
		RemotePublicDir: "/srv/site/public",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	server, err := newServer("", store, nil)
	if err != nil {
		t.Fatalf("newServer returned error: %v", err)
	}
	server.sshTester = func(_ *http.Request, got config.Config, _ string) error {
		if got.RemoteHost != site.Config.RemoteHost {
			t.Fatalf("RemoteHost = %q, want %q", got.RemoteHost, site.Config.RemoteHost)
		}
		return nil
	}

	recorder := httptest.NewRecorder()
	request := authedRequest(t, server, http.MethodPost, "/api/test-ssh", `{}`)

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestPostWorkflowPreviewAndFeaturedEndpoints(t *testing.T) {
	server, contentDir, _ := newTestServer(t)

	save := authedRequest(t, server, http.MethodPost, "/api/posts", `{
		"slug":"hello-world",
		"title":"Hello World",
		"description":"Intro",
		"source":"# Hello\n\nBody",
		"publishedAt":"2026-04-01T09:30:00Z",
		"updatedAt":"2026-04-01T09:30:00Z",
		"syncedAt":"2026-04-01T09:35:00Z",
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
	if !strings.Contains(detailBody.Source, "# Hello") {
		t.Fatalf("detail source = %q, want markdown source", detailBody.Source)
	}
	if detailBody.PublishedAt != "" || detailBody.SyncedAt != "" || detailBody.PublishStatus != "draft" {
		t.Fatalf("detail publish state = %#v, want draft without sync", detailBody)
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

func TestSiteConfigEndpointSavesUnderConfiguredContentDir(t *testing.T) {
	server, contentDir, _ := newTestServer(t)

	save := authedRequest(t, server, http.MethodPost, "/api/site-config", `{
		"title":"Anordine",
		"description":"Software notes",
		"theme":{"palette":"sage","font":"serif","layout":"wide","radius":"soft"},
		"header":{
			"variant":"centered",
			"title":"Anordine Lab",
			"tagline":"Local-first publishing",
			"links":[{"label":"Home","href":"/"},{"label":"RSS","href":"/feed.xml"}]
		},
		"footer":{
			"variant":"links",
			"text":"Built from Markdown files.",
			"links":[{"label":"Email","href":"mailto:hello@example.com"}]
		}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, save)
	if recorder.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Stat(filepath.Join(contentDir, siteconfig.FileName)); err != nil {
		t.Fatalf("site config was not written: %v", err)
	}

	load := authedRequest(t, server, http.MethodGet, "/api/site-config", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, load)
	if recorder.Code != http.StatusOK {
		t.Fatalf("load status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body siteconfig.Config
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode site config: %v", err)
	}
	if body.Title != "Anordine" || body.Theme.Palette != siteconfig.PaletteSage || body.Header.Variant != siteconfig.HeaderCentered {
		t.Fatalf("site config = %#v, want saved values", body)
	}
}

func TestSiteConfigPreviewEndpointReturnsHomepageWithoutWritingPublicFiles(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)
	writePublishedPost(t, contentDir, content.Post{
		Slug:        "preview-style",
		Title:       "Preview Style",
		Description: "Styled homepage entry",
		Source:      "# Preview Style",
		PublishedAt: mustTime(t, "2026-04-01T09:30:00Z"),
		UpdatedAt:   mustTime(t, "2026-04-01T09:30:00Z"),
	})

	preview := authedRequest(t, server, http.MethodPost, "/api/site-config/preview", `{
		"title":"Styled Site",
		"description":"Preview config",
		"theme":{
			"palette":"midnight",
			"font":"mono",
			"layout":"wide",
			"radius":"none",
			"customCss":".site-main { outline: 2px solid lime; }"
		}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, preview)
	if recorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body previewResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	for _, expected := range []string{
		`<title>Styled Site</title>`,
		`<body class="theme-midnight font-mono layout-wide radius-none">`,
		`<style>`,
		`.site-main { outline: 2px solid lime; }`,
		`<a href="/posts/preview-style/">Preview Style</a>`,
	} {
		if !strings.Contains(body.HTML, expected) {
			t.Fatalf("expected %q in preview HTML:\n%s", expected, body.HTML)
		}
	}
	if strings.Contains(body.HTML, `<link rel="stylesheet" href="/assets/styxpress.css">`) {
		t.Fatalf("preview should inline stylesheet instead of linking public CSS:\n%s", body.HTML)
	}
	if _, err := os.Stat(filepath.Join(publicDir, "assets", "styxpress.css")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview should not write stylesheet, stat err: %v", err)
	}
}

func TestSiteConfigStyleCSSEndpointReturnsCSSPayloadWithoutWritingPublicFiles(t *testing.T) {
	server, _, publicDir := newTestServer(t)

	request := authedRequest(t, server, http.MethodPost, "/api/site-config/style-css", `{
		"title":"Styled Site",
		"description":"Preview config",
		"theme":{
			"palette":"midnight",
			"font":"mono",
			"layout":"wide",
			"radius":"none",
			"customCss":".site-main { outline: 2px solid lime; }"
		},
		"header":{"variant":"minimal"},
		"footer":{"variant":"links"}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("style CSS status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body rendering.StyleCSS
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode style CSS: %v", err)
	}
	for _, expected := range []string{
		".theme-midnight {",
		".font-mono {",
		".layout-wide {",
		".radius-none {",
		".site-header-minimal .site-nav",
		".site-footer-links .site-footer-inner",
	} {
		if !strings.Contains(body.ThemeCSS, expected) {
			t.Fatalf("expected %q in theme CSS:\n%s", expected, body.ThemeCSS)
		}
	}
	if strings.Contains(body.ThemeCSS, ".site-main { outline: 2px solid lime; }") {
		t.Fatalf("theme CSS should not include draft custom CSS:\n%s", body.ThemeCSS)
	}
	if !strings.Contains(body.CurrentCSS, ".site-main { outline: 2px solid lime; }") {
		t.Fatalf("current CSS should include draft custom CSS:\n%s", body.CurrentCSS)
	}
	if !strings.Contains(body.CurrentCSS, ".theme-warm {") || !strings.Contains(body.CurrentCSS, ".theme-sage {") || !strings.Contains(body.CurrentCSS, "*::before") {
		t.Fatalf("current CSS should include the full renderer stylesheet:\n%s", body.CurrentCSS)
	}
	if !body.CustomCSSIncluded {
		t.Fatalf("CustomCSSIncluded = false, want true")
	}
	if strings.Join(body.BodyClasses, " ") != "theme-midnight font-mono layout-wide radius-none" {
		t.Fatalf("BodyClasses = %#v, want current theme classes", body.BodyClasses)
	}
	if body.Theme.Palette != siteconfig.PaletteMidnight || body.Header.ClassName != "site-header-minimal" || body.Footer.ClassName != "site-footer-links" {
		t.Fatalf("metadata = %#v %#v %#v, want current theme/header/footer", body.Theme, body.Header, body.Footer)
	}
	for _, expected := range []string{
		"body.theme-midnight.font-mono.layout-wide.radius-none {\n}",
		".site-header-minimal {\n}",
		".site-footer-links {\n}",
		".site-main {\n}",
	} {
		if !strings.Contains(body.BlankThemeCSS, expected) {
			t.Fatalf("expected %q in blank theme CSS:\n%s", expected, body.BlankThemeCSS)
		}
	}
	if _, err := os.Stat(filepath.Join(publicDir, "assets", "styxpress.css")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("style CSS should not write stylesheet, stat err: %v", err)
	}
}

func TestSiteConfigStyleCSSEndpointRejectsInvalidDraft(t *testing.T) {
	server, _, _ := newTestServer(t)

	request := authedRequest(t, server, http.MethodPost, "/api/site-config/style-css", `{
		"theme":{"palette":"unknown"}
	}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("style CSS status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error.Code != "invalid_site_config" {
		t.Fatalf("error code = %q, want invalid_site_config", body.Error.Code)
	}
}

func TestSiteEndpointsManageActiveConfig(t *testing.T) {
	root := t.TempDir()
	server, err := newServer("", config.NewSiteStore(root), nil)
	if err != nil {
		t.Fatalf("newServer returned error: %v", err)
	}

	list := authedRequest(t, server, http.MethodGet, "/api/sites", "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, list)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var listed sitesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if !listed.MultiSite || listed.ActiveSiteID != "" || len(listed.Sites) != 0 {
		t.Fatalf("sites response = %#v, want empty multi-site registry", listed)
	}

	create := authedRequest(t, server, http.MethodPost, "/api/sites", `{
		"name":"Client Site",
		"config":{
			"contentDir":"/tmp/client-content",
			"publicDir":"/tmp/client-public",
			"contentStorageMode":"local"
		}
	}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, create)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var created config.Site
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID != "client-site" || created.Config.Name != "Client Site" {
		t.Fatalf("created site = %#v, want client-site", created)
	}

	saveConfig := authedRequest(t, server, http.MethodPost, "/api/config", `{
		"name":"Client Live",
		"siteBaseUrl":"https://client.example.com",
		"contentDir":"/tmp/client-content",
		"publicDir":"/tmp/client-public",
		"contentStorageMode":"local",
		"remoteHost":"",
		"remoteUser":"",
		"sshKeyPath":"",
		"remotePublicDir":"",
		"remoteContentDir":""
	}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, saveConfig)
	if recorder.Code != http.StatusOK {
		t.Fatalf("save config status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	getConfig := authedRequest(t, server, http.MethodGet, "/api/config", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, getConfig)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get config status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var active config.Config
	if err := json.Unmarshal(recorder.Body.Bytes(), &active); err != nil {
		t.Fatalf("decode active config: %v", err)
	}
	if active.Name != "Client Live" || active.SiteBaseURL != "https://client.example.com" {
		t.Fatalf("active config = %#v, want saved client config", active)
	}

	deleteLast := authedRequest(t, server, http.MethodDelete, "/api/sites/client-site", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, deleteLast)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var empty sitesResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &empty); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if empty.ActiveSiteID != "" || len(empty.Sites) != 0 {
		t.Fatalf("delete response = %#v, want empty registry", empty)
	}
}

func TestConfigEndpointRequiresActiveSiteInMultiSiteMode(t *testing.T) {
	server, err := newServer("", config.NewSiteStore(t.TempDir()), nil)
	if err != nil {
		t.Fatalf("newServer returned error: %v", err)
	}

	request := authedRequest(t, server, http.MethodGet, "/api/config", "")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("config status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error.Code != "site_required" {
		t.Fatalf("error code = %q, want site_required", body.Error.Code)
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

func TestSiteRenderAndPublishEndpointsRenderAllPosts(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)
	for _, post := range []content.Post{
		{Slug: "alpha", Title: "Alpha", Source: "# Alpha", PublishedAt: mustTime(t, "2026-04-01T09:30:00Z"), UpdatedAt: mustTime(t, "2026-04-01T09:30:00Z")},
		{Slug: "bravo", Title: "Bravo", Source: "# Bravo", PublishedAt: mustTime(t, "2026-04-02T09:30:00Z"), UpdatedAt: mustTime(t, "2026-04-02T09:30:00Z")},
	} {
		writePublishedPost(t, contentDir, post)
	}
	saveDraft := authedRequest(t, server, http.MethodPost, "/api/posts", `{"slug":"draft","title":"Draft","source":"# Draft"}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, saveDraft)
	if recorder.Code != http.StatusOK {
		t.Fatalf("save draft status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	staleDraftPath := filepath.Join(publicDir, "posts", "draft", "index.html")
	if err := os.MkdirAll(filepath.Dir(staleDraftPath), 0o755); err != nil {
		t.Fatalf("make stale draft dir: %v", err)
	}
	if err := os.WriteFile(staleDraftPath, []byte("stale draft"), 0o644); err != nil {
		t.Fatalf("write stale draft output: %v", err)
	}

	render := authedRequest(t, server, http.MethodPost, "/api/site/render", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, render)
	if recorder.Code != http.StatusOK {
		t.Fatalf("render status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var renderBody renderSiteResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &renderBody); err != nil {
		t.Fatalf("decode render response: %v", err)
	}
	if len(renderBody.Posts) != 2 || renderBody.Site.IndexPath == "" || renderBody.Site.StylesheetPath == "" {
		t.Fatalf("render response = %#v, want published posts and site paths", renderBody)
	}
	for _, path := range []string{
		filepath.Join(publicDir, "index.html"),
		filepath.Join(publicDir, "feed.xml"),
		filepath.Join(publicDir, "sitemap.xml"),
		filepath.Join(publicDir, "assets", "styxpress.css"),
		filepath.Join(publicDir, "posts", "alpha", "index.html"),
		filepath.Join(publicDir, "posts", "bravo", "index.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected rendered file %s: %v", path, err)
		}
	}
	if _, err := os.Stat(staleDraftPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale draft output stat error = %v, want not exist", err)
	}
	for _, path := range []string{
		filepath.Join(publicDir, "feed.xml"),
		filepath.Join(publicDir, "sitemap.xml"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(data), "draft") || strings.Contains(string(data), "Draft") {
			t.Fatalf("%s contains draft post:\n%s", path, data)
		}
	}

	var gotPassphrase string
	var gotConfig config.Config
	server.publishRunner = func(_ *http.Request, cfg config.Config, passphrase string) (publishing.Result, error) {
		gotConfig = cfg
		gotPassphrase = passphrase
		return publishing.Result{UploadedPaths: []string{"/srv/site/public/index.html"}}, nil
	}
	publish := authedRequest(t, server, http.MethodPost, "/api/site/publish", `{"passphrase":"secret"}`)
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, publish)
	if recorder.Code != http.StatusOK {
		t.Fatalf("publish status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var publishBody publishSiteResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &publishBody); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}
	if len(publishBody.Posts) != 2 || publishBody.Publish.UploadedPaths[0] != "/srv/site/public/index.html" {
		t.Fatalf("publish response = %#v, want all posts and publish result", publishBody)
	}
	if gotPassphrase != "secret" {
		t.Fatalf("passphrase = %q, want secret", gotPassphrase)
	}
	if gotConfig.ContentDir != contentDir || gotConfig.PublicDir != publicDir {
		t.Fatalf("publish config paths = %q %q, want %q %q", gotConfig.ContentDir, gotConfig.PublicDir, contentDir, publicDir)
	}

	list := authedRequest(t, server, http.MethodGet, "/api/posts", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, list)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list after publish status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var listBody postListResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list after publish: %v", err)
	}
	statusBySlug := make(map[string]postResponse)
	for _, post := range listBody.Posts {
		statusBySlug[post.Slug] = post
	}
	if statusBySlug["alpha"].PublishStatus != "published" || statusBySlug["alpha"].SyncedAt == "" {
		t.Fatalf("alpha publish state = %#v, want published with sync timestamp", statusBySlug["alpha"])
	}
	if statusBySlug["draft"].PublishStatus != "draft" || statusBySlug["draft"].SyncedAt != "" {
		t.Fatalf("draft publish state = %#v, want unchanged draft", statusBySlug["draft"])
	}
}

func TestSiteVerifyRemoteEndpointRendersAndRunsVerification(t *testing.T) {
	server, contentDir, publicDir := newTestServer(t)
	writePublishedPost(t, contentDir, content.Post{
		Slug:        "alpha",
		Title:       "Alpha",
		Source:      "# Alpha",
		PublishedAt: mustTime(t, "2026-04-01T09:30:00Z"),
		UpdatedAt:   mustTime(t, "2026-04-01T09:30:00Z"),
	})
	if _, err := content.NewRepository(contentDir).WritePost(content.Post{
		Slug:   "draft",
		Title:  "Draft",
		Source: "# Draft",
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write draft returned error: %v", err)
	}

	var gotPassphrase string
	var gotRemoteOnlyPaths []string
	var gotConfig config.Config
	server.verifyRemoteRunner = func(_ *http.Request, cfg config.Config, opts publishing.Options) (publishing.VerificationResult, error) {
		gotConfig = cfg
		gotPassphrase = opts.Passphrase
		gotRemoteOnlyPaths = append([]string(nil), opts.RemoteOnlyPaths...)
		for _, path := range []string{
			filepath.Join(cfg.PublicDir, "index.html"),
			filepath.Join(cfg.PublicDir, "feed.xml"),
			filepath.Join(cfg.PublicDir, "sitemap.xml"),
			filepath.Join(cfg.PublicDir, "assets", "styxpress.css"),
			filepath.Join(cfg.PublicDir, "posts", "alpha", "index.html"),
		} {
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("expected rendered file %s: %v", path, err)
			}
		}
		return publishing.VerificationResult{
			Files: []publishing.FileVerification{
				{
					RelativePath: "index.html",
					RemotePath:   "/srv/site/public/index.html",
					Status:       publishing.VerificationStatusPublished,
				},
				{
					RelativePath: "posts/alpha/index.html",
					RemotePath:   "/srv/site/public/posts/alpha/index.html",
					Status:       publishing.VerificationStatusPublished,
				},
			},
			Summary: publishing.VerificationSummary{Total: 2, Published: 2},
		}, nil
	}

	request := authedRequest(t, server, http.MethodPost, "/api/site/verify-remote", `{"passphrase":"secret"}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("verify status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	var body publishing.VerificationResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}
	if body.Summary.Published != 2 || len(body.Files) != 2 || body.Files[1].Status != publishing.VerificationStatusPublished {
		t.Fatalf("verify response = %#v, want published result", body)
	}
	if gotPassphrase != "secret" {
		t.Fatalf("passphrase = %q, want secret", gotPassphrase)
	}
	if !reflect.DeepEqual(gotRemoteOnlyPaths, []string{"posts/draft/index.html"}) {
		t.Fatalf("remote-only paths = %#v, want draft post index", gotRemoteOnlyPaths)
	}
	if gotConfig.ContentDir != contentDir || gotConfig.PublicDir != publicDir {
		t.Fatalf("verify config paths = %q %q, want %q %q", gotConfig.ContentDir, gotConfig.PublicDir, contentDir, publicDir)
	}
	alpha, err := content.NewRepository(contentDir).LoadPost("alpha")
	if err != nil {
		t.Fatalf("LoadPost alpha returned error: %v", err)
	}
	if alpha.PublishStatus() != content.PublishStatusPublished || alpha.SyncedAt.IsZero() {
		t.Fatalf("alpha publish state = %q synced %v, want verified published", alpha.PublishStatus(), alpha.SyncedAt)
	}
}

func TestSiteVerifyRemoteEndpointClearsSyncOnMismatch(t *testing.T) {
	server, contentDir, _ := newTestServer(t)
	writePublishedPost(t, contentDir, content.Post{
		Slug:        "alpha",
		Title:       "Alpha",
		Source:      "# Alpha",
		PublishedAt: mustTime(t, "2026-04-01T09:30:00Z"),
		UpdatedAt:   mustTime(t, "2026-04-01T09:30:00Z"),
		SyncedAt:    mustTime(t, "2026-04-01T09:35:00Z"),
	})

	server.verifyRemoteRunner = func(_ *http.Request, _ config.Config, _ publishing.Options) (publishing.VerificationResult, error) {
		return publishing.VerificationResult{
			Files: []publishing.FileVerification{{
				RelativePath: "posts/alpha/index.html",
				RemotePath:   "/srv/site/public/posts/alpha/index.html",
				Status:       publishing.VerificationStatusChangesPending,
			}},
			Summary: publishing.VerificationSummary{Total: 1, ChangesPending: 1},
		}, nil
	}

	request := authedRequest(t, server, http.MethodPost, "/api/site/verify-remote", `{}`)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("verify status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	alpha, err := content.NewRepository(contentDir).LoadPost("alpha")
	if err != nil {
		t.Fatalf("LoadPost alpha returned error: %v", err)
	}
	if alpha.PublishStatus() != content.PublishStatusPendingPublish || !alpha.SyncedAt.IsZero() {
		t.Fatalf("alpha publish state = %q synced %v, want pending without sync", alpha.PublishStatus(), alpha.SyncedAt)
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

	detail := authedRequest(t, server, http.MethodGet, "/api/posts/publish-me", "")
	recorder = httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, detail)
	if recorder.Code != http.StatusOK {
		t.Fatalf("detail after publish status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body postResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode detail after publish: %v", err)
	}
	if body.PublishedAt == "" || body.SyncedAt == "" || body.PublishStatus != "published" {
		t.Fatalf("publish state = %#v, want published and synced", body)
	}
}

func TestPublishEndpointReportsCleanupPathsOnUploadFailure(t *testing.T) {
	server, contentDir, _ := newTestServer(t)
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
	if _, err := os.Stat(filepath.Join(contentDir, "posts", "cleanup-message", "published_at.txt")); err != nil {
		t.Fatalf("published_at.txt should remain after failed upload: %v", err)
	}
	if _, err := os.Stat(filepath.Join(contentDir, "posts", "cleanup-message", "remote_synced_at.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("remote_synced_at.txt stat error = %v, want not exist after failed upload", err)
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

func writePublishedPost(t *testing.T, contentDir string, post content.Post) {
	t.Helper()
	if _, err := content.NewRepository(contentDir).WritePost(post, content.WritePostOptions{}); err != nil {
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
