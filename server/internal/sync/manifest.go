// Package sync provides server-side manifest generation.
// The manifest is a JSON representation of all notes in a workspace,
// used by the native API endpoint and for sync diffing by clients.
// See §7.4 for the manifest response format specification.
package sync

import (
	"time"

	"github.com/hermes-carpet/stonepad/server/internal/storage"
)

// Manifest is the server-side manifest response.
// Matches the format specified in §7.4.
type Manifest struct {
	Version     int           `json:"version"`
	WorkspaceID string        `json:"workspace_id"`
	GeneratedAt string        `json:"generated_at"`
	Notes       []ManifestEntry `json:"notes"`
}

// ManifestEntry represents a single note in the manifest.
type ManifestEntry struct {
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	SizeBytes   int64  `json:"size_bytes"`
	ModifiedAt  string `json:"modified_at"`
}

// BuildManifest creates a manifest from the filesystem metadata.
// This is the canonical source — even though we have SQLite, we derive
// from disk to guarantee the "files are truth" principle.
func BuildManifest(workspaceID string, st storage.Storage) (*Manifest, error) {
	metas, err := st.List(nil, "") // empty context in Go means context.Background()
	if err != nil {
		return nil, err
	}

	entries := make([]ManifestEntry, 0, len(metas))
	for _, m := range metas {
		entries = append(entries, ManifestEntry{
			Path:        m.Path,
			ContentHash: m.ContentHash,
			SizeBytes:   m.SizeBytes,
			ModifiedAt:  m.ModifiedAt.Format(time.RFC3339),
		})
	}

	return &Manifest{
		Version:     1,
		WorkspaceID: workspaceID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Notes:       entries,
	}, nil
}
