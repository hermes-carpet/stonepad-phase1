// Package storage handles file I/O and path validation for the Stonepad server.
// The Storage interface abstracts the filesystem so that different backends
// (direct disk, tmpfs with snapshot) can be swapped in.
// See §7.5 and §7.6 of the Stonepad v1 Implementation Plan.
package storage

import (
	"context"
	"io"
	"time"
)

// NoteMeta is metadata about a stored note, used for manifest generation.
type NoteMeta struct {
	Path        string
	ContentHash string // lowercase hex SHA-256
	SizeBytes   int64
	ModifiedAt  time.Time
}

// Storage is the interface for note persistence backends.
type Storage interface {
	// Put writes note content to the given path atomically.
	// Creates parent directories as needed.
	// Returns the SHA-256 content hash.
	Put(ctx context.Context, path string, content io.Reader) (contentHash string, err error)

	// Get reads note content from the given path.
	// Returns io.ErrUnexpectedEOF or similar if the read is incomplete.
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes a note at the given path.
	// Returns nil if the file doesn't exist (idempotent).
	Delete(ctx context.Context, path string) error

	// Head returns metadata about a note without reading its content.
	// Returns ErrNotFound if the path doesn't exist.
	Head(ctx context.Context, path string) (*NoteMeta, error)

	// List returns all note paths under an optional prefix.
	// Results are sorted by path for consistent manifest generation.
	List(ctx context.Context, prefix string) ([]NoteMeta, error)

	// BasePath returns the filesystem root for note storage.
	BasePath() string
}

// ErrNotFound is returned when a path doesn't exist.
var ErrNotFound = errNotFound{}

type errNotFound struct{}

func (e errNotFound) Error() string { return "note not found" }
