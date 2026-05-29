// Stonepad Server — Self-hostable markdown notes sync server.
// Entry point. Reads environment variables, wires dependencies, starts HTTP server.
// See §7 of the Stonepad v1 Implementation Plan.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hermes-carpet/stonepad/server/internal/auth"
	"github.com/hermes-carpet/stonepad/server/internal/config"
	httpsrv "github.com/hermes-carpet/stonepad/server/internal/http"
	"github.com/hermes-carpet/stonepad/server/internal/s3"
	"github.com/hermes-carpet/stonepad/server/internal/storage"
	"github.com/hermes-carpet/stonepad/server/internal/sync"
	"github.com/hermes-carpet/stonepad/server/internal/tmpfs"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	// Load configuration from environment variables
	cfg := config.Load()

	// Configure structured logging
	logLevel := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	logger.Info("stonepad-server starting", "version", version, "listen_addr", cfg.ListenAddr)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Initialize storage backend
	var store storage.Storage
	var err error

	switch cfg.StorageMode {
	case "direct":
		store, err = storage.NewFilesystemStorage(cfg.DataDir, cfg.WorkspaceID)
		if err != nil {
			logger.Error("initializing storage", "error", err)
			os.Exit(1)
		}
		logger.Info("storage initialized", "mode", "direct", "base_path", store.BasePath())
	case "tmpfs":
		store, err = storage.NewFilesystemStorage(cfg.DataDir, cfg.WorkspaceID)
		if err != nil {
			logger.Error("initializing tmpfs storage", "error", err)
			os.Exit(1)
		}
		logger.Info("storage initialized", "mode", "tmpfs", "base_path", store.BasePath())

		// Ensure persist directory exists
		if err := os.MkdirAll(cfg.TmpfsPersistDir, 0755); err != nil {
			logger.Error("creating tmpfs persist directory", "error", err)
			os.Exit(1)
		}
	default:
		logger.Error("unsupported storage mode", "mode", cfg.StorageMode)
		os.Exit(1)
	}

	// Initialize SQLite metadata store
	dbPath := fmt.Sprintf("%s/meta.db", cfg.DataDir)
	metaStore, err := sync.OpenMetadataStore(dbPath)
	if err != nil {
		logger.Error("opening metadata store", "error", err)
		os.Exit(1)
	}
	defer metaStore.Close()

	// Initialize workspace and default user
	if err := metaStore.InitWorkspace(cfg.WorkspaceID); err != nil {
		logger.Error("initializing workspace", "error", err)
		os.Exit(1)
	}
	if err := metaStore.InitUser(cfg.UserID); err != nil {
		logger.Error("initializing user", "error", err)
		os.Exit(1)
	}

	// tmpfs snapshot management
	var snap *tmpfs.Snapshotter
	if cfg.StorageMode == "tmpfs" {
		snapInterval := time.Duration(cfg.TmpfsSnapshotInterval) * time.Second
		snap = tmpfs.New(cfg.DataDir, cfg.TmpfsPersistDir, snapInterval, metaStore.DB(), logger)

		// Restore from last snapshot (on startup)
		if err := snap.RestoreFromPersist(); err != nil {
			logger.Error("tmpfs restore failed", "error", err)
			os.Exit(1)
		}

		// Start periodic snapshot loop
		snap.Start()
	}

	// Set up authentication
	authenticator := buildAuthenticator(cfg, metaStore, logger)

	// Generate or load S3 credentials
	s3Creds := loadOrGenerateCredentials(cfg, metaStore, logger)

	// Build the HTTP server
	srv := httpsrv.NewServer(cfg, store, metaStore, authenticator, s3Creds, logger)

	httpServer := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      withRequestLogging(srv, logger),
		ReadTimeout:  config.HTTPReadTimeout,
		WriteTimeout: config.HTTPWriteTimeout,
		IdleTimeout:  config.HTTPIdleTimeout,
	}

	// Graceful shutdown handling
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("HTTP server listening", "addr", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-shutdownCh
	logger.Info("received signal, shutting down gracefully", "signal", sig.String())

	// Stop accepting new connections
	shutdownCtx, cancel := context.WithTimeout(context.Background(), config.ForceShutdownTimeout)
	defer cancel()

	// Give in-flight requests time to complete
	gracefulCtx, gracefulCancel := context.WithTimeout(shutdownCtx, config.GracefulShutdownTimeout)
	defer gracefulCancel()

	if err := httpServer.Shutdown(gracefulCtx); err != nil {
		logger.Error("HTTP server forced to shutdown", "error", err)
	}

	// Perform final tmpfs snapshot before exit
	if snap != nil {
		if err := snap.FinalSnapshot(); err != nil {
			logger.Error("tmpfs final snapshot failed", "error", err)
		}
		snap.Stop()
	}

	logger.Info("server stopped")
}

