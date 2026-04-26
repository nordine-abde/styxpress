package rendering

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nordine-abde/styxpress/internal/content"
)

func TestRenderPostWritesDocumentAndAssets(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	publishedAt := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 4, 2, 10, 45, 0, 0, time.UTC)

	repo := content.NewRepository(contentRoot)
	if err := repo.WriteCover("hello-world", "cover.jpg", strings.NewReader("cover image")); err != nil {
		t.Fatalf("write cover: %v", err)
	}
	if err := repo.WriteAsset("hello-world", "images/diagram.png", strings.NewReader("diagram")); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	if _, err := repo.WritePost(content.Post{
		Slug:        "hello-world",
		Title:       `Hello "World"`,
		Description: `A <safe> description & summary.`,
		Source:      "# Heading\n\nHello **Markdown**.\n\n![Diagram](assets/images/diagram.png)\n",
		PublishedAt: publishedAt,
		UpdatedAt:   updatedAt,
		Cover:       "cover.jpg",
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot, Options{SiteBaseURL: "https://blog.example.com/"})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	result, err := renderer.RenderPost("hello-world")
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	assertFileContent(t, result.IndexPath, []string{
		`<!doctype html>`,
		`<title>Hello &#34;World&#34;</title>`,
		`<meta name="description" content="A &lt;safe&gt; description &amp; summary.">`,
		`<meta property="og:image" content="https://blog.example.com/posts/hello-world/cover.jpg">`,
		`<h1>Heading</h1>`,
		`<strong>Markdown</strong>`,
		`<img src="assets/images/diagram.png" alt="Diagram" />`,
	})
	assertFileEquals(t, filepath.Join(publicRoot, "posts", "hello-world", "cover.jpg"), "cover image")
	assertFileEquals(t, filepath.Join(publicRoot, "posts", "hello-world", "assets", "images", "diagram.png"), "diagram")
	if len(result.Assets) != 1 || result.Assets[0] != "images/diagram.png" {
		t.Fatalf("unexpected assets: %#v", result.Assets)
	}
}

func TestRenderPostEscapesRawHTML(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	if _, err := repo.WritePost(content.Post{
		Slug:        "raw-html",
		Title:       "Raw HTML",
		Source:      "Inline <script>alert(1)</script> text.\n\n<div>block</div>\n",
		PublishedAt: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot, Options{})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	result, err := renderer.RenderPost("raw-html")
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	data, err := os.ReadFile(result.IndexPath)
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	html := string(data)
	if strings.Contains(html, "<script>") || strings.Contains(html, "<div>block</div>") {
		t.Fatalf("raw HTML was not escaped:\n%s", html)
	}
	for _, expected := range []string{
		`&lt;script&gt;alert(1)&lt;/script&gt;`,
		`&lt;div&gt;block&lt;/div&gt;`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in rendered HTML:\n%s", expected, html)
		}
	}
}

