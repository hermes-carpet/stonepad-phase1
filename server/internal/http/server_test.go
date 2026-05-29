package http

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hermes-carpet/stonepad/server/internal/auth"
	"github.com/hermes-carpet/stonepad/server/internal/config"
	"github.com/hermes-carpet/stonepad/server/internal/storage"
	"github.com/hermes-carpet/stonepad/server/internal/sync"
)

// setupTestServer creates a test server with in-memory storage.
func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	dir := t.TempDir()
	cfg := &config.Config{
		ListenAddr:            ":0",
		DataDir:               dir,
		StorageMode:           "direct",
		AuthMode:              "none",
		WorkspaceID:           "default",
		UserID:                "owner",
		MaxNoteSizeBytes:      5 * 1024 * 1024,
		MaxNotesPerWorkspace:  100000,
		NativeAPIEnabled:      true,
		S3EndpointEnabled:     false,
	}

	store, err := storage.NewFilesystemStorage(dir, cfg.WorkspaceID)
	if err != nil {
		t.Fatalf("NewFilesystemStorage: %v", err)
	}

	metaStore, err := sync.OpenMetadataStore(dir + "/meta.db")
	if err != nil {
		t.Fatalf("OpenMetadataStore: %v", err)
	}
	t.Cleanup(func() { metaStore.Close() })

	metaStore.InitWorkspace(cfg.WorkspaceID)
	metaStore.InitUser(cfg.UserID)

	authenticator := auth.NewNoneAuth(cfg.UserID)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewServer(cfg, store, metaStore, authenticator, logger), dir
}

func TestHealth(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"ok"`) {
		t.Errorf("expected 'ok' in body, got %q", w.Body.String())
	}
}

func TestManifestEmpty(t *testing.T) {
	srv, _ := setupTestServer(t)
	req := httptest.NewRequest("GET", "/api/v1/manifest", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"notes":[]`) {
		t.Errorf("expected empty notes array, got %q", w.Body.String())
	}
}

func TestPutAndGetNote(t *testing.T) {
	srv, _ := setupTestServer(t)

	// PUT
	req := httptest.NewRequest("PUT", "/api/v1/notes/hello.md",
		strings.NewReader("# Hello\nWorld"))
	req.Header.Set("Content-Type", "text/markdown")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PUT failed: %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"content_hash"`) {
		t.Errorf("PUT response missing content_hash: %s", w.Body.String())
	}

	// GET
	req = httptest.NewRequest("GET", "/api/v1/notes/hello.md", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET failed: %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != "# Hello\nWorld" {
		t.Errorf("GET body mismatch: got %q", w.Body.String())
	}

	// Manifest should include the note
	req = httptest.NewRequest("GET", "/api/v1/manifest", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), `"hello.md"`) {
		t.Errorf("manifest missing hello.md: %s", w.Body.String())
	}
}

func TestDeleteNote(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Put then delete
	req := httptest.NewRequest("PUT", "/api/v1/notes/temp.md",
		strings.NewReader("temp"))
	req.Header.Set("Content-Type", "text/markdown")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	req = httptest.NewRequest("DELETE", "/api/v1/notes/temp.md", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("DELETE: expected 204, got %d", w.Code)
	}
}

func TestNote404(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/notes/nonexistent.md", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestNoteRequiresAuth(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir:      dir,
		StorageMode:  "direct",
		AuthMode:     "token",
		AuthToken:    "secret123",
		WorkspaceID:  "default",
		UserID:       "owner",
		MaxNoteSizeBytes: 5 * 1024 * 1024,
	}

	store, _ := storage.NewFilesystemStorage(dir, cfg.WorkspaceID)
	metaStore, _ := sync.OpenMetadataStore(dir + "/meta.db")
	defer metaStore.Close()
	metaStore.InitWorkspace(cfg.WorkspaceID)
	metaStore.InitUser(cfg.UserID)

	authenticator := auth.NewTokenAuth(cfg.AuthToken)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(cfg, store, metaStore, authenticator, logger)

	// Request without auth should be rejected
	req := httptest.NewRequest("GET", "/api/v1/manifest", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}

	// Request with wrong token should be rejected
	req = httptest.NewRequest("GET", "/api/v1/manifest", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", w.Code)
	}

	// Request with correct token should succeed
	req = httptest.NewRequest("GET", "/api/v1/manifest", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with correct token, got %d: %s", w.Code, w.Body.String())
	}
}

// Ensure context propagation works
func TestContextPropagation(t *testing.T) {
	srv, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/manifest", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
