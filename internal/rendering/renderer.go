package rendering

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	stdhtml "html"
	"html/template"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "embed"

	"github.com/nordine-abde/styxpress/internal/content"
	"github.com/nordine-abde/styxpress/internal/localpath"
	"github.com/nordine-abde/styxpress/internal/siteconfig"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

const (
	postsDirName  = "posts"
	assetsDirName = "assets"
	indexFileName = "index.html"
	directoryMode = 0o755
	fileMode      = 0o644
	timeFormatRSS = "Mon, 02 Jan 2006 15:04:05 GMT"
)

//go:embed assets/favicon.ico
var defaultFavicon []byte

var (
	ErrInvalidRenderConfig = errors.New("invalid render config")
	ErrUnsafeAsset         = errors.New("unsafe asset")
	ErrUnpublishedPost     = errors.New("unpublished post")
)

type Renderer struct {
	contentRoot string
	publicRoot  string
	siteConfig  siteconfig.Config
	markdown    goldmark.Markdown
}

type Result struct {
	Slug      string
	PublicDir string
	IndexPath string
	CoverPath string
	Assets    []string
}

type SiteResult struct {
	IndexPath      string
	FeedPath       string
	SitemapPath    string
	StylesheetPath string
	FaviconPath    string
}

type AllResult struct {
	Posts []Result
	Site  SiteResult
}

type pageData struct {
	Site           siteTemplateData
	Title          string
	Description    string
	CanonicalURL   string
	OpenGraphImage template.URL
	CoverURL       template.URL
	PublishedAt    string
	UpdatedAt      string
	ArticleHTML    template.HTML
}

type sitePageData struct {
	Site   siteTemplateData
	Title  string
	Latest []postSummary
}

type siteTemplateData struct {
	Title       string
	Description string
	FaviconHref template.URL
	Style       styleTemplateData
	Header      headerTemplateData
	Footer      footerTemplateData
}

type styleTemplateData struct {
	Href      string
	InlineCSS template.CSS
}

type headerTemplateData struct {
	Title string
	Links []siteconfig.Link
}

type footerTemplateData struct {
	Text          string
	ShowWatermark bool
	Links         []siteconfig.Link
}

