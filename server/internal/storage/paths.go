// Package storage provides path validation and sanitization utilities.
// Prevents directory traversal attacks and enforces path format rules.
// See §7.5 for the full path validation specification.
package storage

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MaxPathLen is the maximum total path length in bytes.
	MaxPathLen = 512

	// MaxDepth is the maximum number of directory components.
	MaxDepth = 16
)

var (
	// ValidPathComponent matches valid note path components.
	// Alphanumeric, underscore, hyphen, period, space.
	validComponent = regexp.MustCompile(`^[A-Za-z0-9._\- ]+$`)
)

// ValidateNotePath checks that a path is safe and conforms to the Stonepad path rules.
// Rules enforced:
//   - No empty path
//   - No leading slash
//   - No .. components
//   - No empty components (foo//bar)
//   - Components match ^[A-Za-z0-9._\- ]+$
//   - Path length <= 512 bytes
//   - Depth <= 16 components
//
// For native API paths, the extension must be .md (checked separately by caller).
// For S3 paths, dot-prefixed names are rejected (checked separately by caller).
func ValidateNotePath(path string) error {
	if path == "" {
		return fmt.Errorf("path must not be empty")
	}
	if path[0] == '/' {
		return fmt.Errorf("path must not start with '/': %q", path)
	}
	if len(path) > MaxPathLen {
		return fmt.Errorf("path exceeds maximum length of %d bytes: %d bytes", MaxPathLen, len(path))
	}

	components := strings.Split(path, "/")
	if len(components) > MaxDepth {
		return fmt.Errorf("path exceeds maximum depth of %d: %d levels", MaxDepth, len(components))
	}

	for i, comp := range components {
		if comp == "" {
			return fmt.Errorf("path contains empty component at position %d: %q", i, path)
		}
		if comp == ".." {
			return fmt.Errorf("path contains '..' component: %q", path)
		}
		if !validComponent.MatchString(comp) {
			return fmt.Errorf("path component %q contains invalid characters (allowed: A-Za-z0-9._- and space)", comp)
		}
	}

	return nil
}

// IsDotPrefixed returns true if any path component starts with '.'.
// Used by the S3 endpoint to reject dotfiles.
func IsDotPrefixed(path string) bool {
	for _, comp := range strings.Split(path, "/") {
		if strings.HasPrefix(comp, ".") {
			return true
		}
	}
	return false
}

// SafeJoin joins a base directory with a relative path, ensuring the result
// stays within the base directory (path traversal prevention).
func SafeJoin(base, relPath string) (string, error) {
	if err := ValidateNotePath(relPath); err != nil {
		return "", err
	}
	joined := filepath.Join(base, relPath)

	// Resolve symlinks for the base directory only.
	// The target path may not exist yet (e.g., during PUT with nested dirs).
	// We verify the joined path is under base by prefix check.
	realBase, err := filepath.EvalSymlinks(base)
	if err != nil {
		// If base itself doesn't exist, just use it as-is
		realBase = base
	}

	// Walk up parent directories until we find one that exists for symlink check
	checkPath := joined
	for {
		realCheck, err := filepath.EvalSymlinks(checkPath)
		if err == nil {
			// Found an existing ancestor — verify it's under base
			if !strings.HasPrefix(realCheck, realBase) {
				return "", fmt.Errorf("path traversal detected: %q escapes %q", relPath, base)
			}
			break
		}
		parent := filepath.Dir(checkPath)
		if parent == checkPath || parent == "/" || parent == "." {
			// Reached root — path is under base by construction
			break
		}
		checkPath = parent
	}

	return joined, nil
}
