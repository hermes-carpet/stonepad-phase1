// Package storage implements direct filesystem-backed note storage.
// Notes are stored as plain .md files in a real folder hierarchy.
// All writes use atomic rename (write to temp file, then rename).
// See §7.6 and §9.3 for the full file write semantics specification.
package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FilesystemStorage implements Storage on a local directory.
// Notes live under {baseDir}/notes/{workspace_id}/...note hierarchy...
type FilesystemStorage struct {
	baseDir     string
	workspaceID string
	notesDir    string // baseDir/notes/workspaceID
}

// NewFilesystemStorage creates a filesystem-backed storage.
// The base directory and notes subdirectory are created if needed.
func NewFilesystemStorage(baseDir, workspaceID string) (*FilesystemStorage, error) {
	notesDir := filepath.Join(baseDir, "notes", workspaceID)
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return nil, fmt.Errorf("creating notes directory %s: %w", notesDir, err)
	}
	return &FilesystemStorage{
		baseDir:     baseDir,
		workspaceID: workspaceID,
		notesDir:    notesDir,
	}, nil
}

// Put writes note content atomically.
// Steps: validate path, ensure parent dirs, write to temp file, fsync, rename.
// Returns the SHA-256 content hash of the stored bytes.
func (s *FilesystemStorage) Put(ctx context.Context, path string, content io.Reader) (string, error) {
	if err := ValidateNotePath(path); err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	fullPath, err := SafeJoin(s.notesDir, path)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return "", fmt.Errorf("creating parent directory: %w", err)
	}

	// Read all content into memory for hashing
	data, err := io.ReadAll(content)
	if err != nil {
		return "", fmt.Errorf("reading content: %w", err)
	}

	// Compute SHA-256
	hash := sha256.Sum256(data)
	contentHash := hex.EncodeToString(hash[:])

	// Write to temp file in the same directory
	tmpName := fmt.Sprintf("%s.tmp-%s", filepath.Base(path), randomSuffix(8))
	tmpPath := filepath.Join(parentDir, tmpName)

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing temp file: %w", err)
	}

	// fsync the temp file
	f, err := os.Open(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("opening temp file for fsync: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("fsyncing temp file: %w", err)
	}
	f.Close()

	// Atomic rename to final path
	if err := os.Rename(tmpPath, fullPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("renaming temp file: %w", err)
	}

	return contentHash, nil
}

// Get reads note content from the given path.
func (s *FilesystemStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ValidateNotePath(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	fullPath, err := SafeJoin(s.notesDir, path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	f, err := os.Open(fullPath)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	return f, nil
}

// Delete removes a note. Returns nil if the file doesn't exist (idempotent).
func (s *FilesystemStorage) Delete(ctx context.Context, path string) error {
	if err := ValidateNotePath(path); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	fullPath, err := SafeJoin(s.notesDir, path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing file: %w", err)
	}

	return nil
}

// Head returns metadata about a note without reading its content.
func (s *FilesystemStorage) Head(ctx context.Context, path string) (*NoteMeta, error) {
	if err := ValidateNotePath(path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	fullPath, err := SafeJoin(s.notesDir, path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Compute content hash by reading the file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading file for hash: %w", err)
	}
	hash := sha256.Sum256(data)

	return &NoteMeta{
		Path:        path,
		ContentHash: hex.EncodeToString(hash[:]),
		SizeBytes:   info.Size(),
		ModifiedAt:  info.ModTime(),
	}, nil
}

// List returns all note paths under an optional prefix, sorted by path.
func (s *FilesystemStorage) List(ctx context.Context, prefix string) ([]NoteMeta, error) {
	var metas []NoteMeta

	walkDir := s.notesDir
	if prefix != "" {
		// Sanitize prefix — it comes from user input (S3 prefix parameter)
		prefix = strings.TrimPrefix(prefix, "/")
		walkDir = filepath.Join(s.notesDir, prefix)
		// Ensure walkDir is still under notesDir
		if !strings.HasPrefix(walkDir, s.notesDir) {
			return nil, fmt.Errorf("prefix escapes notes directory")
		}
	}

	err := filepath.Walk(walkDir, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files we can't access
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		relPath, err := filepath.Rel(s.notesDir, fullPath)
		if err != nil {
			return nil
		}

		// Compute content hash
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil
		}
		hash := sha256.Sum256(data)

		metas = append(metas, NoteMeta{
			Path:        relPath,
			ContentHash: hex.EncodeToString(hash[:]),
			SizeBytes:   info.Size(),
			ModifiedAt:  info.ModTime(),
		})
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("walking notes directory: %w", err)
	}

	// Sort by path for deterministic output
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Path < metas[j].Path
	})

	return metas, nil
}

// BasePath returns the notes directory path.
func (s *FilesystemStorage) BasePath() string {
	return s.notesDir
}

// randomSuffix generates a random alphanumeric suffix for temp files.
func randomSuffix(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		l := big.NewInt(int64(len(letters)))
		idx, err := rand.Int(rand.Reader, l)
		if err != nil {
			b[i] = letters[i%len(letters)]
			continue
		}
		b[i] = letters[idx.Int64()]
	}
	return string(b)
}
