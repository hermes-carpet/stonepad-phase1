// Package sync manages the SQLite metadata index and related operations.
// The metadata index mirrors the filesystem — it is derived from the .md files
// on disk and can be rebuilt at any time by rescanning.
// See §7.3 and §7.4 of the Stonepad v1 Implementation Plan.
package sync

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// MetadataStore manages the SQLite metadata index for notes.
type MetadataStore struct {
	db *sql.DB
}

// OpenMetadataStore opens (or creates) the SQLite database at the given path.
// Runs schema initialization and applies pragmas on every connection.
func OpenMetadataStore(dbPath string) (*MetadataStore, error) {
	// Make sure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Apply pragmas on the new connection
	if err := applyPragmas(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("applying pragmas: %w", err)
	}

	store := &MetadataStore{db: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return store, nil
}

// Close cleanly closes the database connection.
func (m *MetadataStore) Close() error {
	return m.db.Close()
}

// DB returns the underlying *sql.DB for direct queries by other packages.
func (m *MetadataStore) DB() *sql.DB {
	return m.db
}

// InitWorkspace ensures the workspace row exists.
func (m *MetadataStore) InitWorkspace(workspaceID string) error {
	_, err := m.db.Exec(
		`INSERT OR IGNORE INTO workspaces (workspace_id, display_name, created_at)
		 VALUES (?, ?, ?)`,
		workspaceID, workspaceID, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// InitUser ensures the default user exists (for none/token auth modes).
func (m *MetadataStore) InitUser(userID string) error {
	_, err := m.db.Exec(
		`INSERT OR IGNORE INTO users (user_id, username, created_at)
		 VALUES (?, ?, ?)`,
		userID, userID, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// UpsertNote creates or updates a note's metadata row.
func (m *MetadataStore) UpsertNote(workspaceID, path, contentHash string, sizeBytes int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := m.db.Exec(
		`INSERT INTO notes (workspace_id, path, content_hash, size_bytes, modified_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (workspace_id, path) DO UPDATE SET
		   content_hash = excluded.content_hash,
		   size_bytes = excluded.size_bytes,
		   modified_at = excluded.modified_at`,
		workspaceID, path, contentHash, sizeBytes, now, now,
	)
	return err
}

// DeleteNote removes a note's metadata row.
func (m *MetadataStore) DeleteNote(workspaceID, path string) error {
	_, err := m.db.Exec(
		`DELETE FROM notes WHERE workspace_id = ? AND path = ?`,
		workspaceID, path,
	)
	return err
}

// RecordAudit inserts an audit log entry.
func (m *MetadataStore) RecordAudit(workspaceID, userID, action, path, contentHash string) error {
	_, err := m.db.Exec(
		`INSERT INTO audit_log (workspace_id, user_id, action, path, content_hash, occurred_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		workspaceID, userID, action, path, contentHash, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// GetNoteHash returns the content_hash for a note at the given path.
// Returns empty string and error if not found.
func (m *MetadataStore) GetNoteHash(workspaceID, path string) (string, error) {
	var hash string
	err := m.db.QueryRow(
		`SELECT content_hash FROM notes WHERE workspace_id = ? AND path = ?`,
		workspaceID, path,
	).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

// AllNoteMetas returns metadata for all notes in a workspace.
func (m *MetadataStore) AllNoteMetas(workspaceID string) ([]IndexedNote, error) {
	rows, err := m.db.Query(
		`SELECT path, content_hash, size_bytes, modified_at
		 FROM notes WHERE workspace_id = ?
		 ORDER BY path`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying notes: %w", err)
	}
	defer rows.Close()

	var notes []IndexedNote
	for rows.Next() {
		var n IndexedNote
		var modifiedAt string
		if err := rows.Scan(&n.Path, &n.ContentHash, &n.SizeBytes, &modifiedAt); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		n.ModifiedAt, _ = time.Parse(time.RFC3339, modifiedAt)
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// CountNotes returns the number of notes in a workspace.
func (m *MetadataStore) CountNotes(workspaceID string) (int, error) {
	var count int
	err := m.db.QueryRow(
		`SELECT COUNT(*) FROM notes WHERE workspace_id = ?`,
		workspaceID,
	).Scan(&count)
	return count, err
}

// IndexedNote is a note's metadata as stored in SQLite.
type IndexedNote struct {
	Path        string
	ContentHash string
	SizeBytes   int64
	ModifiedAt  time.Time
}

func (m *MetadataStore) initSchema() error {
	// Check if schema_version table exists
	var exists int
	err := m.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'").Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking schema: %w", err)
	}

	if exists == 0 {
		// First run — execute the embedded schema
		if _, err := m.db.Exec(schemaSQL); err != nil {
			return fmt.Errorf("executing schema: %w", err)
		}
	}

	return nil
}

func applyPragmas(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("pragma %q: %w", p, err)
		}
	}
	return nil
}

// VacuumIfNeeded runs PRAGMA incremental_vacuum if the database file
// exceeds the given thresholdBytes. Called once per week on startup per §7.3.
func (m *MetadataStore) VacuumIfNeeded(dbPath string, thresholdBytes int64) error {
	info, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("stat db: %w", err)
	}
	if info.Size() < thresholdBytes {
		return nil
	}
	if _, err := m.db.Exec("PRAGMA incremental_vacuum"); err != nil {
		return fmt.Errorf("vacuum: %w", err)
	}
	return nil
}