type postSummary struct {
	Slug        string
	Title       string
	Description string
	URL         string
	CoverURL    string
	PublishedAt string
	UpdatedAt   string
}

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	Description string  `xml:"description,omitempty"`
	PubDate     string  `xml:"pubDate"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location     string `xml:"loc"`
	LastModified string `xml:"lastmod,omitempty"`
}

func New(contentRoot string, publicRoot string) (*Renderer, error) {
	if strings.TrimSpace(contentRoot) == "" {
		return nil, fmt.Errorf("%w: content root is required", ErrInvalidRenderConfig)
	}
	if strings.TrimSpace(publicRoot) == "" {
		return nil, fmt.Errorf("%w: public root is required", ErrInvalidRenderConfig)
	}
	contentRoot, err := localpath.CleanRequired(contentRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: content root: %v", ErrInvalidRenderConfig, err)
	}
	publicRoot, err = localpath.CleanRequired(publicRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: public root: %v", ErrInvalidRenderConfig, err)
	}
	if err := localpath.EnsureSeparateRoots("content root", contentRoot, "public root", publicRoot); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRenderConfig, err)
	}
	siteCfg, err := siteconfig.LoadOrDefault(contentRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: site config: %v", ErrInvalidRenderConfig, err)
	}

	return &Renderer{
		contentRoot: contentRoot,
		publicRoot:  publicRoot,
		siteConfig:  siteCfg,
		markdown: goldmark.New(
			goldmark.WithRendererOptions(
				goldmarkhtml.WithXHTML(),
				renderer.WithNodeRenderers(util.Prioritized(escapedHTMLRenderer{}, 900)),
			),
		),
	}, nil
}

func (r *Renderer) RenderSite() (SiteResult, error) {
	posts, err := r.homepagePosts(true)
	if err != nil {
		return SiteResult{}, err
	}

	indexHTML, err := r.renderHomepage(posts, r.siteConfig, linkedStyle())
	if err != nil {
		return SiteResult{}, err
	}
	feedXML, err := r.renderFeed(posts)
	if err != nil {
		return SiteResult{}, err
	}
	sitemapXML, err := r.renderSitemap(posts)
	if err != nil {
		return SiteResult{}, err
	}
	stylesheetPath, err := r.writeStyleSheet()
	if err != nil {
		return SiteResult{}, err
	}
	faviconPath, err := r.writeFavicon()
	if err != nil {
		return SiteResult{}, err
	}

	indexPath := filepath.Join(r.publicRoot, indexFileName)
	feedPath := filepath.Join(r.publicRoot, "feed.xml")
	sitemapPath := filepath.Join(r.publicRoot, "sitemap.xml")
	if err := writeAtomic(indexPath, []byte(indexHTML)); err != nil {
		return SiteResult{}, err
	}
	if err := writeAtomic(feedPath, feedXML); err != nil {
		return SiteResult{}, err
	}
	if err := writeAtomic(sitemapPath, sitemapXML); err != nil {
		return SiteResult{}, err
	}

	return SiteResult{
		IndexPath:      indexPath,
		FeedPath:       feedPath,
		SitemapPath:    sitemapPath,
		StylesheetPath: stylesheetPath,
		FaviconPath:    faviconPath,
	}, nil
}

func (r *Renderer) RenderAll() (AllResult, error) {
	repo := content.NewRepository(r.contentRoot)
	allPosts, err := repo.ListPosts()
	if err != nil {
		return AllResult{}, err
	}
	if err := r.removeUnpublishedPostOutput(allPosts); err != nil {
		return AllResult{}, err
	}
	posts, err := repo.ListPublishedPosts()
	if err != nil {
		return AllResult{}, err
	}

	results := make([]Result, 0, len(posts))
	for _, post := range posts {
		result, err := r.renderLoadedPost(post)
		if err != nil {
			return AllResult{}, err
		}
		results = append(results, result)
	}

	siteResult, err := r.RenderSite()
	if err != nil {
		return AllResult{}, err
	}
	return AllResult{Posts: results, Site: siteResult}, nil
}

func (r *Renderer) RenderPost(slug string) (Result, error) {
	repo := content.NewRepository(r.contentRoot)
	post, err := repo.LoadPost(slug)
	if err != nil {
		return Result{}, err
	}
	if !post.IsPublished() {
		return Result{}, fmt.Errorf("%w: %s", ErrUnpublishedPost, slug)
	}

	result, err := r.renderLoadedPost(post)
	if err != nil {
		return Result{}, err
	}
	if _, err := r.writeStyleSheet(); err != nil {
		return Result{}, err
	}
	if _, err := r.writeFavicon(); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (r *Renderer) renderLoadedPost(post content.Post) (Result, error) {
	document, err := r.renderPostDocument(post, r.siteConfig, linkedStyle(), false)
	if err != nil {
		return Result{}, err
	}
	publicDir := filepath.Join(r.publicRoot, postsDirName, post.Slug)
	if err := os.MkdirAll(publicDir, directoryMode); err != nil {
		return Result{}, err
	}

	coverPath, err := r.reconcileCover(post, publicDir)
	if err != nil {
		return Result{}, err
	}
	assets, err := r.reconcileAssets(post, publicDir)
	if err != nil {
		return Result{}, err
	}

	indexPath := filepath.Join(publicDir, indexFileName)
	if err := writeAtomic(indexPath, []byte(document)); err != nil {
		return Result{}, err
	}

	return Result{
		Slug:      post.Slug,
		PublicDir: publicDir,
		IndexPath: indexPath,
		CoverPath: coverPath,
		Assets:    assets,
	}, nil
}

func (r *Renderer) RenderPreview(post content.Post) (string, error) {
	return r.renderPostDocument(post, r.siteConfig, inlineStyle(), true)
}

func (r *Renderer) RenderSitePreview(cfg siteconfig.Config) (string, error) {
	cfg = siteconfig.WithDefaults(cfg)
	if err := cfg.Validate(); err != nil {
		return "", err
	}
	posts, err := r.homepagePosts(false)
	if err != nil {
		return "", err
	}
	return r.renderHomepage(posts, cfg, inlineStyle())
}

func (r *Renderer) renderPostDocument(post content.Post, cfg siteconfig.Config, style styleTemplateData, preview bool) (string, error) {
	if err := content.ValidateSlug(post.Slug); err != nil {
		return "", err
	}

	var article bytes.Buffer
	if err := r.markdown.Convert([]byte(normalizeMarkdownSource(post.Source)), &article); err != nil {
		return "", err
	}

	basePostURL := "/posts/" + post.Slug
	coverURL := ""
	if post.Cover != "" {
		if preview {
			var err error
			coverURL, err = r.previewCoverURL(post)
			if err != nil {
				return "", err
			}
		} else {
			coverURL = basePostURL + "/" + post.Cover
		}
	}

	data := pageData{
		Site:           r.siteData(cfg, style),
		Title:          post.Title,
		Description:    post.Description,
		CanonicalURL:   r.absoluteURL(basePostURL),
		OpenGraphImage: template.URL(r.absoluteURL(coverURL)),
		CoverURL:       template.URL(coverURL),
		PublishedAt:    formatOptionalTime(post.PublishedAt, "2006-01-02T15:04:05Z"),
		UpdatedAt:      formatOptionalTime(post.UpdatedAt, "2006-01-02T15:04:05Z"),
		ArticleHTML:    template.HTML(article.String()),
	}

	var document bytes.Buffer
	if err := postTemplate.Execute(&document, data); err != nil {
		return "", err
	}
	return document.String(), nil
}

func (r *Renderer) previewCoverURL(post content.Post) (string, error) {
	if post.Cover == "" {
		return "", nil
	}
	if !isCoverFile(post.Cover) {
		return "", content.ErrUnsupportedCover
	}
	data, err := os.ReadFile(filepath.Join(r.contentRoot, postsDirName, post.Slug, post.Cover))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	contentType := mime.TypeByExtension(filepath.Ext(post.Cover))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (r *Renderer) homepagePosts(cleanupOutput bool) ([]content.Post, error) {
	repo := content.NewRepository(r.contentRoot)
	allPosts, err := repo.ListPosts()
	if err != nil {
		return nil, err
	}
	if cleanupOutput {
		if err := r.removeUnpublishedPostOutput(allPosts); err != nil {
			return nil, err
		}
	}
	posts, err := repo.ListPublishedPosts()
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *Renderer) renderHomepage(latestPosts []content.Post, cfg siteconfig.Config, style styleTemplateData) (string, error) {
	data := sitePageData{
		Site:   r.siteData(cfg, style),
		Title:  cfg.Title,
		Latest: r.summarizePosts(latestPosts, false),
	}

	var document bytes.Buffer
	if err := homepageTemplate.Execute(&document, data); err != nil {
		return "", err
	}
	return document.String(), nil
}

func (r *Renderer) renderFeed(posts []content.Post) ([]byte, error) {
	channel := rssChannel{
		Title:       r.siteConfig.Title,
		Link:        r.absoluteURL("/"),
		Description: r.siteConfig.Description,
		Items:       make([]rssItem, 0, len(posts)),
	}
	for _, post := range posts {
		channel.Items = append(channel.Items, rssItem{
			Title:       post.Title,
			Link:        r.postURL(post.Slug),
			GUID:        rssGUID{Value: r.postURL(post.Slug), IsPermaLink: "true"},
			Description: post.Description,
			PubDate:     post.PublishedAt.UTC().Format(timeFormatRSS),
		})
	}
	data, err := xml.MarshalIndent(rssDocument{
		Version: "2.0",
		Channel: channel,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), data...), nil
}

func (r *Renderer) renderSitemap(posts []content.Post) ([]byte, error) {
	urls := []sitemapURL{{
		Location: r.absoluteURL("/"),
	}}
	for _, post := range posts {
		urls = append(urls, sitemapURL{
			Location:     r.postURL(post.Slug),
			LastModified: post.UpdatedAt.UTC().Format("2006-01-02"),
		})
	}
	data, err := xml.MarshalIndent(sitemap{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), data...), nil
}

func (r *Renderer) removeUnpublishedPostOutput(posts []content.Post) error {
	for _, post := range posts {
		if post.IsPublished() {
			continue
		}
		if err := os.RemoveAll(filepath.Join(r.publicRoot, postsDirName, post.Slug)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) summarizePosts(posts []content.Post, absolute bool) []postSummary {
	summaries := make([]postSummary, 0, len(posts))
	for _, post := range posts {
		postPath := "/posts/" + post.Slug + "/"
		coverURL := ""
		if post.Cover != "" {
			coverURL = postPath + post.Cover
			if absolute {
				coverURL = r.absoluteURL(coverURL)
			}
		}
		url := postPath
		if absolute {
			url = r.absoluteURL(url)
		}
		summaries = append(summaries, postSummary{
			Slug:        post.Slug,
			Title:       post.Title,
			Description: post.Description,
			URL:         url,
			CoverURL:    coverURL,
			PublishedAt: formatOptionalTime(post.PublishedAt, "2006-01-02"),
			UpdatedAt:   formatOptionalTime(post.UpdatedAt, "2006-01-02"),
		})
	}
	return summaries
}

func formatOptionalTime(value time.Time, layout string) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(layout)
}

func normalizeMarkdownSource(source string) string {
	if source == "" {
		return ""
	}
	lines := strings.SplitAfter(source, "\n")
	var normalized strings.Builder
	for _, line := range lines {
		body, ending := splitLineEnding(line)
		if isStandaloneBreakTag(body) {
			normalized.WriteString(ending)
			continue
		}
		normalized.WriteString(line)
	}
	return normalized.String()
}

func splitLineEnding(line string) (string, string) {
	switch {
	case strings.HasSuffix(line, "\r\n"):
		return strings.TrimSuffix(line, "\r\n"), "\r\n"
	case strings.HasSuffix(line, "\n"):
		return strings.TrimSuffix(line, "\n"), "\n"
	default:
		return line, ""
	}
}

func isStandaloneBreakTag(line string) bool {
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "<br>", "<br/>", "<br />":
		return true
	default:
		return false
	}
}

func (r *Renderer) reconcileCover(post content.Post, publicDir string) (string, error) {
	entries, err := os.ReadDir(publicDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() || !isCoverFile(entry.Name()) {
			continue
		}
		if err := os.Remove(filepath.Join(publicDir, entry.Name())); err != nil {
			return "", err
		}
	}
	if post.Cover == "" {
		return "", nil
	}

	source := filepath.Join(r.contentRoot, postsDirName, post.Slug, post.Cover)
	destination := filepath.Join(publicDir, post.Cover)
	if err := copyFile(destination, source); err != nil {
		return "", err
	}
	return destination, nil
}

func (r *Renderer) reconcileAssets(post content.Post, publicDir string) ([]string, error) {
	publicAssetsDir := filepath.Join(publicDir, assetsDirName)
	if err := os.RemoveAll(publicAssetsDir); err != nil {
		return nil, err
	}
	if len(post.Assets) == 0 {
		return nil, nil
	}
	if err := os.MkdirAll(publicAssetsDir, directoryMode); err != nil {
		return nil, err
	}

	assets := append([]string(nil), post.Assets...)
	sort.Strings(assets)
	for _, asset := range assets {
		cleaned, err := content.CleanAssetPath(asset)
		if err != nil {
			return nil, err
		}
		source := filepath.Join(r.contentRoot, postsDirName, post.Slug, assetsDirName, filepath.FromSlash(cleaned))
		destination := filepath.Join(publicAssetsDir, filepath.FromSlash(cleaned))
		if err := copyFile(destination, source); err != nil {
			return nil, err
		}
	}
	return assets, nil
}

func (r *Renderer) absoluteURL(path string) string {
	return path
}

func (r *Renderer) postURL(slug string) string {
	return r.absoluteURL("/posts/" + slug + "/")
}

func (r *Renderer) siteData(cfg siteconfig.Config, style styleTemplateData) siteTemplateData {
	cfg = siteconfig.WithDefaults(cfg)
	return siteTemplateData{
		Title:       cfg.Title,
		Description: cfg.Description,
		FaviconHref: faviconHref(cfg.Favicon),
		Style:       style,
		Header: headerTemplateData{
			Title: cfg.Title,
			Links: cfg.Header.Links,
		},
		Footer: footerTemplateData{
			Text:          cfg.Footer.Text,
			ShowWatermark: cfg.Footer.ShowWatermark,
			Links:         cfg.Footer.Links,
		},
	}
}

func (r *Renderer) writeStyleSheet() (string, error) {
	path := filepath.Join(r.publicRoot, "assets", "styxpress.css")
	return path, writeAtomic(path, []byte(styleSheet(r.siteConfig)))
}

func (r *Renderer) writeFavicon() (string, error) {
	cfg := siteconfig.WithDefaults(r.siteConfig)
	faviconPath, err := siteconfig.CleanFaviconPath(cfg.Favicon)
	if err != nil {
		return "", err
	}

	destination := filepath.Join(r.publicRoot, filepath.FromSlash(faviconPath))
	source := filepath.Join(r.contentRoot, filepath.FromSlash(faviconPath))
	if err := copyFile(destination, source); err != nil {
		if errors.Is(err, os.ErrNotExist) && faviconPath == siteconfig.DefaultFaviconPath {
			return destination, writeAtomic(destination, defaultFavicon)
		}
		return "", err
	}
	return destination, nil
}

func linkedStyle() styleTemplateData {
	return styleTemplateData{Href: "/assets/styxpress.css"}
}

func inlineStyle() styleTemplateData {
	return styleTemplateData{InlineCSS: template.CSS(styleElementCSS(siteStyleSheet))}
}

func faviconHref(faviconPath string) template.URL {
	cleaned, err := siteconfig.CleanFaviconPath(faviconPath)
	if err != nil {
		return ""
	}
	return template.URL((&url.URL{Path: "/" + cleaned}).EscapedPath())
}

func styleSheet(cfg siteconfig.Config) string {
	return siteStyleSheet
}

func styleElementCSS(css string) string {
	lower := strings.ToLower(css)
	var output strings.Builder
	start := 0
	for {
		index := strings.Index(lower[start:], "</style")
		if index == -1 {
			output.WriteString(css[start:])
			return output.String()
		}
		index += start
		output.WriteString(css[start:index])
		output.WriteString("<\\/")
		output.WriteString(css[index+2 : index+len("</style")])
		start = index + len("</style")
	}
}

func copyFile(destination string, source string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: %s", ErrUnsafeAsset, source)
	}

	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	destinationDir := filepath.Dir(destination)
	if err := os.MkdirAll(destinationDir, directoryMode); err != nil {
		return err
	}
	output, err := os.CreateTemp(destinationDir, "."+filepath.Base(destination)+".*")
	if err != nil {
		return err
	}
	temp := output.Name()
	_, copyErr := io.Copy(output, input)
	chmodErr := output.Chmod(fileMode)
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(temp)
		return copyErr
	}
	if chmodErr != nil {
		_ = os.Remove(temp)
		return chmodErr
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return closeErr
	}
	return os.Rename(temp, destination)
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	temp := file.Name()
	_, writeErr := file.Write(data)
	chmodErr := file.Chmod(fileMode)
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(temp)
		return writeErr
	}
	if chmodErr != nil {
		_ = os.Remove(temp)
		return chmodErr
	}
	if closeErr != nil {
		_ = os.Remove(temp)
		return closeErr
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

func isCoverFile(name string) bool {
	switch strings.ToLower(name) {
	case "cover.jpg", "cover.jpeg", "cover.png", "cover.webp", "cover.avif":
		return true
	default:
		return false
	}
}

type escapedHTMLRenderer struct{}

func (r escapedHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
}

func (r escapedHTMLRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	block := node.(*ast.HTMLBlock)
	if entering {
		for i := 0; i < block.Lines().Len(); i++ {
			line := block.Lines().At(i)
			_, _ = w.WriteString(stdhtml.EscapeString(string(line.Value(source))))
		}
		return ast.WalkContinue, nil
	}
	if block.HasClosure() {
		closure := block.ClosureLine
		_, _ = w.WriteString(stdhtml.EscapeString(string(closure.Value(source))))
	}
	return ast.WalkContinue, nil
}

func (r escapedHTMLRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	raw := node.(*ast.RawHTML)
	for i := 0; i < raw.Segments.Len(); i++ {
		segment := raw.Segments.At(i)
		_, _ = w.WriteString(stdhtml.EscapeString(string(segment.Value(source))))
	}
	return ast.WalkSkipChildren, nil
}

var postTemplate = template.Must(template.New("post").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{ .Title }}</title>
{{- if .Site.FaviconHref }}
<link rel="icon" href="{{ .Site.FaviconHref }}" type="image/x-icon">
{{- end }}
{{- if .Site.Style.InlineCSS }}
<style>
{{ .Site.Style.InlineCSS }}</style>
{{- else }}
<link rel="stylesheet" href="{{ .Site.Style.Href }}">
{{- end }}
{{- if .Description }}
<meta name="description" content="{{ .Description }}">
{{- end }}
{{- if .CanonicalURL }}
<link rel="canonical" href="{{ .CanonicalURL }}">
{{- end }}
<meta property="og:type" content="article">
<meta property="og:title" content="{{ .Title }}">
{{- if .Description }}
<meta property="og:description" content="{{ .Description }}">
{{- end }}
{{- if .CanonicalURL }}
<meta property="og:url" content="{{ .CanonicalURL }}">
{{- end }}
{{- if .OpenGraphImage }}
<meta property="og:image" content="{{ .OpenGraphImage }}">
{{- end }}
{{- if .PublishedAt }}
<meta property="article:published_time" content="{{ .PublishedAt }}">
{{- end }}
{{- if .UpdatedAt }}
<meta property="article:modified_time" content="{{ .UpdatedAt }}">
{{- end }}
</head>
<body>
<a class="skip-link" href="#content">Skip to content</a>
<header class="site-header">
<div class="site-header-inner">
<div class="site-branding">
<a class="site-title" href="/">{{ .Site.Header.Title }}</a>
</div>
{{- if .Site.Header.Links }}
<nav class="site-nav" aria-label="Primary">
{{- range .Site.Header.Links }}
<a href="{{ .Href }}">{{ .Label }}</a>
{{- end }}
</nav>
{{- end }}
</div>
</header>
<main id="content" class="site-main post-main">
<article class="post-article">
<header class="post-header">
{{- if .CoverURL }}
<img src="{{ .CoverURL }}" alt="">
{{- end }}
<h1>{{ .Title }}</h1>
{{- if .Description }}
<p>{{ .Description }}</p>
{{- end }}
</header>
<div class="post-content">
{{ .ArticleHTML }}
</div>
</article>
</main>
<footer class="site-footer">
<div class="site-footer-inner">
{{- if .Site.Footer.Text }}
<p>{{ .Site.Footer.Text }}</p>
{{- end }}
{{- if .Site.Footer.ShowWatermark }}
<p>Published with <a href="https://styxpress.anordine.com">Styxpress</a></p>
{{- end }}
{{- if .Site.Footer.Links }}
<nav class="site-nav" aria-label="Footer">
{{- range .Site.Footer.Links }}
<a href="{{ .Href }}">{{ .Label }}</a>
{{- end }}
</nav>
{{- end }}
</div>
</footer>
</body>
</html>
`))

