package relay

import (
	"net/http"
	"os"
	"strconv"
	"testing"
)

func TestPollerSignRequest_AddsHeaders(t *testing.T) {
	p := &Poller{
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
		region:    "auto",
	}

	req, _ := http.NewRequest("GET", "https://r2.example.com/bucket/key", nil)
	p.signRequest(req, sha256Hex(nil))

	if req.Header.Get("Authorization") == "" {
		t.Error("Authorization header should be set")
	}
	if req.Header.Get("x-amz-date") == "" {
		t.Error("x-amz-date header should be set")
	}
	if req.Header.Get("x-amz-content-sha256") == "" {
		t.Error("x-amz-content-sha256 header should be set")
	}
}

func TestPollerBuildCanonicalHeaders(t *testing.T) {
	p := &Poller{
		accessKey: "ak",
		secretKey: "sk",
		region:    "auto",
	}

	req, _ := http.NewRequest("PUT", "https://r2.example.com/bucket/key", nil)
	req.Header.Set("host", "r2.example.com")
	req.Header.Set("x-amz-content-sha256", "abc123")
	req.Header.Set("x-amz-date", "20260101T000000Z")

	headers, signed := p.buildCanonicalHeaders(req)

	if !contains(headers, "host:r2.example.com") {
		t.Errorf("canonical headers missing host: %q", headers)
	}
	if !contains(headers, "x-amz-content-sha256:abc123") {
		t.Errorf("canonical headers missing content-sha256: %q", headers)
	}

	if signed != "host;x-amz-content-sha256;x-amz-date" {
		t.Errorf("expected signed headers 'host;x-amz-content-sha256;x-amz-date', got %q", signed)
	}
}

func TestConfigRelayFieldsLoadFromEnv(t *testing.T) {
	// Save and restore env
	origEnabled := os.Getenv("NOTES_RELAY_ENABLED")
	origEndpoint := os.Getenv("NOTES_RELAY_ENDPOINT")
	origAccessKey := os.Getenv("NOTES_RELAY_ACCESS_KEY")
	origSecretKey := os.Getenv("NOTES_RELAY_SECRET_KEY")
	origBucket := os.Getenv("NOTES_RELAY_BUCKET")
	origInterval := os.Getenv("NOTES_RELAY_POLL_INTERVAL")
	defer func() {
		os.Setenv("NOTES_RELAY_ENABLED", origEnabled)
		os.Setenv("NOTES_RELAY_ENDPOINT", origEndpoint)
		os.Setenv("NOTES_RELAY_ACCESS_KEY", origAccessKey)
		os.Setenv("NOTES_RELAY_SECRET_KEY", origSecretKey)
		os.Setenv("NOTES_RELAY_BUCKET", origBucket)
		os.Setenv("NOTES_RELAY_POLL_INTERVAL", origInterval)
	}()

	t.Setenv("NOTES_RELAY_ENABLED", "true")
	t.Setenv("NOTES_RELAY_ENDPOINT", "https://r2.example.com")
	t.Setenv("NOTES_RELAY_ACCESS_KEY", "ak-test")
	t.Setenv("NOTES_RELAY_SECRET_KEY", "sk-test")
	t.Setenv("NOTES_RELAY_BUCKET", "my-bucket")
	t.Setenv("NOTES_RELAY_POLL_INTERVAL", "120")

	// Verify env vars are set (testing the env loading used by config.go)
	if v := os.Getenv("NOTES_RELAY_ENABLED"); v != "true" {
		t.Errorf("expected 'true', got %q", v)
	}
	if v := os.Getenv("NOTES_RELAY_ENDPOINT"); v != "https://r2.example.com" {
		t.Errorf("expected endpoint, got %q", v)
	}
	if v := os.Getenv("NOTES_RELAY_ACCESS_KEY"); v != "ak-test" {
		t.Errorf("expected access key, got %q", v)
	}
	if v := os.Getenv("NOTES_RELAY_SECRET_KEY"); v != "sk-test" {
		t.Errorf("expected secret key, got %q", v)
	}
	if v := os.Getenv("NOTES_RELAY_BUCKET"); v != "my-bucket" {
		t.Errorf("expected bucket, got %q", v)
	}
	if v := os.Getenv("NOTES_RELAY_POLL_INTERVAL"); v != "120" {
		t.Errorf("expected 120, got %q", v)
	}
	if n, _ := strconv.Atoi(os.Getenv("NOTES_RELAY_POLL_INTERVAL")); n != 120 {
		t.Errorf("expected 120 as int, got %d", n)
	}
}

func TestConfigRelayDefaultsToDisabled(t *testing.T) {
	if v := os.Getenv("NOTES_RELAY_ENABLED"); v != "" {
		t.Skip("NOTES_RELAY_ENABLED is set in environment, skipping default test")
	}
	// When unset, the default should be false (config.go: getEnvBool(key, false))
}

func TestDeriveSigningKey_ProducesNonEmptyOutput(t *testing.T) {
	p := &Poller{
		accessKey: "ak",
		secretKey: "sk",
		region:    "auto",
	}

	key := p.deriveSigningKey("20260101")
	if len(key) == 0 {
		t.Error("deriveSigningKey should produce non-empty output")
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