// buildAuthenticator creates the appropriate Authenticator based on config.
func buildAuthenticator(cfg *config.Config, metaStore *sync.MetadataStore, logger *slog.Logger) auth.Authenticator {
	switch cfg.AuthMode {
	case "none":
		logger.Info("auth mode: none (all requests pass through)")
		return auth.NewNoneAuth(cfg.UserID)

	case "token":
		logger.Info("auth mode: token")
		return auth.NewTokenAuth(cfg.AuthToken)

	case "users":
		logger.Info("auth mode: users")
		usersAuth := auth.NewUsersAuth(metaStore.DB())

		// Check if users table is empty — generate initial admin
		var count int
		if err := metaStore.DB().QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err == nil && count == 0 {
			password, err := usersAuth.GenerateInitialAdmin()
			if err != nil {
				logger.Error("failed to generate initial admin user", "error", err)
				os.Exit(1)
			}
			// Log the initial admin password ONCE and only in users mode
			logger.Warn("INITIAL ADMIN PASSWORD GENERATED",
				"username", "admin",
				"password", password,
				"message", "CHANGE THIS PASSWORD IMMEDIATELY via the API",
			)
		}
		return usersAuth

	default:
		logger.Error("unknown auth mode", "mode", cfg.AuthMode)
		os.Exit(1)
		return nil
	}
}

// withRequestLogging wraps an http.Handler with structured request logging.
// Logs: timestamp, request ID, method, path, status code, duration.
func withRequestLogging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := httpsrv.GenerateRequestID()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		logger.Info("request",
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// loadOrGenerateCredentials loads S3 credentials from credentials.txt or
// generates new ones on first startup.
func loadOrGenerateCredentials(cfg *config.Config, metaStore *sync.MetadataStore, logger *slog.Logger) []s3.Credential {
	credsPath := fmt.Sprintf("%s/credentials.txt", cfg.DataDir)
	creds, err := readCredentialsFile(credsPath)
	if err == nil && len(creds) > 0 {
		logger.Info("loaded S3 credentials from file", "count", len(creds))
		return creds
	}

	// Generate new credentials
	cred, err := s3.GenerateCredentials(cfg.UserID)
	if err != nil {
		logger.Error("failed to generate S3 credentials", "error", err)
		os.Exit(1)
	}

	// Write to credentials.txt with mode 0600
	line := fmt.Sprintf("%s:%s:%s\n", cred.AccessKey, cred.SecretKey, cred.UserID)
	if err := os.WriteFile(credsPath, []byte(line), 0600); err != nil {
		logger.Error("failed to write credentials file", "error", err)
		os.Exit(1)
	}

	logger.Info("generated new S3 credentials",
		"access_key", cred.AccessKey,
		"file", credsPath,
	)

	return []s3.Credential{*cred}
}

// readCredentialsFile parses the credentials.txt file.
// Format: access_key:secret_key:user_id (one per line)
func readCredentialsFile(path string) ([]s3.Credential, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var creds []s3.Credential
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 {
			creds = append(creds, s3.Credential{
				AccessKey: parts[0],
				SecretKey: parts[1],
				UserID:    parts[2],
			})
		}
	}
	return creds, nil
}
