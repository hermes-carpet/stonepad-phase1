// Package auth implements no-op authentication.
// All requests pass through and are attributed to a configured user ID.
// Used when the server is behind Tailscale or on LAN and the user
// doesn't want app-level auth. See §7.7.
package auth

import "net/http"

// NoneAuth is the no-op authenticator. Every request passes through
// and is attributed to the configured default user.
type NoneAuth struct {
	DefaultUser string
}

// NewNoneAuth creates a new NoneAuth that attributes all requests to defaultUser.
func NewNoneAuth(defaultUser string) *NoneAuth {
	return &NoneAuth{DefaultUser: defaultUser}
}

// Authenticate always returns the default user ID without checking any credentials.
func (a *NoneAuth) Authenticate(r *http.Request) (string, error) {
	return a.DefaultUser, nil
}
