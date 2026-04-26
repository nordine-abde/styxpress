package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/nordine-abde/styxpress/internal/config"
	"github.com/nordine-abde/styxpress/internal/content"
	"github.com/nordine-abde/styxpress/internal/publishing"
	"github.com/nordine-abde/styxpress/internal/rendering"
)

const SessionHeader = "X-Styxpress-Session"

const maxUploadBytes = 64 << 20

type Server struct {
	configPath    string
	token         string
	logger        *log.Logger
	sshTester     func(*http.Request, config.Config, string) error
	publishRunner func(*http.Request, config.Config, string) (publishing.Result, error)
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(configPath string, logger *log.Logger) (*Server, error) {
	token, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Server{
		configPath: configPath,
		token:      token,
		logger:     logger,
		sshTester: func(r *http.Request, cfg config.Config, passphrase string) error {
			return publishing.TestSSH(r.Context(), cfg, passphrase)
		},
		publishRunner: func(r *http.Request, cfg config.Config, passphrase string) (publishing.Result, error) {
			return publishing.New(cfg, nil).Publish(r.Context(), publishing.Options{Passphrase: passphrase})
		},
	}, nil
}

func (s *Server) Token() string {
	return s.token
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.withAuth(s.health))
	mux.HandleFunc("GET /api/config", s.withAuth(s.getConfig))
	mux.HandleFunc("POST /api/config", s.withAuth(s.saveConfig))
	mux.HandleFunc("POST /api/test-ssh", s.withAuth(s.testSSH))
	mux.HandleFunc("GET /api/posts", s.withAuth(s.listPosts))
	mux.HandleFunc("POST /api/posts", s.withAuth(s.savePost))
	mux.HandleFunc("GET /api/posts/{slug}", s.withAuth(s.getPost))
	mux.HandleFunc("POST /api/posts/{slug}", s.withAuth(s.savePost))
	mux.HandleFunc("POST /api/posts/{slug}/cover", s.withAuth(s.uploadCover))
	mux.HandleFunc("DELETE /api/posts/{slug}/cover", s.withAuth(s.deleteCover))
	mux.HandleFunc("POST /api/posts/{slug}/assets", s.withAuth(s.uploadAsset))
	mux.HandleFunc("DELETE /api/posts/{slug}/assets/{assetPath...}", s.withAuth(s.deleteAsset))
	mux.HandleFunc("POST /api/render-preview", s.withAuth(s.renderPreview))
	mux.HandleFunc("POST /api/posts/{slug}/render", s.withAuth(s.renderPost))
	mux.HandleFunc("POST /api/posts/{slug}/publish", s.withAuth(s.publishPost))
	mux.HandleFunc("POST /api/publish", s.withAuth(s.publishPost))
	mux.HandleFunc("GET /api/featured", s.withAuth(s.getFeatured))
	mux.HandleFunc("POST /api/featured", s.withAuth(s.saveFeatured))
	return mux
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r) {
			WriteError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid session token")
			return
		}
		next(w, r)
	}
}

