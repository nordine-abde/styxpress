package rendering

import (
	"bytes"
	"errors"
	"fmt"
	stdhtml "html"
	"html/template"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nordine-abde/styxpress/internal/content"
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
)

var (
	ErrInvalidRenderConfig = errors.New("invalid render config")
	ErrUnsafeAsset         = errors.New("unsafe asset")
)

type Renderer struct {
	contentRoot string
	publicRoot  string
	siteBaseURL string
	markdown    goldmark.Markdown
}

type Options struct {
	SiteBaseURL string
}

type Result struct {
	Slug      string
	PublicDir string
	IndexPath string
	CoverPath string
	Assets    []string
}

type pageData struct {
	Title          string
	Description    string
	CanonicalURL   string
	OpenGraphImage string
	CoverURL       string
	PublishedAt    string
	UpdatedAt      string
	ArticleHTML    template.HTML
}

func New(contentRoot string, publicRoot string, opts Options) (*Renderer, error) {
	if strings.TrimSpace(contentRoot) == "" {
		return nil, fmt.Errorf("%w: content root is required", ErrInvalidRenderConfig)
	}
	if strings.TrimSpace(publicRoot) == "" {
		return nil, fmt.Errorf("%w: public root is required", ErrInvalidRenderConfig)
	}
	siteBaseURL := strings.TrimRight(strings.TrimSpace(opts.SiteBaseURL), "/")
	if siteBaseURL != "" {
		parsed, err := url.Parse(siteBaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, fmt.Errorf("%w: site base URL must be absolute", ErrInvalidRenderConfig)
		}
	}

	return &Renderer{
		contentRoot: contentRoot,
		publicRoot:  publicRoot,
		siteBaseURL: siteBaseURL,
		markdown: goldmark.New(
			goldmark.WithRendererOptions(
				goldmarkhtml.WithXHTML(),
				renderer.WithNodeRenderers(util.Prioritized(escapedHTMLRenderer{}, 900)),
			),
		),
	}, nil
}

func (r *Renderer) RenderPost(slug string) (Result, error) {
	repo := content.NewRepository(r.contentRoot)
	post, err := repo.LoadPost(slug)
	if err != nil {
		return Result{}, err
	}

	document, err := r.RenderPreview(post)
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
	if err := content.ValidateSlug(post.Slug); err != nil {
		return "", err
	}

	var article bytes.Buffer
	if err := r.markdown.Convert([]byte(post.Source), &article); err != nil {
		return "", err
	}

	basePostURL := "/posts/" + post.Slug
	coverURL := ""
	if post.Cover != "" {
		coverURL = basePostURL + "/" + post.Cover
	}

	data := pageData{
		Title:          post.Title,
		Description:    post.Description,
		CanonicalURL:   r.absoluteURL(basePostURL),
		OpenGraphImage: r.absoluteURL(coverURL),
		CoverURL:       coverURL,
		PublishedAt:    post.PublishedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      post.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		ArticleHTML:    template.HTML(article.String()),
	}

	var document bytes.Buffer
	if err := postTemplate.Execute(&document, data); err != nil {
		return "", err
	}
	return document.String(), nil
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
	if path == "" {
		return ""
	}
	if r.siteBaseURL == "" {
		return path
	}
	return r.siteBaseURL + path
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
<main>
<article>
<header>
<h1>{{ .Title }}</h1>
{{- if .Description }}
<p>{{ .Description }}</p>
{{- end }}
{{- if .CoverURL }}
<img src="{{ .CoverURL }}" alt="">
{{- end }}
</header>
{{ .ArticleHTML }}
</article>
</main>
</body>
</html>
`))
