package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/nordine-abde/styxpress/internal/config"
	"github.com/nordine-abde/styxpress/internal/publishing"
)

const SessionHeader = "X-Styxpress-Session"

type Server struct {
	configPath string
	token      string
	logger     *log.Logger
	sshTester  func(*http.Request, config.Config, string) error
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
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be a valid config object")
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