var homepageTemplate = template.Must(template.New("homepage").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{ .Title }}</title>
{{- if .Site.FaviconHref }}
<link rel="icon" href="{{ .Site.FaviconHref }}" type="image/x-icon">
{{- end }}
{{- if .Site.Style.InlineCSS }}
<style>
{{ .Site.Style.InlineCSS }}</style>
{{- else }}
<link rel="stylesheet" href="{{ .Site.Style.Href }}">
{{- end }}
{{- if .Site.Description }}
<meta name="description" content="{{ .Site.Description }}">
{{- end }}
</head>
<body>
<a class="skip-link" href="#content">Skip to content</a>
<header class="site-header">
<div class="site-header-inner">
<div class="site-branding">
<a class="site-title" href="/">{{ .Site.Header.Title }}</a>
</div>
{{- if .Site.Header.Links }}
<nav class="site-nav" aria-label="Primary">
{{- range .Site.Header.Links }}
<a href="{{ .Href }}">{{ .Label }}</a>
{{- end }}
</nav>
{{- end }}
</div>
</header>
<main id="content" class="site-main home-main">
<section class="post-section" aria-labelledby="latest-posts">
<h1 id="latest-posts">Latest Posts</h1>
{{- if .Latest }}
<div class="post-list">
{{- range .Latest }}
<article class="post-card">
{{- if .CoverURL }}
<img src="{{ .CoverURL }}" alt="">
{{- end }}
<div>
<h2><a href="{{ .URL }}">{{ .Title }}</a></h2>
{{- if .Description }}
<p>{{ .Description }}</p>
{{- end }}
<time datetime="{{ .PublishedAt }}">{{ .PublishedAt }}</time>
</div>
</article>
{{- end }}
</div>
{{- else }}
<p>No posts yet.</p>
{{- end }}
</section>
</main>
<footer class="site-footer">
<div class="site-footer-inner">
{{- if .Site.Footer.Text }}
<p>{{ .Site.Footer.Text }}</p>
{{- end }}
{{- if .Site.Footer.ShowWatermark }}
<p>Published with <a href="https://styxpress.anordine.com">Styxpress</a></p>
{{- end }}
{{- if .Site.Footer.Links }}
<nav class="site-nav" aria-label="Footer">
{{- range .Site.Footer.Links }}
<a href="{{ .Href }}">{{ .Label }}</a>
{{- end }}
</nav>
{{- end }}
</div>
</footer>
</body>
</html>
`))

const siteStyleSheet = `:root {
    color-scheme: light;
    --site-bg: #ffffff;
    --site-surface: #ffffff;
    --site-text: #262626;
    --site-muted: #6f6f6f;
    --site-heading: #111111;
    --site-accent: #0f766e;
    --site-border: #e5e5e5;
    --site-radius: 6px;
    --site-width: 760px;
}

