// Package auth implements shared-token authentication.
// Every request must include an Authorization: Bearer header matching
// the configured token. See §7.7.
package auth

import (
	"fmt"
	"net/http"
	"strings"
)

// TokenAuth validates requests using a shared bearer token.
type TokenAuth struct {
	Token string
}

// NewTokenAuth creates a new TokenAuth with the given shared token.
func NewTokenAuth(token string) *TokenAuth {
	return &TokenAuth{Token: token}
}

// Authenticate extracts the bearer token from the request and compares
// it against the configured token.
func (a *TokenAuth) Authenticate(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok {
		return "", fmt.Errorf("invalid Authorization header format: expected 'Bearer <token>'")
	}

	if token != a.Token {
		return "", fmt.Errorf("invalid token")
	}

	// All token-authenticated requests map to a single user.
	// In v2, tokens could be associated with specific users.
	return "owner", nil
}