func (s *Server) authenticated(r *http.Request) bool {
	token := r.Header.Get(SessionHeader)
	if token == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	return token != "" && token == s.token
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	cfg, err := config.LoadOrDefault(s.configPath)
	if err != nil {
		s.logger.Printf("load config: %v", err)
		WriteError(w, http.StatusInternalServerError, "config_load_failed", "failed to load config")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) saveConfig(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var cfg config.Config
	if err := decodeJSONBody(r, &cfg, "request body must be a valid config object"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	if err := config.Save(s.configPath, cfg); err != nil {
		if errors.Is(err, config.ErrInvalidConfig) {
			WriteError(w, http.StatusBadRequest, "invalid_config", err.Error())
			return
		}
		s.logger.Printf("save config: %v", err)
		WriteError(w, http.StatusInternalServerError, "config_save_failed", "failed to save config")
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

type testSSHRequest struct {
	Passphrase string `json:"passphrase"`
}

type postPayload struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	Cover       string   `json:"cover"`
	Assets      []string `json:"assets"`
	PublishedAt string   `json:"publishedAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type postResponse struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Source      string   `json:"source,omitempty"`
	Cover       string   `json:"cover,omitempty"`
	Assets      []string `json:"assets"`
	PublishedAt string   `json:"publishedAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type postListResponse struct {
	Posts []postResponse `json:"posts"`
}

type previewResponse struct {
	HTML string `json:"html"`
}

type publishRequest struct {
	Slug       string `json:"slug"`
	Passphrase string `json:"passphrase"`
}

type renderPostResponse struct {
	Post rendering.Result     `json:"post"`
	Site rendering.SiteResult `json:"site"`
}

type publishResponse struct {
	Post    rendering.Result     `json:"post"`
	Site    rendering.SiteResult `json:"site"`
	Publish publishing.Result    `json:"publish"`
}

type featuredRequest struct {
	Slugs []string `json:"slugs"`
}

type featuredResponse struct {
	Slugs []string `json:"slugs"`
}

func (s *Server) testSSH(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req testSSHRequest
	if r.Body != http.NoBody {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be an object with an optional passphrase")
			return
		}
	}

	cfg, err := config.LoadOrDefault(s.configPath)
	if err != nil {
		s.logger.Printf("load config: %v", err)
		WriteError(w, http.StatusInternalServerError, "config_load_failed", "failed to load config")
		return
	}

	if err := s.sshTester(r, cfg, req.Passphrase); err != nil {
		if errors.Is(err, publishing.ErrInvalidPublishConfig) {
			WriteError(w, http.StatusBadRequest, "invalid_publish_config", err.Error())
			return
		}
		s.logger.Printf("test SSH: %v", err)
		WriteError(w, http.StatusBadGateway, "ssh_test_failed", "failed to connect to the configured SSH server")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) listPosts(w http.ResponseWriter, _ *http.Request) {
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	posts, err := repo.ListPosts()
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	response := postListResponse{Posts: make([]postResponse, 0, len(posts))}
	for _, post := range posts {
		response.Posts = append(response.Posts, newPostResponse(post, false))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) getPost(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	post, err := repo.LoadPost(r.PathValue("slug"))
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newPostResponse(post, true))
}

func (s *Server) savePost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload postPayload
	if err := decodeJSONBody(r, &payload, "request body must be a post object"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if slug := r.PathValue("slug"); slug != "" {
		if payload.Slug != "" && payload.Slug != slug {
			WriteError(w, http.StatusBadRequest, "invalid_post", "request slug must match URL slug")
			return
		}
		payload.Slug = slug
	}

	post, err := payload.toPost()
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_post", err.Error())
		return
	}
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	saved, err := repo.WritePost(post, content.WritePostOptions{})
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newPostResponse(saved, true))
}

func (s *Server) uploadCover(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	file, header, ok := readUpload(w, r, false)
	if !ok {
		return
	}
	defer file.Close()

	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := repo.WriteCover(slug, filepath.Base(header.Filename), file); err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"cover": filepath.Base(header.Filename)})
}

func (s *Server) deleteCover(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := repo.DeleteCover(r.PathValue("slug")); err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) uploadAsset(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	file, header, ok := readUpload(w, r, true)
	if !ok {
		return
	}
	defer file.Close()

	assetPath := strings.TrimSpace(r.FormValue("path"))
	if assetPath == "" {
		assetPath = header.Filename
	}
	cleaned, err := content.CleanAssetPath(assetPath)
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := repo.WriteAsset(slug, cleaned, file); err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"asset": cleaned})
}

func (s *Server) deleteAsset(w http.ResponseWriter, r *http.Request) {
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := repo.DeleteAsset(r.PathValue("slug"), r.PathValue("assetPath")); err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) renderPreview(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload postPayload
	if err := decodeJSONBody(r, &payload, "request body must be a post object"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	post, err := payload.toPost()
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_post", err.Error())
		return
	}
	now := time.Now().UTC()
	if post.PublishedAt.IsZero() {
		post.PublishedAt = now
	}
	if post.UpdatedAt.IsZero() {
		post.UpdatedAt = post.PublishedAt
	}

	renderer, err := s.renderer()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	html, err := renderer.RenderPreview(post)
	if err != nil {
		s.writeRenderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, previewResponse{HTML: html})
}

