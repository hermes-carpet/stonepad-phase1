// Package auth provides authentication for the Stonepad server.
// Three modes are supported: none, token, and users.
// The Authenticator interface allows pluggable authentication — adding
// a JWT/OAuth implementation in v2 means adding a new struct, not rewriting handlers.
// See §7.7 of the Stonepad v1 Implementation Plan.
package auth

import "net/http"

// Authenticator validates incoming HTTP requests and returns the authenticated user's ID.
// Implementations: NoneAuth, TokenAuth, UsersAuth.
type Authenticator interface {
	// Authenticate validates the request and returns the authenticated user ID.
	// Returns ("", error) if authentication fails.
	Authenticate(r *http.Request) (userID string, err error)
}