*,
*::before,
*::after {
    box-sizing: border-box;
}

body {
    min-width: 320px;
    margin: 0;
    background: var(--site-bg);
    color: var(--site-text);
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    line-height: 1.68;
    text-rendering: optimizeLegibility;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
}

a {
    color: var(--site-accent);
    text-decoration-thickness: 0.08em;
    text-underline-offset: 0.18em;
}

img {
    max-width: 100%;
    height: auto;
}

.skip-link {
    position: absolute;
    left: 1rem;
    top: 0;
    transform: translateY(-120%);
    border-radius: var(--site-radius);
    padding: 0.5rem 0.75rem;
    background: var(--site-heading);
    color: var(--site-bg);
}

.skip-link:focus {
    transform: translateY(1rem);
}

.site-header {
    border-bottom: 1px solid var(--site-border);
}

.site-footer {
    border-top: 1px solid var(--site-border);
}

.site-header-inner,
.site-footer-inner,
.site-main {
    width: min(calc(100% - 2rem), var(--site-width));
    margin: 0 auto;
}

.site-header-inner,
.site-footer-inner {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    align-items: center;
    justify-content: space-between;
    padding: 1rem 0;
}

.site-branding {
    display: grid;
    gap: 0.1rem;
}

.site-title {
    color: var(--site-heading);
    font-size: 1.05rem;
    font-weight: 750;
    text-decoration: none;
}

