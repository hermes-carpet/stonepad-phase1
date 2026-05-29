// Package http provides the HTTP server, routing, and handlers
// for the Stonepad server. Uses Go 1.22+ stdlib net/http with method
// matching on http.ServeMux.
// See §7.4 and §7.5 for the API specifications.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hermes-carpet/stonepad/server/internal/auth"
	"github.com/hermes-carpet/stonepad/server/internal/config"
	"github.com/hermes-carpet/stonepad/server/internal/s3"
	"github.com/hermes-carpet/stonepad/server/internal/storage"
	"github.com/hermes-carpet/stonepad/server/internal/sync"
)

// Server wraps the HTTP server and associated dependencies.
type Server struct {
	cfg       *config.Config
	store     storage.Storage
	metaStore *sync.MetadataStore
	auth      auth.Authenticator
	s3Handler *S3Handler
	mux       *http.ServeMux
	logger    *slog.Logger
}

// NewServer creates a new Server with all dependencies wired in.
func NewServer(
	cfg *config.Config,
	store storage.Storage,
	metaStore *sync.MetadataStore,
	authenticator auth.Authenticator,
	s3Creds []s3.Credential,
	logger *slog.Logger,
) *Server {
	s := &Server{
		cfg:       cfg,
		store:     store,
		metaStore: metaStore,
		auth:      authenticator,
		s3Handler: NewS3Handler(store, metaStore, s3Creds, cfg.WorkspaceID, cfg.MaxNoteSizeBytes),
		mux:       http.NewServeMux(),
		logger:    logger,
	}
	s.registerRoutes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// Handler returns the http.Handler for use with http.ListenAndServe.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	// Health check — always accessible, no auth
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	// Native API endpoints — auth required
	s.mux.HandleFunc("GET /api/v1/manifest", s.wrapAuth(s.handleManifest))
	s.mux.HandleFunc("GET /api/v1/notes/{path...}", s.wrapAuth(s.handleGetNote))
	s.mux.HandleFunc("PUT /api/v1/notes/{path...}", s.wrapAuth(s.handlePutNote))
	s.mux.HandleFunc("DELETE /api/v1/notes/{path...}", s.wrapAuth(s.handleDeleteNote))

	// Auth endpoints (users mode only)
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.wrapAuth(s.handleLogout))

	// S3-compatible endpoints — authenticated via Sig V4
	if s.cfg.S3EndpointEnabled {
		s.mux.HandleFunc("GET /s3/", s.wrapS3Auth(s.s3Handler.HandleListBuckets))
		s.mux.HandleFunc("GET /s3/{bucket}", s.wrapS3Auth(s.s3Handler.HandleListObjects))
		s.mux.HandleFunc("HEAD /s3/{bucket}/{key...}", s.wrapS3Auth(s.s3Handler.HandleHeadObject))
		s.mux.HandleFunc("GET /s3/{bucket}/{key...}", s.wrapS3Auth(s.s3Handler.HandleGetObject))
		s.mux.HandleFunc("PUT /s3/{bucket}/{key...}", s.wrapS3Auth(s.s3Handler.HandlePutObject))
		s.mux.HandleFunc("DELETE /s3/{bucket}/{key...}", s.wrapS3Auth(s.s3Handler.HandleDeleteObject))
	}
}

// wrapAuth wraps a handler with authentication middleware.
func (s *Server) wrapAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := s.auth.Authenticate(r)
		if err != nil {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
		// Store userID in context for handlers
		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// wrapS3Auth wraps an S3 handler with Sig V4 authentication.
func (s *Server) wrapS3Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := s.s3Handler.authS3Request(r)
		if err != nil {
			writeS3Error(w, r, "AccessDenied", err.Error(), r.URL.Path)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// Health check handler. Always returns 200 OK.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Manifest handler. Returns the full manifest for the workspace.
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	manifest, err := sync.BuildManifest(s.cfg.WorkspaceID, s.store)
	if err != nil {
		s.logger.Error("building manifest", "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to build manifest")
		return
	}
	writeJSON(w, http.StatusOK, manifest)
}

// Get note handler. Returns the raw markdown content of a note.
func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, r, http.StatusBadRequest, "bad_request", "path is required")
		return
	}

	reader, err := s.store.Get(r.Context(), path)
	if err == storage.ErrNotFound {
		writeError(w, r, http.StatusNotFound, "not_found", fmt.Sprintf("Note not found at path '%s'", path))
		return
	}
	if err != nil {
		s.logger.Error("getting note", "path", path, "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to read note")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	if _, err := io.Copy(w, reader); err != nil {
		s.logger.Error("writing response", "path", path, "error", err)
	}
}