func (s *Server) renderPost(w http.ResponseWriter, r *http.Request) {
	result, site, err := s.renderPostAndSite(r.PathValue("slug"))
	if err != nil {
		s.writeRenderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, renderPostResponse{Post: result, Site: site})
}

func (s *Server) publishPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	req := publishRequest{Slug: r.PathValue("slug")}
	if r.Body != http.NoBody {
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be an object with optional slug and passphrase")
			return
		}
	}
	if req.Slug == "" {
		WriteError(w, http.StatusBadRequest, "invalid_publish", "slug is required")
		return
	}

	postResult, siteResult, err := s.renderPostAndSite(req.Slug)
	if err != nil {
		s.writeRenderError(w, err)
		return
	}
	cfg, err := s.loadConfig()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	cfg, err = normalizeLocalPaths(cfg)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	publishResult, err := s.publishRunner(r, cfg, req.Passphrase)
	if err != nil {
		s.writePublishError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publishResponse{
		Post:    postResult,
		Site:    siteResult,
		Publish: publishResult,
	})
}

func (s *Server) getFeatured(w http.ResponseWriter, _ *http.Request) {
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	slugs, err := repo.ReadFeatured()
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, featuredResponse{Slugs: slugs})
}

func (s *Server) saveFeatured(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req featuredRequest
	if err := decodeJSONBody(r, &req, "request body must be an object with slugs"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := repo.WriteFeatured(req.Slugs); err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, featuredResponse{Slugs: req.Slugs})
}

func (s *Server) renderPostAndSite(slug string) (rendering.Result, rendering.SiteResult, error) {
	renderer, err := s.renderer()
	if err != nil {
		return rendering.Result{}, rendering.SiteResult{}, err
	}
	postResult, err := renderer.RenderPost(slug)
	if err != nil {
		return rendering.Result{}, rendering.SiteResult{}, err
	}
	siteResult, err := renderer.RenderSite()
	if err != nil {
		return rendering.Result{}, rendering.SiteResult{}, err
	}
	return postResult, siteResult, nil
}

func (s *Server) repository() (*content.Repository, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}
	contentDir, err := configuredPath(cfg.ContentDir)
	if err != nil {
		return nil, err
	}
	return content.NewRepository(contentDir), nil
}

func (s *Server) renderer() (*rendering.Renderer, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}
	contentDir, err := configuredPath(cfg.ContentDir)
	if err != nil {
		return nil, err
	}
	publicDir, err := configuredPath(cfg.PublicDir)
	if err != nil {
		return nil, err
	}
	return rendering.New(contentDir, publicDir, rendering.Options{SiteBaseURL: cfg.SiteBaseURL})
}

func normalizeLocalPaths(cfg config.Config) (config.Config, error) {
	contentDir, err := configuredPath(cfg.ContentDir)
	if err != nil {
		return config.Config{}, err
	}
	publicDir, err := configuredPath(cfg.PublicDir)
	if err != nil {
		return config.Config{}, err
	}
	cfg.ContentDir = contentDir
	cfg.PublicDir = publicDir
	return cfg, nil
}

func (s *Server) loadConfig() (config.Config, error) {
	return config.LoadOrDefault(s.configPath)
}

func (s *Server) writeConfigPathError(w http.ResponseWriter, err error) {
	if errors.Is(err, config.ErrInvalidConfig) || errors.Is(err, ErrInvalidLocalPath) {
		WriteError(w, http.StatusBadRequest, "invalid_config", err.Error())
		return
	}
	s.logger.Printf("config path error: %v", err)
	WriteError(w, http.StatusInternalServerError, "config_load_failed", "failed to load config")
}

func (s *Server) writeContentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, content.ErrPostNotFound):
		WriteError(w, http.StatusNotFound, "post_not_found", "post not found")
	case errors.Is(err, content.ErrInvalidSlug), errors.Is(err, content.ErrInvalidPost), errors.Is(err, content.ErrPostExists), errors.Is(err, content.ErrDuplicateCover), errors.Is(err, content.ErrUnsupportedCover), errors.Is(err, content.ErrInvalidAsset), errors.Is(err, content.ErrInvalidAssetPath):
		WriteError(w, http.StatusBadRequest, "invalid_content", err.Error())
	default:
		s.logger.Printf("content error: %v", err)
		WriteError(w, http.StatusInternalServerError, "content_failed", "failed to access content")
	}
}