.site-footer p,
.post-card p,
.post-header p {
    margin: 0;
    color: var(--site-muted);
}

.site-nav {
    display: flex;
    flex-wrap: wrap;
    gap: 0.85rem;
    align-items: center;
}

.site-nav a {
    color: var(--site-text);
    font-size: 0.95rem;
    font-weight: 650;
    text-decoration: none;
}

.site-nav a:hover {
    color: var(--site-accent);
}

.site-main {
    padding: 3rem 0;
}

.post-section {
    display: grid;
    gap: 1rem;
}

.post-section h1,
.post-header h1 {
    margin: 0;
    color: var(--site-heading);
    line-height: 1.12;
}

.post-header {
    display: grid;
    gap: 1rem;
    margin-bottom: 2rem;
}

.post-header img,
.post-card img {
    border-radius: var(--site-radius);
}

.post-header img {
    display: block;
    width: 100%;
    height: clamp(12rem, 42vw, 24rem);
    max-height: 24rem;
    aspect-ratio: 16 / 9;
    object-fit: cover;
    object-position: center;
}

.post-content > * + * {
    margin-top: 1rem;
}

.post-content img {
    display: block;
    width: auto;
    max-width: min(100%, 32rem);
    max-height: 30rem;
    margin-right: auto;
    margin-left: auto;
    border-radius: var(--site-radius);
    object-fit: contain;
}

