// Package config provides server configuration loaded from environment variables.
// All configuration is via env vars — no config files in v1.
// See §7.2 of the Stonepad v1 Implementation Plan for the full spec.
package config

import (
	"os"
	"strconv"
)

// Config holds all server configuration loaded from environment variables.
type Config struct {
	ListenAddr            string
	DataDir               string
	StorageMode           string // "direct" or "tmpfs"
	TmpfsSnapshotInterval int
	TmpfsPersistDir       string
	AuthMode              string // "none", "token", or "users"
	AuthToken             string
	WorkspaceID           string
	UserID                string
	LogLevel              string
	S3EndpointEnabled     bool
	NativeAPIEnabled      bool
	MaxNoteSizeBytes      int64
	MaxNotesPerWorkspace  int
	RelayEnabled          bool
	RelayEndpoint         string
	RelayAccessKey        string
	RelaySecretKey        string
	RelayBucket           string
	RelayPollInterval     int
}

// Load reads all configuration from environment variables with sensible defaults.
// See §7.2 for the full env var specification.
func Load() *Config {
	return &Config{
		ListenAddr:            getEnv("NOTES_LISTEN_ADDR", ":8080"),
		DataDir:               getEnv("NOTES_DATA_DIR", "/data"),
		StorageMode:           getEnv("NOTES_STORAGE_MODE", "direct"),
		TmpfsSnapshotInterval: getEnvInt("NOTES_TMPFS_SNAPSHOT_INTERVAL", 300),
		TmpfsPersistDir:       getEnv("NOTES_TMPFS_PERSIST_DIR", "/data/persist"),
		AuthMode:              getEnv("NOTES_AUTH_MODE", "none"),
		AuthToken:             getEnv("NOTES_AUTH_TOKEN", ""),
		WorkspaceID:           getEnv("NOTES_WORKSPACE_ID", "default"),
		UserID:                getEnv("NOTES_USER_ID", "owner"),
		LogLevel:              getEnv("NOTES_LOG_LEVEL", "info"),
		S3EndpointEnabled:     getEnvBool("NOTES_S3_ENDPOINT_ENABLED", true),
		NativeAPIEnabled:      getEnvBool("NOTES_NATIVE_ENDPOINT_ENABLED", true),
		MaxNoteSizeBytes:      getEnvInt64("NOTES_MAX_NOTE_SIZE_BYTES", 5*1024*1024),
		MaxNotesPerWorkspace:  getEnvInt("NOTES_MAX_NOTES_PER_WORKSPACE", 100000),
		RelayEnabled:          getEnvBool("NOTES_RELAY_ENABLED", false),
		RelayEndpoint:         getEnv("NOTES_RELAY_ENDPOINT", ""),
		RelayAccessKey:        getEnv("NOTES_RELAY_ACCESS_KEY", ""),
		RelaySecretKey:        getEnv("NOTES_RELAY_SECRET_KEY", ""),
		RelayBucket:           getEnv("NOTES_RELAY_BUCKET", ""),
		RelayPollInterval:     getEnvInt("NOTES_RELAY_POLL_INTERVAL", 300),
	}
}

// Validate checks that the configuration is internally consistent.
// Returns nil if valid, or an error describing the first problem.
func (c *Config) Validate() error {
	if c.StorageMode != "direct" && c.StorageMode != "tmpfs" {
		return &ConfigError{"NOTES_STORAGE_MODE must be 'direct' or 'tmpfs'"}
	}
	if c.AuthMode != "none" && c.AuthMode != "token" && c.AuthMode != "users" {
		return &ConfigError{"NOTES_AUTH_MODE must be 'none', 'token', or 'users'"}
	}
	if c.AuthMode == "token" && c.AuthToken == "" {
		return &ConfigError{"NOTES_AUTH_TOKEN is required when NOTES_AUTH_MODE is 'token'"}
	}
	if c.MaxNoteSizeBytes <= 0 {
		return &ConfigError{"NOTES_MAX_NOTE_SIZE_BYTES must be positive"}
	}
	return nil
}

// ConfigError is a simple configuration-level error.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return "config: " + e.Message
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func getEnvInt64(key string, defaultVal int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultVal
	}
	return n
}

func getEnvBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}
