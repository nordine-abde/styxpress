package rendering

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/nordine-abde/styxpress/internal/content"
	"github.com/nordine-abde/styxpress/internal/siteconfig"
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

	renderer, err := New(contentRoot, publicRoot)
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
		`<meta property="og:image" content="/posts/hello-world/cover.jpg">`,
		`<img src="/posts/hello-world/cover.jpg" alt="">`,
		`<h1>Heading</h1>`,
		`<strong>Markdown</strong>`,
		`<img src="assets/images/diagram.png" alt="Diagram" />`,
	})
	assertOrderAfter(t, result.IndexPath, `<header class="post-header">`, []string{
		`<img src="/posts/hello-world/cover.jpg" alt="">`,
		`<h1>Hello &#34;World&#34;</h1>`,
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

	renderer, err := New(contentRoot, publicRoot)
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

func TestRenderPostNormalizesStandaloneBreakTagsBeforeImages(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	if err := repo.WriteAsset("inline-images", "styxpress.png", strings.NewReader("logo")); err != nil {
		t.Fatalf("write logo asset: %v", err)
	}
	imagePath := "WhatsApp Image 2026-05-02 at 21.23.39 (1).jpeg"
	if err := repo.WriteAsset("inline-images", imagePath, strings.NewReader("photo")); err != nil {
		t.Fatalf("write photo asset: %v", err)
	}
	if _, err := repo.WritePost(content.Post{
		Slug:  "inline-images",
		Title: "Inline Images",
		Source: strings.Join([]string{
			"# Inline Images",
			"",
			"Intro",
			"<br>",
			"<br/>",
			"![styxpress](assets/styxpress.png#small)",
			"<br />",
			"![Phone](assets/WhatsApp%20Image%202026-05-02%20at%2021.23.39%20(1).jpeg#small)",
			"",
		}, "\n"),
		PublishedAt: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	result, err := renderer.RenderPost("inline-images")
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	assertFileContent(t, result.IndexPath, []string{
		`<img src="assets/styxpress.png#small" alt="styxpress" />`,
		`<img src="assets/WhatsApp%20Image%202026-05-02%20at%2021.23.39%20(1).jpeg#small" alt="Phone" />`,
	})
	assertFileOmits(t, result.IndexPath, []string{
		`&lt;br`,
		`![styxpress]`,
		`![Phone]`,
	})
}

func TestRenderPostRejectsDraft(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	if _, err := repo.WritePost(content.Post{
		Slug:   "draft",
		Title:  "Draft",
		Source: "# Draft\n",
	}, content.WritePostOptions{Now: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("write draft: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderPost("draft"); !errors.Is(err, ErrUnpublishedPost) {
		t.Fatalf("RenderPost error = %v, want ErrUnpublishedPost", err)
	}
	assertMissing(t, filepath.Join(publicRoot, "posts", "draft", "index.html"))
}

func TestRenderPreviewRejectsSymlinkCover(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	if _, err := repo.WritePost(content.Post{
		Slug:   "hello-world",
		Title:  "Hello",
		Source: "Body",
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}

	secretPath := filepath.Join(t.TempDir(), "secret-cover.jpg")
	if err := os.WriteFile(secretPath, []byte("secret cover"), 0o644); err != nil {
		t.Fatalf("write secret cover: %v", err)
	}
	coverPath := filepath.Join(contentRoot, "posts", "hello-world", "cover.jpg")
	if err := os.Symlink(secretPath, coverPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	html, err := renderer.RenderPreview(content.Post{
		Slug:   "hello-world",
		Title:  "Hello",
		Source: "Body",
		Cover:  "cover.jpg",
	})
	if !errors.Is(err, content.ErrInvalidAsset) {
		t.Fatalf("RenderPreview error = %v, want ErrInvalidAsset", err)
	}
	if strings.Contains(html, "secret cover") {
		t.Fatalf("preview leaked symlink target content: %s", html)
	}
}

func TestRenderPostRejectsSymlinkedPublicParent(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	outsideRoot := t.TempDir()
	publishedAt := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	repo := content.NewRepository(contentRoot)
	if _, err := repo.WritePost(content.Post{
		Slug:        "hello-world",
		Title:       "Hello",
		Source:      "Body",
		PublishedAt: publishedAt,
		UpdatedAt:   publishedAt,
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}
	if err := os.MkdirAll(publicRoot, 0o755); err != nil {
		t.Fatalf("mkdir public root: %v", err)
	}
	if err := os.Symlink(outsideRoot, filepath.Join(publicRoot, "posts")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderPost("hello-world"); !errors.Is(err, ErrUnsafeAsset) {
		t.Fatalf("RenderPost error = %v, want ErrUnsafeAsset", err)
	}
	assertMissing(t, filepath.Join(outsideRoot, "hello-world", "index.html"))
}

func TestRenderAllRejectsSymlinkedPublicParentBeforeDeletingDraftOutput(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	outsideRoot := t.TempDir()
	repo := content.NewRepository(contentRoot)
	if _, err := repo.WritePost(content.Post{
		Slug:   "draft",
		Title:  "Draft",
		Source: "Body",
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}
	outsideDraftOutput := filepath.Join(outsideRoot, "draft", "index.html")
	if err := os.MkdirAll(filepath.Dir(outsideDraftOutput), 0o755); err != nil {
		t.Fatalf("mkdir outside draft output: %v", err)
	}
	if err := os.WriteFile(outsideDraftOutput, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write outside draft output: %v", err)
	}
	if err := os.MkdirAll(publicRoot, 0o755); err != nil {
		t.Fatalf("mkdir public root: %v", err)
	}
	if err := os.Symlink(outsideRoot, filepath.Join(publicRoot, "posts")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderAll(); !errors.Is(err, ErrUnsafeAsset) {
		t.Fatalf("RenderAll error = %v, want ErrUnsafeAsset", err)
	}
	assertFileEquals(t, outsideDraftOutput, "keep")
}

func TestNewRejectsOverlappingContentAndPublicRoots(t *testing.T) {
	root := t.TempDir()
	contentRoot := filepath.Join(root, "content")
	if err := os.MkdirAll(filepath.Join(contentRoot, "posts", "draft"), 0o755); err != nil {
		t.Fatalf("MkdirAll draft: %v", err)
	}
	sourcePath := filepath.Join(contentRoot, "posts", "draft", "source.md")
	if err := os.WriteFile(sourcePath, []byte("# Draft\n"), 0o644); err != nil {
		t.Fatalf("WriteFile source: %v", err)
	}

	_, err := New(contentRoot, contentRoot)
	if !errors.Is(err, ErrInvalidRenderConfig) {
		t.Fatalf("New error = %v, want ErrInvalidRenderConfig", err)
	}
	assertFileEquals(t, sourcePath, "# Draft\n")
}

func TestNewRejectsSymlinkOverlappingContentAndPublicRoots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on some Windows setups")
	}

	root := t.TempDir()
	contentRoot := filepath.Join(root, "content")
	if err := os.MkdirAll(contentRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll content: %v", err)
	}
	publicRoot := filepath.Join(root, "public-link")
	if err := os.Symlink(contentRoot, publicRoot); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	_, err := New(contentRoot, publicRoot)
	if !errors.Is(err, ErrInvalidRenderConfig) {
		t.Fatalf("New error = %v, want ErrInvalidRenderConfig", err)
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
	renderer, err := New(contentRoot, publicRoot)
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

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderPost("broken"); !errors.Is(err, content.ErrInvalidPost) {
		t.Fatalf("expected invalid post error, got %v", err)
	}
	assertFileEquals(t, indexPath, "previous")
}

func TestRenderPreviewDoesNotWritePublicFiles(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	repo := content.NewRepository(contentRoot)
	if err := repo.WriteCover("preview", "cover.jpg", strings.NewReader("cover image")); err != nil {
		t.Fatalf("write preview cover: %v", err)
	}
	html, err := renderer.RenderPreview(content.Post{
		Slug:        "preview",
		Title:       "Preview",
		Description: "Draft preview",
		Source:      "# Preview\n",
		Cover:       "cover.jpg",
		PublishedAt: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("render preview: %v", err)
	}
	if !strings.Contains(html, "<h1>Preview</h1>") {
		t.Fatalf("preview did not render markdown:\n%s", html)
	}
	if !strings.Contains(html, `<img src="data:image/jpeg;base64,`) {
		t.Fatalf("preview did not inline the cover:\n%s", html)
	}
	assertStringOrder(t, html, `<header class="post-header">`, []string{
		`<img src="data:image/jpeg;base64,`,
		`<h1>Preview</h1>`,
	})
	if !strings.Contains(html, "<style>") || !strings.Contains(html, "font-family:") {
		t.Fatalf("preview did not inline the stylesheet:\n%s", html)
	}
	if strings.Contains(html, `<link rel="stylesheet" href="/assets/styxpress.css">`) {
		t.Fatalf("preview should not link the public stylesheet:\n%s", html)
	}
	if _, err := os.Stat(publicRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("preview should not create public root, stat err: %v", err)
	}
}

func TestStyleElementCSSNeutralizesStyleEndTag(t *testing.T) {
	css := `.x::before { content: "</style><script>alert(1)</script>"; }`
	safe := styleElementCSS(css)
	if strings.Contains(safe, `</style><script>`) {
		t.Fatalf("inline CSS can close the style element:\n%s", safe)
	}
	if !strings.Contains(safe, `<\/style><script>`) {
		t.Fatalf("inline CSS did not neutralize style end tag:\n%s", safe)
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
		{Slug: "alpha", Title: "Alpha & Friends", Description: "Public <post>", Source: "Alpha", PublishedAt: second, UpdatedAt: second, Cover: "cover.jpg"},
		{Slug: "older", Title: "Older", Source: "Older", PublishedAt: first, UpdatedAt: first},
		{Slug: "draft-post", Title: "Draft Post", Source: "Draft", UpdatedAt: second.Add(time.Hour)},
	}
	for _, post := range posts {
		if _, err := repo.WritePost(post, content.WritePostOptions{}); err != nil {
			t.Fatalf("write post %s: %v", post.Slug, err)
		}
	}
	staleDraftPath := filepath.Join(publicRoot, "posts", "draft-post", "index.html")
	if err := os.MkdirAll(filepath.Dir(staleDraftPath), 0o755); err != nil {
		t.Fatalf("make stale draft dir: %v", err)
	}
	if err := os.WriteFile(staleDraftPath, []byte("stale draft"), 0o644); err != nil {
		t.Fatalf("write stale draft output: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	result, err := renderer.RenderSite()
	if err != nil {
		t.Fatalf("render site: %v", err)
	}

	assertFileContent(t, result.IndexPath, []string{
		`<h1 id="latest-posts">Latest Posts</h1>`,
		`<a href="/posts/alpha/">Alpha &amp; Friends</a>`,
		`<p>Public &lt;post&gt;</p>`,
		`<img src="/posts/alpha/cover.jpg" alt="">`,
		`Published with <a href="https://styxpress.anordine.com">Styxpress</a>`,
	})
	assertFileOmits(t, result.IndexPath, []string{`Featured Posts`, `Draft Post`, `/posts/draft-post/`})
	assertOrderAfter(t, result.IndexPath, `<h1 id="latest-posts">Latest Posts</h1>`, []string{
		`<a href="/posts/alpha/">Alpha &amp; Friends</a>`,
		`<a href="/posts/zulu/">Zulu</a>`,
		`<a href="/posts/older/">Older</a>`,
	})
	assertFileContent(t, result.FeedPath, []string{
		`<link>/posts/alpha/</link>`,
		`<guid isPermaLink="true">/posts/zulu/</guid>`,
		`<description>Public &lt;post&gt;</description>`,
	})
	assertFileOmits(t, result.FeedPath, []string{`draft-post`, `Draft Post`})
	assertFileContent(t, result.SitemapPath, []string{
		`<loc>/</loc>`,
		`<loc>/posts/alpha/</loc>`,
		`<lastmod>2026-04-02</lastmod>`,
	})
	assertFileOmits(t, result.SitemapPath, []string{`draft-post`})
	assertMissing(t, staleDraftPath)
}

func TestRenderUsesSiteConfigHeaderFooterAndStylesheet(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	publishedAt := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	if _, err := repo.WritePost(content.Post{
		Slug:        "configured",
		Title:       "Configured",
		Description: "Configured post",
		Source:      "# Configured\n\nBody.",
		PublishedAt: publishedAt,
		UpdatedAt:   publishedAt,
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}
	if err := siteconfig.Save(contentRoot, siteconfig.Config{
		Title:       "Anordine",
		Description: "Software notes",
		Header: siteconfig.HeaderConfig{
			Links: []siteconfig.Link{
				{Label: "Home", Href: "/"},
				{Label: "RSS", Href: "/feed.xml"},
			},
		},
		Footer: siteconfig.FooterConfig{
			Text:          "Built from Markdown files.",
			ShowWatermark: false,
			Links: []siteconfig.Link{
				{Label: "Email", Href: "mailto:hello@example.com"},
			},
		},
	}); err != nil {
		t.Fatalf("save site config: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	siteResult, err := renderer.RenderSite()
	if err != nil {
		t.Fatalf("render site: %v", err)
	}
	postResult, err := renderer.RenderPost("configured")
	if err != nil {
		t.Fatalf("render post: %v", err)
	}

	assertFileContent(t, siteResult.IndexPath, []string{
		`<link rel="icon" href="/favicon.ico" type="image/x-icon">`,
		`<a class="site-title" href="/">Anordine</a>`,
		`<a href="/feed.xml">RSS</a>`,
		`Built from Markdown files.`,
		`<a href="mailto:hello@example.com">Email</a>`,
	})
	assertFileOmits(t, siteResult.IndexPath, []string{`Published with <a href="https://styxpress.anordine.com">Styxpress</a>`})
	assertFileContent(t, postResult.IndexPath, []string{
		`<link rel="icon" href="/favicon.ico" type="image/x-icon">`,
		`<link rel="stylesheet" href="/assets/styxpress.css">`,
		`<h1>Configured</h1>`,
	})
	assertFileExists(t, siteResult.FaviconPath)
	assertFileContent(t, filepath.Join(publicRoot, "assets", "styxpress.css"), []string{
		`:root`,
		`font-family:`,
		`.post-header img`,
		`.post-content img[src$="#small"]`,
		`@media (min-width: 760px)`,
	})
}

func TestRenderUsesCustomFavicon(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	publishedAt := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	if _, err := repo.WritePost(content.Post{
		Slug:        "custom-icon",
		Title:       "Custom Icon",
		Source:      "Custom icon.",
		PublishedAt: publishedAt,
		UpdatedAt:   publishedAt,
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(contentRoot, "assets"), 0o755); err != nil {
		t.Fatalf("make assets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentRoot, "assets", "site icon.ico"), []byte("ico"), 0o644); err != nil {
		t.Fatalf("write favicon: %v", err)
	}
	cfg := siteconfig.Default()
	cfg.Favicon = "assets/site icon.ico"
	if err := siteconfig.Save(contentRoot, cfg); err != nil {
		t.Fatalf("save site config: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	result, err := renderer.RenderSite()
	if err != nil {
		t.Fatalf("render site: %v", err)
	}

	assertFileContent(t, result.IndexPath, []string{
		`<link rel="icon" href="/assets/site%20icon.ico" type="image/x-icon">`,
	})
	assertFileEquals(t, filepath.Join(publicRoot, "assets", "site icon.ico"), "ico")
}

func TestRenderSitePreviewUsesProvidedConfigAndDoesNotWritePublicFiles(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	publishedAt := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	if _, err := repo.WritePost(content.Post{
		Slug:        "previewed",
		Title:       "Previewed",
		Description: "Visible in preview",
		Source:      "Previewed",
		PublishedAt: publishedAt,
		UpdatedAt:   publishedAt,
	}, content.WritePostOptions{}); err != nil {
		t.Fatalf("write post: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	cfg := siteconfig.Default()
	cfg.Title = "Preview Site"
	cfg.Footer.ShowWatermark = false
	html, err := renderer.RenderSitePreview(cfg)
	if err != nil {
		t.Fatalf("render site preview: %v", err)
	}

	for _, expected := range []string{
		`<title>Preview Site</title>`,
		`<style>`,
		`<a href="/posts/previewed/">Previewed</a>`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("expected %q in site preview:\n%s", expected, html)
		}
	}
	if strings.Contains(html, `Published with <a href="https://styxpress.anordine.com">Styxpress</a>`) {
		t.Fatalf("site preview should honor disabled watermark:\n%s", html)
	}
	if strings.Contains(html, `<link rel="stylesheet" href="/assets/styxpress.css">`) {
		t.Fatalf("site preview should not link the public stylesheet:\n%s", html)
	}
	if _, err := os.Stat(publicRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("site preview should not create public root, stat err: %v", err)
	}
}

func TestRenderSitePreviewDoesNotRemoveDraftOutput(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	if _, err := repo.WritePost(content.Post{
		Slug:   "draft",
		Title:  "Draft",
		Source: "Draft",
	}, content.WritePostOptions{Now: time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)}); err != nil {
		t.Fatalf("write draft: %v", err)
	}
	staleDraftPath := filepath.Join(publicRoot, "posts", "draft", "index.html")
	if err := os.MkdirAll(filepath.Dir(staleDraftPath), 0o755); err != nil {
		t.Fatalf("make stale draft dir: %v", err)
	}
	if err := os.WriteFile(staleDraftPath, []byte("stale draft"), 0o644); err != nil {
		t.Fatalf("write stale draft output: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	if _, err := renderer.RenderSitePreview(siteconfig.Default()); err != nil {
		t.Fatalf("render site preview: %v", err)
	}
	assertFileEquals(t, staleDraftPath, "stale draft")
}

func TestRenderAllWritesPostsSiteAndStylesheet(t *testing.T) {
	contentRoot := filepath.Join(t.TempDir(), "content")
	publicRoot := filepath.Join(t.TempDir(), "public")
	repo := content.NewRepository(contentRoot)
	publishedAt := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)
	for _, post := range []content.Post{
		{Slug: "alpha", Title: "Alpha", Source: "# Alpha", PublishedAt: publishedAt, UpdatedAt: publishedAt},
		{Slug: "bravo", Title: "Bravo", Source: "# Bravo", PublishedAt: publishedAt.Add(time.Hour), UpdatedAt: publishedAt.Add(time.Hour)},
		{Slug: "draft", Title: "Draft", Source: "# Draft", UpdatedAt: publishedAt.Add(2 * time.Hour)},
	} {
		if _, err := repo.WritePost(post, content.WritePostOptions{}); err != nil {
			t.Fatalf("write post %s: %v", post.Slug, err)
		}
	}
	staleDraftPath := filepath.Join(publicRoot, "posts", "draft", "index.html")
	if err := os.MkdirAll(filepath.Dir(staleDraftPath), 0o755); err != nil {
		t.Fatalf("make stale draft dir: %v", err)
	}
	if err := os.WriteFile(staleDraftPath, []byte("stale draft"), 0o644); err != nil {
		t.Fatalf("write stale draft output: %v", err)
	}

	renderer, err := New(contentRoot, publicRoot)
	if err != nil {
		t.Fatalf("new renderer: %v", err)
	}
	result, err := renderer.RenderAll()
	if err != nil {
		t.Fatalf("render all: %v", err)
	}
	if len(result.Posts) != 2 {
		t.Fatalf("rendered posts = %d, want 2", len(result.Posts))
	}
	assertMissing(t, staleDraftPath)

	assertFileContent(t, filepath.Join(publicRoot, "posts", "alpha", "index.html"), []string{
		`<link rel="stylesheet" href="/assets/styxpress.css">`,
		`<h1>Alpha</h1>`,
	})
	assertFileContent(t, filepath.Join(publicRoot, "posts", "bravo", "index.html"), []string{
		`<h1>Bravo</h1>`,
	})
	assertFileContent(t, result.Site.IndexPath, []string{
		`<body>`,
		`<a href="/posts/bravo/">Bravo</a>`,
		`<a href="/posts/alpha/">Alpha</a>`,
	})
	assertFileContent(t, result.Site.FeedPath, []string{`<link>/posts/alpha/</link>`})
	assertFileOmits(t, result.Site.FeedPath, []string{`draft`})
	assertFileContent(t, result.Site.SitemapPath, []string{`<loc>/posts/bravo/</loc>`})
	assertFileOmits(t, result.Site.SitemapPath, []string{`draft`})
	assertFileContent(t, result.Site.StylesheetPath, []string{`:root`, `.post-card`})
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
	assertStringOrder(t, string(data), marker, values)
}

func assertStringOrder(t *testing.T, content string, marker string, values []string) {
	t.Helper()
	markerIndex := strings.Index(content, marker)
	if markerIndex == -1 {
		t.Fatalf("expected marker %q in content:\n%s", marker, content)
	}
	content = content[markerIndex:]
	previous := -1
	for _, value := range values {
		current := strings.Index(content, value)
		if current == -1 {
			t.Fatalf("expected %q in content:\n%s", value, content)
		}
		if current <= previous {
			t.Fatalf("expected %q after previous value in content:\n%s", value, content)
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

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		t.Fatalf("expected %s to exist as a file, stat info: %#v err: %v", path, info, err)
	}
}

func assertFileOmits(t *testing.T, path string, values []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, value := range values {
		if strings.Contains(string(data), value) {
			t.Fatalf("did not expect %q in %s:\n%s", value, path, string(data))
		}
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected %s to be missing, stat err: %v", path, err)
	}
}