.post-content img[src$="#small"] {
    max-width: min(100%, 18rem);
    max-height: 18rem;
}

.post-content img[src$="#medium"] {
    max-width: min(100%, 32rem);
    max-height: 30rem;
}

.post-content img[src$="#large"] {
    max-width: min(100%, 44rem);
    max-height: 36rem;
}

.post-content img[src$="#full"] {
    width: 100%;
    max-width: 100%;
    max-height: 42rem;
}

.post-content h1,
.post-content h2,
.post-content h3,
.post-card h2 {
    color: var(--site-heading);
    line-height: 1.2;
}

.post-content pre,
.post-content code {
    border-radius: var(--site-radius);
    background: #f6f6f6;
}

.post-content code {
    padding: 0.12rem 0.25rem;
}

.post-content pre {
    overflow-x: auto;
    padding: 1rem;
}

.post-content pre code {
    padding: 0;
    background: transparent;
}

.post-list {
    display: grid;
    gap: 1rem;
}

.post-card {
    display: grid;
    gap: 1rem;
    border: 1px solid var(--site-border);
    border-radius: var(--site-radius);
    padding: 1rem;
    background: var(--site-surface);
}

.post-card img {
    display: block;
    width: 100%;
    height: clamp(9rem, 42vw, 12rem);
    max-height: 12rem;
    aspect-ratio: 16 / 9;
    object-fit: cover;
    object-position: center;
}

.post-card h2 {
    margin: 0 0 0.35rem;
    font-size: 1.1rem;
}

.post-card time {
    color: var(--site-muted);
    font-size: 0.9rem;
}

@media (min-width: 760px) {
    .post-card {
        grid-template-columns: minmax(0, 10rem) minmax(0, 1fr);
        align-items: start;
    }

    .post-card img {
        width: 10rem;
        height: 6.25rem;
        max-height: none;
    }

    .post-card:not(:has(img)) {
        grid-template-columns: 1fr;
    }
}
`
