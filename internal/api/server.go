package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nordine-abde/styxpress/internal/config"
	"github.com/nordine-abde/styxpress/internal/content"
	deploypkg "github.com/nordine-abde/styxpress/internal/deploy"
	"github.com/nordine-abde/styxpress/internal/localpath"
	"github.com/nordine-abde/styxpress/internal/rendering"
	"github.com/nordine-abde/styxpress/internal/siteconfig"
)

const SessionHeader = "X-Styxpress-Session"

const (
	maxUploadBytes      = 64 << 20
	siteAssetFileMode   = 0o644
	siteAssetFolderMode = 0o755
)

type Server struct {
	configPath     string
	siteStore      *config.SiteStore
	token          string
	logger         *log.Logger
	deploySecretMu sync.RWMutex
	deploySecret   string
}

type sitePaths struct {
	contentDir string
	publicDir  string
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(configPath string, logger *log.Logger) (*Server, error) {
	return newServer(configPath, nil, logger)
}

func NewDefault(logger *log.Logger) (*Server, error) {
	store, err := config.DefaultSiteStore()
	if err != nil {
		return nil, err
	}
	return newServer("", store, logger)
}

func newServer(configPath string, siteStore *config.SiteStore, logger *log.Logger) (*Server, error) {
	token, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Server{
		configPath: configPath,
		siteStore:  siteStore,
		token:      token,
		logger:     logger,
	}, nil
}

func (s *Server) Token() string {
	return s.token
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.withAuth(s.health))
	mux.HandleFunc("GET /api/sites", s.withAuth(s.listSites))
	mux.HandleFunc("GET /api/sites/suggestion", s.withAuth(s.suggestSite))
	mux.HandleFunc("POST /api/sites", s.withAuth(s.createSite))
	mux.HandleFunc("POST /api/sites/{id}/select", s.withAuth(s.selectSite))
	mux.HandleFunc("DELETE /api/sites/{id}", s.withAuth(s.deleteSite))
	mux.HandleFunc("GET /api/config", s.withAuth(s.getConfig))
	mux.HandleFunc("POST /api/config", s.withAuth(s.saveConfig))
	mux.HandleFunc("GET /api/site-config", s.withAuth(s.getSiteConfig))
	mux.HandleFunc("POST /api/site-config", s.withAuth(s.saveSiteConfig))
	mux.HandleFunc("POST /api/site-config/preview", s.withAuth(s.previewSiteConfig))
	mux.HandleFunc("POST /api/site-config/favicon", s.withAuth(s.uploadSiteFavicon))
	mux.HandleFunc("DELETE /api/site-config/favicon", s.withAuth(s.resetSiteFavicon))
	mux.HandleFunc("GET /api/posts", s.withAuth(s.listPosts))
	mux.HandleFunc("POST /api/posts", s.withAuth(s.savePost))
	mux.HandleFunc("GET /api/posts/{slug}", s.withAuth(s.getPost))
	mux.HandleFunc("POST /api/posts/{slug}", s.withAuth(s.savePost))
	mux.HandleFunc("GET /api/posts/{slug}/cover", s.withAuth(s.getCover))
	mux.HandleFunc("POST /api/posts/{slug}/cover", s.withAuth(s.uploadCover))
	mux.HandleFunc("DELETE /api/posts/{slug}/cover", s.withAuth(s.deleteCover))
	mux.HandleFunc("GET /api/posts/{slug}/assets/{assetPath...}", s.withAuth(s.getAsset))
	mux.HandleFunc("POST /api/posts/{slug}/assets", s.withAuth(s.uploadAsset))
	mux.HandleFunc("DELETE /api/posts/{slug}/assets/{assetPath...}", s.withAuth(s.deleteAsset))
	mux.HandleFunc("POST /api/render-preview", s.withAuth(s.renderPreview))
	mux.HandleFunc("POST /api/posts/{slug}/render", s.withAuth(s.renderPost))
	mux.HandleFunc("POST /api/posts/{slug}/publish", s.withAuth(s.publishPost))
	mux.HandleFunc("POST /api/site/render", s.withAuth(s.renderSite))
	mux.HandleFunc("POST /api/deploy/setup", s.withAuth(s.setupDeploy))
	mux.HandleFunc("GET /api/deploy/status", s.withAuth(s.deployStatus))
	mux.HandleFunc("POST /api/deploy/secret", s.withAuth(s.saveDeploySecret))
	mux.HandleFunc("DELETE /api/deploy/secret", s.withAuth(s.deleteDeploySecret))
	mux.HandleFunc("POST /api/deploy", s.withAuth(s.deployNow))
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

type sitesResponse struct {
	Sites        []config.Site `json:"sites"`
	ActiveSiteID string        `json:"activeSiteId"`
	MultiSite    bool          `json:"multiSite"`
}

type siteRequest struct {
	Name   string        `json:"name"`
	Config config.Config `json:"config"`
}

func (s *Server) listSites(w http.ResponseWriter, _ *http.Request) {
	sites, activeID, err := s.sites()
	if err != nil {
		s.writeSiteStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sitesResponse{
		Sites:        sites,
		ActiveSiteID: activeID,
		MultiSite:    s.siteStore != nil,
	})
}

func (s *Server) suggestSite(w http.ResponseWriter, r *http.Request) {
	if s.siteStore == nil {
		WriteError(w, http.StatusBadRequest, "site_registry_unavailable", "multiple sites are only available when styxpress-admin runs without -config")
		return
	}
	site, err := s.siteStore.Suggest(r.URL.Query().Get("name"))
	if err != nil {
		s.writeSiteStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) createSite(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if s.siteStore == nil {
		WriteError(w, http.StatusBadRequest, "site_registry_unavailable", "multiple sites are only available when styxpress-admin runs without -config")
		return
	}
	var req siteRequest
	if err := decodeJSONBody(r, &req, "request body must be an object with name and optional config"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	cfg := req.Config
	if strings.TrimSpace(req.Name) != "" {
		cfg.Name = req.Name
	}
	site, err := s.siteStore.Create(cfg)
	if err != nil {
		s.writeSiteStoreError(w, err)
		return
	}
	if err := initializeDefaultSiteOutput(site.Config); err != nil {
		_, _ = s.siteStore.Delete(site.ID)
		s.writeRenderError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, site)
}

func (s *Server) selectSite(w http.ResponseWriter, r *http.Request) {
	if s.siteStore == nil {
		WriteError(w, http.StatusBadRequest, "site_registry_unavailable", "multiple sites are only available when styxpress-admin runs without -config")
		return
	}
	site, err := s.siteStore.Select(r.PathValue("id"))
	if err != nil {
		s.writeSiteStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) deleteSite(w http.ResponseWriter, r *http.Request) {
	if s.siteStore == nil {
		WriteError(w, http.StatusBadRequest, "site_registry_unavailable", "multiple sites are only available when styxpress-admin runs without -config")
		return
	}
	activeID, err := s.siteStore.Delete(r.PathValue("id"))
	if err != nil {
		s.writeSiteStoreError(w, err)
		return
	}
	sites, _, err := s.siteStore.List()
	if err != nil {
		s.writeSiteStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sitesResponse{
		Sites:        sites,
		ActiveSiteID: activeID,
		MultiSite:    true,
	})
}

func (s *Server) getConfig(w http.ResponseWriter, _ *http.Request) {
	cfg, err := s.loadConfig()
	if err != nil {
		s.writeConfigPathError(w, err)
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
	current, err := s.loadConfig()
	if err == nil && deploySetupRequired(current, cfg) {
		WriteError(w, http.StatusConflict, "deploy_setup_required", "first SFTP deploy setup must be tested before saving")
		return
	} else if err != nil && !errors.Is(err, config.ErrNoActiveSite) {
		s.writeConfigPathError(w, err)
		return
	}

	saved, err := s.saveActiveConfig(cfg)
	if err != nil {
		if errors.Is(err, config.ErrInvalidConfig) || errors.Is(err, ErrInvalidLocalPath) {
			WriteError(w, http.StatusBadRequest, "invalid_config", err.Error())
			return
		}
		if errors.Is(err, config.ErrNoActiveSite) || errors.Is(err, config.ErrSiteNotFound) {
			s.writeConfigPathError(w, err)
			return
		}
		s.logger.Printf("save config: %v", err)
		WriteError(w, http.StatusInternalServerError, "config_save_failed", "failed to save config")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) getSiteConfig(w http.ResponseWriter, _ *http.Request) {
	contentDir, err := s.configuredContentDir()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	cfg, err := siteconfig.LoadOrDefault(contentDir)
	if err != nil {
		s.writeSiteConfigError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) saveSiteConfig(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var cfg siteconfig.Config
	if err := decodeJSONBody(r, &cfg, "request body must be a valid site config object"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	contentDir, err := s.configuredContentDir()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := siteconfig.Save(contentDir, cfg); err != nil {
		s.writeSiteConfigError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, siteconfig.WithDefaults(cfg))
}

func (s *Server) previewSiteConfig(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var cfg siteconfig.Config
	if err := decodeJSONBody(r, &cfg, "request body must be a valid site config object"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	renderer, err := s.renderer()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	html, err := renderer.RenderSitePreview(cfg)
	if err != nil {
		if errors.Is(err, siteconfig.ErrInvalidConfig) {
			s.writeSiteConfigError(w, err)
			return
		}
		s.writeRenderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, previewResponse{HTML: html})
}

func (s *Server) uploadSiteFavicon(w http.ResponseWriter, r *http.Request) {
	file, header, ok := readUpload(w, r, false)
	if !ok {
		return
	}
	defer file.Close()

	faviconName, err := faviconUploadName(header.Filename)
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	contentDir, err := s.configuredContentDir()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}

	faviconPath := path.Join("assets", faviconName)
	if err := writeUploadedFile(filepath.Join(contentDir, filepath.FromSlash(faviconPath)), file); err != nil {
		s.logger.Printf("site favicon upload: %v", err)
		WriteError(w, http.StatusInternalServerError, "site_favicon_failed", "failed to save site favicon")
		return
	}
	cfg, err := siteconfig.LoadOrDefault(contentDir)
	if err != nil {
		s.writeSiteConfigError(w, err)
		return
	}
	cfg.Favicon = faviconPath
	if err := siteconfig.Save(contentDir, cfg); err != nil {
		s.writeSiteConfigError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, siteconfig.WithDefaults(cfg))
}

func (s *Server) resetSiteFavicon(w http.ResponseWriter, _ *http.Request) {
	contentDir, err := s.configuredContentDir()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	cfg, err := siteconfig.LoadOrDefault(contentDir)
	if err != nil {
		s.writeSiteConfigError(w, err)
		return
	}
	cfg.Favicon = siteconfig.DefaultFaviconPath
	if err := siteconfig.Save(contentDir, cfg); err != nil {
		s.writeSiteConfigError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, siteconfig.WithDefaults(cfg))
}

type postPayload struct {
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Source        string   `json:"source"`
	Cover         string   `json:"cover"`
	Assets        []string `json:"assets"`
	PublishedAt   string   `json:"publishedAt"`
	UpdatedAt     string   `json:"updatedAt"`
	PublishStatus string   `json:"publishStatus"`
}

type postResponse struct {
	Slug          string   `json:"slug"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Source        string   `json:"source,omitempty"`
	Cover         string   `json:"cover,omitempty"`
	Assets        []string `json:"assets"`
	PublishedAt   string   `json:"publishedAt"`
	UpdatedAt     string   `json:"updatedAt"`
	PublishStatus string   `json:"publishStatus"`
}

type postListResponse struct {
	Posts []postResponse `json:"posts"`
}

type previewResponse struct {
	HTML string `json:"html"`
}

type renderPostResponse struct {
	Post rendering.Result     `json:"post"`
	Site rendering.SiteResult `json:"site"`
}

type renderSiteResponse struct {
	Posts []rendering.Result   `json:"posts"`
	Site  rendering.SiteResult `json:"site"`
}

type deployStatusResponse struct {
	Enabled    bool               `json:"enabled"`
	Configured bool               `json:"configured"`
	OutOfSync  bool               `json:"outOfSync"`
	SecretSet  bool               `json:"secretSet"`
	Summary    *deploypkg.Summary `json:"summary,omitempty"`
}

type deploySetupRequest struct {
	Config                 config.Config `json:"config"`
	Secret                 string        `json:"secret"`
	ConfirmRemoteOverwrite bool          `json:"confirmRemoteOverwrite"`
}

type deploySetupResponse struct {
	RequiresConfirmation bool               `json:"requiresConfirmation"`
	RemoteFiles          int                `json:"remoteFiles"`
	Config               config.Config      `json:"config,omitempty"`
	SecretSet            bool               `json:"secretSet"`
	Summary              *deploypkg.Summary `json:"summary,omitempty"`
}

type deploySecretRequest struct {
	Secret string `json:"secret"`
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
	post.PublishedAt = time.Time{}
	post.UpdatedAt = time.Time{}
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

func (s *Server) getCover(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	post, err := repo.LoadPost(slug)
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	if post.Cover == "" {
		WriteError(w, http.StatusNotFound, "cover_not_found", "cover not found")
		return
	}
	contentDir, err := s.configuredContentDir()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	contentType := mime.TypeByExtension(filepath.Ext(post.Cover))
	if err := serveContentMediaFile(w, r, contentType, contentDir, "posts", slug, post.Cover); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			WriteError(w, http.StatusNotFound, "cover_not_found", "cover not found")
			return
		}
		s.writeContentError(w, err)
		return
	}
}

func (s *Server) uploadCover(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	file, header, ok := readUpload(w, r, false)
	if !ok {
		return
	}
	defer file.Close()

	coverName, err := coverUploadName(header.Filename)
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := repo.WriteCover(slug, coverName, file); err != nil {
		s.writeContentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"cover": coverName})
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

func (s *Server) getAsset(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	assetPath, err := content.CleanAssetPath(r.PathValue("assetPath"))
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	post, err := repo.LoadPost(slug)
	if err != nil {
		s.writeContentError(w, err)
		return
	}
	if !postHasAsset(post, assetPath) {
		WriteError(w, http.StatusNotFound, "asset_not_found", "asset not found")
		return
	}

	contentDir, err := s.configuredContentDir()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	contentType := mime.TypeByExtension(filepath.Ext(assetPath))
	components := append([]string{"posts", slug, "assets"}, strings.Split(assetPath, "/")...)
	if err := serveContentMediaFile(w, r, contentType, contentDir, components...); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			WriteError(w, http.StatusNotFound, "asset_not_found", "asset not found")
			return
		}
		s.writeContentError(w, err)
		return
	}
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
	if !isSupportedImageFile(cleaned) {
		s.writeContentError(w, fmt.Errorf("%w: asset must be an image", content.ErrInvalidAsset))
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

func (s *Server) renderSite(w http.ResponseWriter, r *http.Request) {
	result, err := s.renderAll()
	if err != nil {
		s.writeRenderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, renderSiteResponse{Posts: result.Posts, Site: result.Site})
}

func (s *Server) publishPost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	slug := r.PathValue("slug")
	if slug == "" {
		WriteError(w, http.StatusBadRequest, "invalid_publish", "slug is required")
		return
	}

	repo, err := s.repository()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	published, err := repo.MarkPostPublished(slug, content.TimestampOptions{})
	if err != nil {
		s.writeContentError(w, err)
		return
	}

	postResult, siteResult, err := s.renderPostAndSite(published.Slug)
	if err != nil {
		s.writeRenderError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, renderPostResponse{Post: postResult, Site: siteResult})
}

func (s *Server) setupDeploy(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	site, err := s.activeSite()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	var req deploySetupRequest
	if err := decodeJSONBody(r, &req, "request body must contain deploy setup config"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	cfg := config.WithDefaults(req.Config)
	if !cfg.Deploy.Enabled {
		WriteError(w, http.StatusBadRequest, "deploy_disabled", "SFTP deploy is disabled")
		return
	}
	if !deployConfigured(cfg) {
		WriteError(w, http.StatusBadRequest, "invalid_deploy_config", "SFTP host, user, and remote folder are required")
		return
	}
	paths, err := configuredSitePaths(cfg)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	deployConfig, err := s.deployConfigFor(site.ID, cfg, req.Secret)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if req.Secret != "" {
		if err := deploypkg.VerifySecret(r.Context(), deployConfig); err != nil {
			s.writeDeployError(w, err)
			return
		}
	}
	inspection, err := deploypkg.Inspect(r.Context(), deployConfig)
	if err != nil {
		s.writeDeployError(w, err)
		return
	}
	if deploySetupRequired(site.Config, cfg) && inspection.RemoteFiles > 0 && !req.ConfirmRemoteOverwrite {
		writeJSON(w, http.StatusOK, deploySetupResponse{
			RequiresConfirmation: true,
			RemoteFiles:          inspection.RemoteFiles,
			SecretSet:            s.deploySecretSet(),
		})
		return
	}
	if err := renderSiteOutput(cfg); err != nil {
		s.writeRenderError(w, err)
		return
	}
	saved, err := s.saveActiveConfig(cfg)
	if err != nil {
		if errors.Is(err, config.ErrInvalidConfig) || errors.Is(err, ErrInvalidLocalPath) {
			WriteError(w, http.StatusBadRequest, "invalid_config", err.Error())
			return
		}
		s.writeConfigPathError(w, err)
		return
	}
	if req.Secret != "" {
		s.setDeploySecret(req.Secret)
	}
	summary, err := deploypkg.Sync(r.Context(), paths.publicDir, deployConfig)
	if err != nil {
		s.writeDeployError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deploySetupResponse{
		RemoteFiles: inspection.RemoteFiles,
		Config:      saved,
		SecretSet:   s.deploySecretSet(),
		Summary:     &summary,
	})
}

func (s *Server) deployStatus(w http.ResponseWriter, r *http.Request) {
	site, err := s.activeSite()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	cfg := site.Config
	response := deployStatusResponse{
		Enabled:   cfg.Deploy.Enabled,
		SecretSet: s.deploySecretSet(),
	}
	if !cfg.Deploy.Enabled {
		writeJSON(w, http.StatusOK, response)
		return
	}
	response.Configured = deployConfigured(cfg)
	if !response.Configured {
		writeJSON(w, http.StatusOK, response)
		return
	}
	paths, err := configuredSitePaths(cfg)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	deployConfig, err := s.deployConfig(site)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	summary, err := deploypkg.Status(r.Context(), paths.publicDir, deployConfig)
	if err != nil {
		s.writeDeployError(w, err)
		return
	}
	response.OutOfSync = summary.OutOfSync
	response.Summary = &summary
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) saveDeploySecret(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	site, err := s.activeSite()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	var req deploySecretRequest
	if err := decodeJSONBody(r, &req, "request body must contain a deploy secret"); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if req.Secret == "" {
		WriteError(w, http.StatusBadRequest, "invalid_deploy_secret", "password or passphrase is required")
		return
	}
	cfg := site.Config
	if !cfg.Deploy.Enabled {
		WriteError(w, http.StatusBadRequest, "deploy_disabled", "SFTP deploy is disabled")
		return
	}
	if !deployConfigured(cfg) {
		WriteError(w, http.StatusBadRequest, "invalid_deploy_config", "SFTP host, user, and remote folder are required")
		return
	}
	deployConfig, err := s.deployConfigFor(site.ID, cfg, req.Secret)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	if err := deploypkg.VerifySecret(r.Context(), deployConfig); err != nil {
		s.writeDeployError(w, err)
		return
	}
	s.setDeploySecret(req.Secret)
	writeJSON(w, http.StatusOK, map[string]bool{"secretSet": true})
}

func (s *Server) deleteDeploySecret(w http.ResponseWriter, _ *http.Request) {
	s.clearDeploySecret()
	writeJSON(w, http.StatusOK, map[string]bool{"secretSet": false})
}

func (s *Server) deployNow(w http.ResponseWriter, r *http.Request) {
	site, err := s.activeSite()
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	cfg := site.Config
	if !cfg.Deploy.Enabled {
		WriteError(w, http.StatusBadRequest, "deploy_disabled", "SFTP deploy is disabled")
		return
	}
	paths, err := configuredSitePaths(cfg)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	deployConfig, err := s.deployConfig(site)
	if err != nil {
		s.writeConfigPathError(w, err)
		return
	}
	summary, err := deploypkg.Sync(r.Context(), paths.publicDir, deployConfig)
	if err != nil {
		s.writeDeployError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) renderPostAndSite(slug string) (rendering.Result, rendering.SiteResult, error) {
	repo, err := s.repository()
	if err != nil {
		return rendering.Result{}, rendering.SiteResult{}, err
	}
	post, err := repo.LoadPost(slug)
	if err != nil {
		return rendering.Result{}, rendering.SiteResult{}, err
	}
	if !post.IsPublished() {
		return rendering.Result{}, rendering.SiteResult{}, fmt.Errorf("%w: draft post cannot be rendered to public output", rendering.ErrUnpublishedPost)
	}

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

func (s *Server) renderAll() (rendering.AllResult, error) {
	renderer, err := s.renderer()
	if err != nil {
		return rendering.AllResult{}, err
	}
	return renderer.RenderAll()
}

func initializeDefaultSiteOutput(cfg config.Config) error {
	cfg = config.WithDefaults(cfg)
	paths, err := configuredSitePaths(cfg)
	if err != nil {
		return err
	}
	if _, err := os.Stat(siteconfig.Path(paths.contentDir)); errors.Is(err, os.ErrNotExist) {
		defaults := siteconfig.Default()
		if name := strings.TrimSpace(cfg.Name); name != "" {
			defaults.Title = name
		}
		if err := siteconfig.Save(paths.contentDir, defaults); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return renderSiteOutput(cfg)
}

func renderSiteOutput(cfg config.Config) error {
	cfg = config.WithDefaults(cfg)
	paths, err := configuredSitePaths(cfg)
	if err != nil {
		return err
	}
	renderer, err := rendering.New(paths.contentDir, paths.publicDir)
	if err != nil {
		return err
	}
	_, err = renderer.RenderAll()
	return err
}

func (s *Server) repository() (*content.Repository, error) {
	contentDir, err := s.configuredContentDir()
	if err != nil {
		return nil, err
	}
	return content.NewRepository(contentDir), nil
}

func (s *Server) configuredContentDir() (string, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return "", err
	}
	paths, err := configuredSitePaths(cfg)
	if err != nil {
		return "", err
	}
	return paths.contentDir, nil
}

func (s *Server) renderer() (*rendering.Renderer, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}
	paths, err := configuredSitePaths(cfg)
	if err != nil {
		return nil, err
	}
	return rendering.New(paths.contentDir, paths.publicDir)
}

func (s *Server) deployConfig(site config.Site) (deploypkg.Config, error) {
	return s.deployConfigFor(site.ID, site.Config, s.deploySecretValue())
}

func (s *Server) deployConfigFor(siteID string, cfg config.Config, secret string) (deploypkg.Config, error) {
	statePath, err := s.deployStatePath(siteID)
	if err != nil {
		return deploypkg.Config{}, err
	}
	return deploypkg.Config{
		Host:           cfg.Deploy.SFTP.Host,
		Port:           cfg.Deploy.SFTP.Port,
		User:           cfg.Deploy.SFTP.User,
		RemotePath:     cfg.Deploy.SFTP.RemotePath,
		KeyPath:        cfg.Deploy.SFTP.KeyPath,
		KnownHostsPath: cfg.Deploy.SFTP.KnownHostsPath,
		Secret:         secret,
		StatePath:      statePath,
	}, nil
}

func (s *Server) deployStatePath(siteID string) (string, error) {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		siteID = "single"
	}
	var root string
	if s.siteStore != nil {
		root = s.siteStore.Root()
	} else if strings.TrimSpace(s.configPath) != "" {
		root = filepath.Dir(s.configPath)
	} else {
		var err error
		root, err = config.DefaultDir()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(root, "deploy-state", siteID+".json"), nil
}

func (s *Server) setDeploySecret(secret string) {
	s.deploySecretMu.Lock()
	defer s.deploySecretMu.Unlock()
	s.deploySecret = secret
}

func (s *Server) clearDeploySecret() {
	s.deploySecretMu.Lock()
	defer s.deploySecretMu.Unlock()
	s.deploySecret = ""
}

func (s *Server) deploySecretValue() string {
	s.deploySecretMu.RLock()
	defer s.deploySecretMu.RUnlock()
	return s.deploySecret
}

func (s *Server) deploySecretSet() bool {
	s.deploySecretMu.RLock()
	defer s.deploySecretMu.RUnlock()
	return s.deploySecret != ""
}

func deployConfigured(cfg config.Config) bool {
	return strings.TrimSpace(cfg.Deploy.SFTP.Host) != "" &&
		strings.TrimSpace(cfg.Deploy.SFTP.User) != "" &&
		strings.TrimSpace(cfg.Deploy.SFTP.RemotePath) != ""
}

func deploySetupRequired(current config.Config, next config.Config) bool {
	next = config.WithDefaults(next)
	return next.Deploy.Enabled && deployConfigured(next) && !deployConfigured(current)
}

func (s *Server) loadConfig() (config.Config, error) {
	if s.siteStore != nil {
		site, err := s.siteStore.Active()
		if err != nil {
			return config.Config{}, err
		}
		return site.Config, nil
	}
	return config.LoadOrDefault(s.configPath)
}

func (s *Server) activeSite() (config.Site, error) {
	if s.siteStore != nil {
		return s.siteStore.Active()
	}
	cfg, err := config.LoadOrDefault(s.configPath)
	if err != nil {
		return config.Site{}, err
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = "Configured site"
	}
	cfg.Name = name
	return config.Site{ID: "single", Name: name, Config: cfg}, nil
}

func (s *Server) saveActiveConfig(cfg config.Config) (config.Config, error) {
	if s.siteStore != nil {
		cfg = config.WithDefaults(cfg)
		if _, err := configuredSitePaths(cfg); err != nil {
			return config.Config{}, err
		}
		site, err := s.siteStore.SaveActive(cfg)
		if err != nil {
			return config.Config{}, err
		}
		return site.Config, nil
	}
	cfg = config.WithDefaults(cfg)
	if _, err := configuredSitePaths(cfg); err != nil {
		return config.Config{}, err
	}
	if err := config.Save(s.configPath, cfg); err != nil {
		return config.Config{}, err
	}
	return config.WithDefaults(cfg), nil
}

func (s *Server) sites() ([]config.Site, string, error) {
	if s.siteStore != nil {
		return s.siteStore.List()
	}
	cfg, err := s.loadConfig()
	if err != nil {
		return nil, "", err
	}
	name := strings.TrimSpace(cfg.Name)
	if name == "" {
		name = "Configured site"
	}
	cfg.Name = name
	site := config.Site{ID: "single", Name: name, Config: cfg}
	return []config.Site{site}, site.ID, nil
}

func (s *Server) writeConfigPathError(w http.ResponseWriter, err error) {
	if errors.Is(err, config.ErrInvalidConfig) || errors.Is(err, config.ErrInvalidSiteID) || errors.Is(err, ErrInvalidLocalPath) || errors.Is(err, rendering.ErrInvalidRenderConfig) {
		WriteError(w, http.StatusBadRequest, "invalid_config", err.Error())
		return
	}
	if errors.Is(err, config.ErrNoActiveSite) {
		WriteError(w, http.StatusBadRequest, "site_required", "create or select a site first")
		return
	}
	if errors.Is(err, config.ErrSiteNotFound) {
		WriteError(w, http.StatusNotFound, "site_not_found", err.Error())
		return
	}
	s.logger.Printf("config path error: %v", err)
	WriteError(w, http.StatusInternalServerError, "config_load_failed", "failed to load config")
}

func (s *Server) writeSiteStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, config.ErrInvalidConfig), errors.Is(err, config.ErrInvalidSiteID):
		WriteError(w, http.StatusBadRequest, "invalid_site", err.Error())
	case errors.Is(err, config.ErrNoActiveSite):
		WriteError(w, http.StatusBadRequest, "site_required", "create or select a site first")
	case errors.Is(err, config.ErrSiteNotFound):
		WriteError(w, http.StatusNotFound, "site_not_found", err.Error())
	default:
		s.logger.Printf("site store error: %v", err)
		WriteError(w, http.StatusInternalServerError, "site_store_failed", "failed to access saved sites")
	}
}

func (s *Server) writeSiteConfigError(w http.ResponseWriter, err error) {
	if errors.Is(err, siteconfig.ErrInvalidConfig) {
		WriteError(w, http.StatusBadRequest, "invalid_site_config", err.Error())
		return
	}
	s.logger.Printf("site config error: %v", err)
	WriteError(w, http.StatusInternalServerError, "site_config_failed", "failed to access site config")
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
	case errors.Is(err, siteconfig.ErrInvalidConfig):
		WriteError(w, http.StatusBadRequest, "invalid_site_config", err.Error())
	case errors.Is(err, content.ErrPostNotFound):
		WriteError(w, http.StatusNotFound, "post_not_found", "post not found")
	case errors.Is(err, rendering.ErrInvalidRenderConfig), errors.Is(err, rendering.ErrUnsafeAsset), errors.Is(err, rendering.ErrUnpublishedPost), errors.Is(err, content.ErrInvalidSlug), errors.Is(err, content.ErrInvalidPost), errors.Is(err, content.ErrInvalidAssetPath), errors.Is(err, content.ErrInvalidAsset):
		WriteError(w, http.StatusBadRequest, "render_failed", err.Error())
	default:
		s.logger.Printf("render error: %v", err)
		WriteError(w, http.StatusInternalServerError, "render_failed", "failed to render content")
	}
}

func (s *Server) writeDeployError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidLocalPath), errors.Is(err, config.ErrInvalidConfig), errors.Is(err, deploypkg.ErrInvalidConfig):
		WriteError(w, http.StatusBadRequest, "invalid_deploy_config", err.Error())
	default:
		s.logger.Printf("deploy error: %v", err)
		WriteError(w, http.StatusBadGateway, "deploy_failed", err.Error())
	}
}

var ErrInvalidLocalPath = errors.New("invalid local path")

func configuredPath(value string) (string, error) {
	path, err := localpath.CleanRequired(value)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidLocalPath, err)
	}
	return path, nil
}

func configuredSitePaths(cfg config.Config) (sitePaths, error) {
	contentDir, err := configuredPath(cfg.ContentDir)
	if err != nil {
		return sitePaths{}, err
	}
	publicDir, err := configuredPath(cfg.PublicDir)
	if err != nil {
		return sitePaths{}, err
	}
	if err := localpath.EnsureSeparateRoots("contentDir", contentDir, "publicDir", publicDir); err != nil {
		return sitePaths{}, fmt.Errorf("%w: %v", ErrInvalidLocalPath, err)
	}
	return sitePaths{contentDir: contentDir, publicDir: publicDir}, nil
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

func coverUploadName(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filepath.Base(filename)))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".avif":
		return "cover" + ext, nil
	default:
		return "", content.ErrUnsupportedCover
	}
}

func faviconUploadName(filename string) (string, error) {
	cleaned, err := content.CleanAssetPath(filepath.Base(strings.TrimSpace(filename)))
	if err != nil || strings.Contains(cleaned, "/") {
		return "", content.ErrInvalidAsset
	}
	if !strings.EqualFold(filepath.Ext(cleaned), ".ico") {
		return "", fmt.Errorf("%w: favicon must be an .ico file", content.ErrInvalidAsset)
	}
	return cleaned, nil
}

func isSupportedImageFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filepath.Base(filename))) {
	case ".jpg", ".jpeg", ".png", ".webp", ".avif", ".gif":
		return true
	default:
		return false
	}
}

func postHasAsset(post content.Post, assetPath string) bool {
	for _, asset := range post.Assets {
		if asset == assetPath {
			return true
		}
	}
	return false
}

func serveContentMediaFile(w http.ResponseWriter, r *http.Request, contentType string, root string, components ...string) error {
	file, info, err := openContentMediaFile(root, components...)
	if err != nil {
		return err
	}
	defer file.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return nil
}

func openContentMediaFile(root string, components ...string) (*os.File, os.FileInfo, error) {
	if len(components) == 0 {
		return nil, nil, content.ErrInvalidAsset
	}

	cleanRoot, err := localpath.CleanRequired(root)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidLocalPath, err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		return nil, nil, err
	}

	current := resolvedRoot
	for index, component := range components {
		if unsafeMediaComponent(component) {
			return nil, nil, fmt.Errorf("%w: unsafe path component", content.ErrInvalidAsset)
		}
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, nil, fmt.Errorf("%w: symlink %s", content.ErrInvalidAsset, filepath.ToSlash(current))
		}
		if index < len(components)-1 {
			if !info.IsDir() {
				return nil, nil, fmt.Errorf("%w: path component is not a directory", content.ErrInvalidAsset)
			}
			continue
		}
		if info.IsDir() {
			return nil, nil, content.ErrInvalidAsset
		}
	}

	if !pathWithinRoot(resolvedRoot, current) {
		return nil, nil, fmt.Errorf("%w: path escapes content root", content.ErrInvalidAsset)
	}
	file, err := os.Open(current)
	if err != nil {
		return nil, nil, err
	}

	info, statErr := file.Stat()
	lstatInfo, lstatErr := os.Lstat(current)
	resolvedFile, resolveErr := filepath.EvalSymlinks(current)
	switch {
	case statErr != nil:
		file.Close()
		return nil, nil, statErr
	case lstatErr != nil:
		file.Close()
		return nil, nil, lstatErr
	case resolveErr != nil:
		file.Close()
		return nil, nil, resolveErr
	case info.IsDir(), lstatInfo.Mode()&os.ModeSymlink != 0, !os.SameFile(info, lstatInfo), !pathWithinRoot(resolvedRoot, resolvedFile):
		file.Close()
		return nil, nil, content.ErrInvalidAsset
	default:
		return file, info, nil
	}
}

func unsafeMediaComponent(component string) bool {
	return component == "" || component == "." || component == ".." || strings.Contains(component, "/") || strings.Contains(component, "\\")
}

func pathWithinRoot(root string, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	if root == candidate {
		return true
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func writeUploadedFile(destination string, reader io.Reader) error {
	dir := filepath.Dir(destination)
	if err := os.MkdirAll(dir, siteAssetFolderMode); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "."+filepath.Base(destination)+".*")
	if err != nil {
		return err
	}
	temp := file.Name()
	_, copyErr := io.Copy(file, reader)
	chmodErr := file.Chmod(siteAssetFileMode)
	closeErr := file.Close()
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
	if err := os.Rename(temp, destination); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
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
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("time values must be RFC3339")
	}
	return parsed, nil
}

func newPostResponse(post content.Post, includeSource bool) postResponse {
	response := postResponse{
		Slug:          post.Slug,
		Title:         post.Title,
		Description:   post.Description,
		Cover:         post.Cover,
		Assets:        append([]string(nil), post.Assets...),
		PublishedAt:   formatPostTime(post.PublishedAt),
		UpdatedAt:     formatPostTime(post.UpdatedAt),
		PublishStatus: string(post.PublishStatus()),
	}
	if includeSource {
		response.Source = post.Source
	}
	return response
}

func formatPostTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
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