func (s *Server) writeRenderError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidLocalPath), errors.Is(err, config.ErrInvalidConfig):
		WriteError(w, http.StatusBadRequest, "invalid_config", err.Error())
	case errors.Is(err, content.ErrPostNotFound):
		WriteError(w, http.StatusNotFound, "post_not_found", "post not found")
	case errors.Is(err, rendering.ErrInvalidRenderConfig), errors.Is(err, rendering.ErrUnsafeAsset), errors.Is(err, content.ErrInvalidSlug), errors.Is(err, content.ErrInvalidPost), errors.Is(err, content.ErrInvalidAssetPath), errors.Is(err, content.ErrInvalidAsset):
		WriteError(w, http.StatusBadRequest, "render_failed", err.Error())
	default:
		s.logger.Printf("render error: %v", err)
		WriteError(w, http.StatusInternalServerError, "render_failed", "failed to render post")
	}
}

func (s *Server) writePublishError(w http.ResponseWriter, err error) {
	var uploadErr *publishing.UploadError
	switch {
	case errors.As(err, &uploadErr):
		writeJSON(w, http.StatusBadGateway, ErrorResponse{Error: ErrorBody{
			Code:    "publish_upload_failed",
			Message: fmt.Sprintf("failed to upload %s; cleanup paths: %s", uploadErr.Path, strings.Join(uploadErr.CleanupPaths, ", ")),
		}})
	case errors.Is(err, publishing.ErrInvalidPublishConfig):
		WriteError(w, http.StatusBadRequest, "invalid_publish_config", err.Error())
	default:
		s.logger.Printf("publish error: %v", err)
		WriteError(w, http.StatusBadGateway, "publish_failed", "failed to publish files")
	}
}

var ErrInvalidLocalPath = errors.New("invalid local path")

func configuredPath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%w: path is required", ErrInvalidLocalPath)
	}
	if strings.Contains(value, "\x00") {
		return "", fmt.Errorf("%w: path contains NUL byte", ErrInvalidLocalPath)
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func readUpload(w http.ResponseWriter, r *http.Request, allowPathOverride bool) (multipart.File, *multipart.FileHeader, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_upload", "request must be multipart/form-data with a file field")
		return nil, nil, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_upload", "file field is required")
		return nil, nil, false
	}
	if strings.TrimSpace(header.Filename) == "" || filepath.Base(header.Filename) != header.Filename && (!allowPathOverride || r.FormValue("path") == "") {
		_ = file.Close()
		WriteError(w, http.StatusBadRequest, "invalid_upload", "uploaded filename must be a simple file name unless an asset path is provided")
		return nil, nil, false
	}
	return file, header, true
}

func decodeJSONBody(r *http.Request, target any, message string) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errors.New(message)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func (p postPayload) toPost() (content.Post, error) {
	publishedAt, err := parseOptionalTime(p.PublishedAt)
	if err != nil {
		return content.Post{}, err
	}
	updatedAt, err := parseOptionalTime(p.UpdatedAt)
	if err != nil {
		return content.Post{}, err
	}
	return content.Post{
		Slug:        p.Slug,
		Title:       p.Title,
		Description: p.Description,
		Source:      p.Source,
		Cover:       p.Cover,
		Assets:      p.Assets,
		PublishedAt: publishedAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func parseOptionalTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("time values must be RFC3339")
	}
	return parsed, nil
}

func newPostResponse(post content.Post, includeSource bool) postResponse {
	response := postResponse{
		Slug:        post.Slug,
		Title:       post.Title,
		Description: post.Description,
		Cover:       post.Cover,
		Assets:      append([]string(nil), post.Assets...),
		PublishedAt: post.PublishedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   post.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if includeSource {
		response.Source = post.Source
	}
	return response
}

func WriteError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func newSessionToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
