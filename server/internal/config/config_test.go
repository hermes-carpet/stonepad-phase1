// Package config tests configuration loading and validation.
package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("NOTES_LISTEN_ADDR")
	os.Unsetenv("NOTES_DATA_DIR")
	os.Unsetenv("NOTES_AUTH_MODE")
	os.Unsetenv("NOTES_WORKSPACE_ID")

	cfg := Load()

	if cfg.ListenAddr != ":8080" {
		t.Errorf("expected ListenAddr=:8080, got %s", cfg.ListenAddr)
	}
	if cfg.DataDir != "/data" {
		t.Errorf("expected DataDir=/data, got %s", cfg.DataDir)
	}
	if cfg.StorageMode != "direct" {
		t.Errorf("expected StorageMode=direct, got %s", cfg.StorageMode)
	}
	if cfg.AuthMode != "none" {
		t.Errorf("expected AuthMode=*** got %s", cfg.AuthMode)
	}
	if cfg.WorkspaceID != "default" {
		t.Errorf("expected WorkspaceID=default, got %s", cfg.WorkspaceID)
	}
	if cfg.MaxNoteSizeBytes != 5*1024*1024 {
		t.Errorf("expected MaxNoteSize=5MB, got %d", cfg.MaxNoteSizeBytes)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("NOTES_LISTEN_ADDR", ":9090")
	os.Setenv("NOTES_DATA_DIR", "/custom/data")
	os.Setenv("NOTES_AUTH_MODE", "token")
	os.Setenv("NOTES_AUTH_TOKEN", "secret123")
	os.Setenv("NOTES_WORKSPACE_ID", "myworkspace")
	defer func() {
		os.Unsetenv("NOTES_LISTEN_ADDR")
		os.Unsetenv("NOTES_DATA_DIR")
		os.Unsetenv("NOTES_AUTH_MODE")
		os.Unsetenv("NOTES_AUTH_TOKEN")
		os.Unsetenv("NOTES_WORKSPACE_ID")
	}()

	cfg := Load()

	if cfg.ListenAddr != ":9090" {
		t.Errorf("expected ListenAddr=:9090, got %s", cfg.ListenAddr)
	}
	if cfg.AuthMode != "token" {
		t.Errorf("expected AuthMode=token, got %s", cfg.AuthMode)
	}
	if cfg.AuthToken != "secret123" {
		t.Errorf("expected AuthToken=secret123, got %s", cfg.AuthToken)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{"valid direct none", &Config{StorageMode: "direct", AuthMode: "none", MaxNoteSizeBytes: 1024}, false},
		{"valid tmpfs none", &Config{StorageMode: "tmpfs", AuthMode: "none", MaxNoteSizeBytes: 1024}, false},
		{"valid token", &Config{StorageMode: "direct", AuthMode: "token", AuthToken: "secret", MaxNoteSizeBytes: 1024}, false},
		{"valid users", &Config{StorageMode: "direct", AuthMode: "users", MaxNoteSizeBytes: 1024}, false},
		{"invalid storage mode", &Config{StorageMode: "bad", AuthMode: "none", MaxNoteSizeBytes: 1024}, true},
		{"invalid auth mode", &Config{StorageMode: "direct", AuthMode: "oauth", MaxNoteSizeBytes: 1024}, true},
		{"token without token", &Config{StorageMode: "direct", AuthMode: "token", MaxNoteSizeBytes: 1024}, true},
		{"zero max note size", &Config{StorageMode: "direct", AuthMode: "none", MaxNoteSizeBytes: 0}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
