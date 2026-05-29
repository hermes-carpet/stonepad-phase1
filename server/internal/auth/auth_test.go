package auth

import (
	"net/http"
	"strings"
	"testing"
)

func TestNoneAuth(t *testing.T) {
	a := NewNoneAuth("testuser")

	req, _ := http.NewRequest("GET", "/", nil)
	userID, err := a.Authenticate(req)

	if err != nil {
		t.Errorf("NoneAuth should never error: %v", err)
	}
	if userID != "testuser" {
		t.Errorf("expected userID='testuser', got %q", userID)
	}
}

func TestTokenAuth_MissingHeader(t *testing.T) {
	a := NewTokenAuth("secret")

	req, _ := http.NewRequest("GET", "/", nil)
	_, err := a.Authenticate(req)
	if err == nil {
		t.Error("expected error for missing Authorization header")
	}
}

func TestTokenAuth_InvalidFormat(t *testing.T) {
	a := NewTokenAuth("secret")

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	_, err := a.Authenticate(req)
	if err == nil {
		t.Error("expected error for non-Bearer token")
	}
}

func TestTokenAuth_WrongToken(t *testing.T) {
	a := NewTokenAuth("secret")

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	_, err := a.Authenticate(req)
	if err == nil {
		t.Error("expected error for wrong token")
	}
}

func TestTokenAuth_Success(t *testing.T) {
	a := NewTokenAuth("secret")

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	userID, err := a.Authenticate(req)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if userID != "owner" {
		t.Errorf("expected userID='owner', got %q", userID)
	}
}

func TestHashPassword(t *testing.T) {
	password := "mysecretpassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Error("hash is empty")
	}
	// Should contain colon separator
	if !strings.Contains(hash, ":") {
		t.Error("hash should contain colon separator")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "correcthorsebatterystaple"
	hash, _ := HashPassword(password)

	if !VerifyPassword(password, hash) {
		t.Error("VerifyPassword should return true for correct password")
	}
	if VerifyPassword("wrong", hash) {
		t.Error("VerifyPassword should return false for wrong password")
	}
}

func TestVerifyPassword_InvalidFormat(t *testing.T) {
	if VerifyPassword("anything", "invalid") {
		t.Error("VerifyPassword should return false for invalid format")
	}
	if VerifyPassword("anything", "bad:format:extra") {
		t.Error("VerifyPassword should return false for invalid format")
	}
}
