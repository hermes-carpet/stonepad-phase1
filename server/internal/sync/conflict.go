// Package sync provides server-side conflict detection.
// The server is mostly passive about conflicts — the client handles
// conflict resolution. The server only provides If-Match precondition
// checking for optimistic concurrency control.
// See §7.11 for the full conflict handling specification.
package sync

// ConflictInfo is returned when an If-Match precondition fails.
type ConflictInfo struct {
	CurrentHash string `json:"current_hash"`
	Path        string `json:"path"`
}