func TestRenderPostReconcilesRemovedAssetsAndReplacedCover(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	if err := repo.WriteCover("hello-world", "cover.jpg", strings.NewReader("first cover")); err != nil {
		t.Fatalf("write first cover: %v", err)
	}
	if err := repo.WriteAsset("hello-world", "old.txt", strings.NewReader("old")); err != nil {
		t.Fatalf("write old asset: %v", err)
	}
	if _, err := repo.WritePost(content.Post{
		Slug:        "hello-world",
		Title:       "Hello",
		Source:      "Hello",
		PublishedAt: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
		Cover:       "cover.jpg",
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write first post: %v", err)
	}
	renderer, err := New(contentRoot, publicRoot, Options{})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderPost("hello-world"); err != nil {
		t.Fatalf("render first post: %v", err)
	}

	if err := repo.WriteCover("hello-world", "cover.png", strings.NewReader("second cover")); err != nil {
		t.Fatalf("write second cover: %v", err)
	}
	if err := repo.DeleteAsset("hello-world", "old.txt"); err != nil {
		t.Fatalf("delete old asset: %v", err)
	}
	if err := repo.WriteAsset("hello-world", "new.txt", strings.NewReader("new")); err != nil {
		t.Fatalf("write new asset: %v", err)
	}
	if _, err := repo.WritePost(content.Post{
		Slug:   "hello-world",
		Title:  "Hello",
		Source: "Updated",
		Cover:  "cover.png",
	}, content.WritePostOptions{Now: time.Date(2026, 4, 2, 9, 30, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("write updated post: %v", err)
	}
	if _, err := renderer.RenderPost("hello-world"); err != nil {
		t.Fatalf("render updated post: %v", err)
	}

	assertMissing(t, filepath.Join(publicRoot, "posts", "hello-world", "cover.jpg"))
	assertFileEquals(t, filepath.Join(publicRoot, "posts", "hello-world", "cover.png"), "second cover")
	assertMissing(t, filepath.Join(publicRoot, "posts", "hello-world", "assets", "old.txt"))
	assertFileEquals(t, filepath.Join(publicRoot, "posts", "hello-world", "assets", "new.txt"), "new")
}

func TestRenderPostDoesNotOverwriteIndexWhenLoadFails(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	publicPostDir := filepath.Join(publicRoot, "posts", "broken")
	if err := os.MkdirAll(publicPostDir, 0o755); err != nil {
		t.Fatalf("make public post dir: %v", err)
	}
	indexPath := filepath.Join(publicPostDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("previous"), 0o644); err != nil {
		t.Fatalf("write previous index: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(contentRoot, "posts", "broken"), 0o755); err != nil {
		t.Fatalf("make broken content dir: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot, Options{})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderPost("broken"); !errors.Is(err, content.ErrInvalidPost) {
		t.Fatalf("expected invalid post error, got %v", err)
	}
	assertFileEquals(t, indexPath, "previous")
}

func TestRenderPreviewDoesNotWritePublicFiles(t *testing.T) {
	publicRoot := filepath.Join(t.TempDir(), "public")
	renderer, err := New(filepath.Join(t.TempDir(), "content"), publicRoot, Options{})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	html, err := renderer.RenderPreview(content.Post{
		Slug:        "preview",
		Title:       "Preview",
		Description: "Draft preview",
		Source:      "# Preview\n",
		PublishedAt: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("render preview: %v", err)
	}
	if !strings.Contains(html, "<h1>Preview</h1>") {
		t.Fatalf("preview did not render markdown:\n%s", html)
	}
	if _, err := os.Stat(publicRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview should not create public root, stat err: %v", err)
	}
}

func TestRenderSiteWritesHomepageFeedAndSitemap(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	first := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	second := time.Date(2026, 4, 2, 9, 30, 0, 0, time.UTC)

	if err := repo.WriteCover("alpha", "cover.jpg", strings.NewReader("cover")); err != nil {
		t.Fatalf("write cover: %v", err)
	}
	if err := repo.WriteAsset("alpha", "diagram.png", strings.NewReader("diagram")); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	posts := []content.Post{
		{Slug: "zulu", Title: "Zulu", Description: "Last alphabetically", Source: "Zulu", PublishedAt: second, UpdatedAt: second},
		{Slug: "alpha", Title: "Alpha & Friends", Description: "Featured <post>", Source: "Alpha", PublishedAt: second, UpdatedAt: second, Cover: "cover.jpg"},
		{Slug: "older", Title: "Older", Source: "Older", PublishedAt: first, UpdatedAt: first},
	}
	for _, post := range posts {
		if _, err := repo.WritePost(post, content.WritePostOptions{}); err != nil {
			t.Fatalf("write post %s: %v", post.Slug, err)
		}
	}
	if err := repo.WriteFeatured([]string{"older", "alpha"}); err != nil {
		t.Fatalf("write featured: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot, Options{SiteBaseURL: "https://blog.example.com"})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	result, err := renderer.RenderSite()
	if err != nil {
		t.Fatalf("render site: %v", err)
	}

	assertFileContent(t, result.IndexPath, []string{
		`<h1 id="featured-posts">Featured Posts</h1>`,
		`<a href="/posts/older/">Older</a>`,
		`<a href="/posts/alpha/">Alpha &amp; Friends</a>`,
		`<p>Featured &lt;post&gt;</p>`,
		`<img src="/posts/alpha/cover.jpg" alt="">`,
	})
	assertOrderAfter(t, result.IndexPath, `<h1 id="latest-posts">Latest Posts</h1>`, []string{
		`<a href="/posts/alpha/">Alpha &amp; Friends</a>`,
		`<a href="/posts/zulu/">Zulu</a>`,
		`<a href="/posts/older/">Older</a>`,
	})
	assertFileContent(t, result.FeedPath, []string{
		`<link>https://blog.example.com/posts/alpha/</link>`,
		`<guid isPermaLink="true">https://blog.example.com/posts/zulu/</guid>`,
		`<description>Featured &lt;post&gt;</description>`,
	})
	assertFileContent(t, result.SitemapPath, []string{
		`<loc>https://blog.example.com/</loc>`,
		`<loc>https://blog.example.com/posts/alpha/</loc>`,
		`<lastmod>2026-04-02</lastmod>`,
	})

	sitemap, err := os.ReadFile(result.SitemapPath)
	if err != nil {
		t.Fatalf("read sitemap: %v", err)
	}
	for _, excluded := range []string{"feed.xml", "cover.jpg", "diagram.png", "source.md"} {
		if strings.Contains(string(sitemap), excluded) {
			t.Fatalf("sitemap contains excluded path %q:\n%s", excluded, sitemap)
		}
	}
}

func TestRenderSiteRequiresAbsoluteBaseURL(t *testing.T) {
	renderer, err := New(filepath.Join(t.TempDir(), "content"), filepath.Join(t.TempDir(), "public"), Options{})
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderSite(); !errors.Is(err, ErrInvalidRenderConfig) {
		t.Fatalf("RenderSite error = %v, want ErrInvalidRenderConfig", err)
	}
	if _, err := New(filepath.Join(t.TempDir(), "content"), filepath.Join(t.TempDir(), "public"), Options{SiteBaseURL: "/relative"}); !errors.Is(err, ErrInvalidRenderConfig) {
		t.Fatalf("New relative base URL error = %v, want ErrInvalidRenderConfig", err)
	}
}

func assertFileContent(t *testing.T, path string, values []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, value := range values {
		if !strings.Contains(string(data), value) {
			t.Fatalf("expected %q in %s:\n%s", value, path, string(data))
		}
	}
}

func assertOrderAfter(t *testing.T, path string, marker string, values []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)
	markerIndex := strings.Index(content, marker)
	if markerIndex == -1 {
		t.Fatalf("expected marker %q in %s:\n%s", marker, path, content)
	}
	content = content[markerIndex:]
	previous := -1
	for _, value := range values {
		current := strings.Index(content, value)
		if current == -1 {
			t.Fatalf("expected %q in %s:\n%s", value, path, content)
		}
		if current <= previous {
			t.Fatalf("expected %q after previous value in %s:\n%s", value, path, content)
		}
		previous = current
	}
}

func assertFileEquals(t *testing.T, path string, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != expected {
		t.Fatalf("unexpected %s content: got %q want %q", path, string(data), expected)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to be missing, stat err: %v", path, err)
	}
}