// Put note handler. Creates or updates a note.
func (s *Server) handlePutNote(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, r, http.StatusBadRequest, "bad_request", "path is required")
		return
	}

	// Ensure .md extension for native API
	if !strings.HasSuffix(path, ".md") {
		writeError(w, r, http.StatusBadRequest, "bad_request", "file extension must be .md")
		return
	}

	// Enforce max note size
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxNoteSizeBytes)

	// Check workspace note count limit
	count, err := s.metaStore.CountNotes(s.cfg.WorkspaceID)
	if err == nil && count >= s.cfg.MaxNotesPerWorkspace {
		writeError(w, r, http.StatusForbidden, "forbidden", "workspace note limit reached")
		return
	}

	// Check If-Match precondition (optimistic concurrency)
	ifMatch := r.Header.Get("If-Match")
	if ifMatch != "" {
		currentHash, err := s.metaStore.GetNoteHash(s.cfg.WorkspaceID, path)
		if err != nil {
			s.logger.Error("checking hash for If-Match", "path", path, "error", err)
			writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to check precondition")
			return
		}
		if currentHash != "" && currentHash != ifMatch {
			writeJSON(w, http.StatusPreconditionFailed, sync.ConflictInfo{
				CurrentHash: currentHash,
				Path:        path,
			})
			return
		}
	}

	contentHash, err := s.store.Put(r.Context(), path, r.Body)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			writeError(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "note exceeds maximum size")
			return
		}
		if strings.Contains(err.Error(), "invalid path") {
			writeError(w, r, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		s.logger.Error("storing note", "path", path, "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to store note")
		return
	}

	// Get size
	meta, err := s.store.Head(r.Context(), path)
	sizeBytes := int64(0)
	if err == nil {
		sizeBytes = meta.SizeBytes
	}

	// Update SQLite metadata
	if err := s.metaStore.UpsertNote(s.cfg.WorkspaceID, path, contentHash, sizeBytes); err != nil {
		s.logger.Error("upserting metadata", "path", path, "error", err)
	}

	// Record audit log
	userID, _ := r.Context().Value(ctxKeyUserID).(string)
	action := "update"
	if ifMatch == "" {
		action = "create"
	}
	if err := s.metaStore.RecordAudit(s.cfg.WorkspaceID, userID, action, path, contentHash); err != nil {
		s.logger.Error("recording audit", "path", path, "error", err)
	}

	modifiedAt := time.Now().UTC().Format(time.RFC3339)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":         path,
		"content_hash": contentHash,
		"size_bytes":   sizeBytes,
		"modified_at":  modifiedAt,
	})
}

// Delete note handler. Removes a note.
func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, r, http.StatusBadRequest, "bad_request", "path is required")
		return
	}

	if err := s.store.Delete(r.Context(), path); err != nil {
		s.logger.Error("deleting note", "path", path, "error", err)
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to delete note")
		return
	}

	// Update SQLite metadata
	if err := s.metaStore.DeleteNote(s.cfg.WorkspaceID, path); err != nil {
		s.logger.Error("deleting metadata", "path", path, "error", err)
	}

	// Record audit log
	userID, _ := r.Context().Value(ctxKeyUserID).(string)
	if err := s.metaStore.RecordAudit(s.cfg.WorkspaceID, userID, "delete", path, ""); err != nil {
		s.logger.Error("recording audit", "path", path, "error", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// Login handler (users auth mode only).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	usersAuth, ok := s.auth.(*auth.UsersAuth)
	if !ok {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "users auth not configured")
		return
	}

	token, err := usersAuth.ValidateLogin(req.Username, req.Password)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, "unauthorized", "invalid username or password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

// Logout handler (users auth mode only).
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeError(w, r, http.StatusBadRequest, "bad_request", "missing Authorization header")
		return
	}
	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		writeError(w, r, http.StatusBadRequest, "bad_request", "invalid Authorization header format")
		return
	}

	usersAuth, ok := s.auth.(*auth.UsersAuth)
	if !ok {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "users auth not configured")
		return
	}

	if err := usersAuth.InvalidateToken(token); err != nil {
		s.logger.Error("invalidating token", "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// Context keys
type contextKey string

const ctxKeyUserID contextKey = "userID"

// --- Response helpers ---

// errorResponse is the standard error format per §7.4.
type errorResponse struct {
	Error errorBody `json:"error"`
}
type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeError writes a JSON error response with consistent format.
func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// GenerateRequestID creates a unique request ID for logging.
func GenerateRequestID() string {
	return uuid.New().String()
}
