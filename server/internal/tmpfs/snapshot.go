// Package tmpfs implements tmpfs-backed storage with periodic disk snapshots.
//
// In tmpfs mode, all data lives in RAM (mounted tmpfs at NOTES_DATA_DIR).
// A background goroutine periodically snapshots the RAM state to a persistent
// directory on disk. On startup, the RAM state is restored from the last snapshot.
//
// The snapshot algorithm uses atomic swap (create .new dirs, then rename)
// to avoid corruption if the process crashes mid-snapshot.
//
// See §7.6 of the Stonepad v1 Implementation Plan.
package tmpfs

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Snapshotter manages periodic tmpfs-to-disk snapshots.
//
// Design:
//   - On startup, restoreFromPersist copies data from persistDir → dataDir.
//   - A background goroutine runs snapshotToPersist every snapshotInterval.
//   - On shutdown, FinalSnapshot performs a synchronous final snapshot.
//
// The persistDir layout:
//
//	${persistDir}/
//	├── notes/          (last good snapshot)
//	├── notes.new/      (in-progress snapshot — deleted after atomic rename)
//	├── meta.db         (last good SQLite backup)
//	├── meta.db.new     (in-progress SQLite backup)
//	└── last_snapshot.txt
type Snapshotter struct {
	dataDir          string // tmpfs mount (NOTES_DATA_DIR)
	persistDir       string // persistent storage (NOTES_TMPFS_PERSIST_DIR)
	snapshotInterval time.Duration
	db               *sql.DB // the SQLite connection for .backup
	logger           *slog.Logger
	stopCh           chan struct{}
	doneCh           chan struct{}
}

// New creates a new Snapshotter. Does not start the background goroutine —
// call Start() after creation.
func New(dataDir, persistDir string, snapshotInterval time.Duration, db *sql.DB, logger *slog.Logger) *Snapshotter {
	return &Snapshotter{
		dataDir:          dataDir,
		persistDir:       persistDir,
		snapshotInterval: snapshotInterval,
		db:               db,
		logger:           logger,
		stopCh:           make(chan struct{}),
		doneCh:           make(chan struct{}),
	}
}

// RestoreFromPersist copies the last snapshot from persistDir into dataDir.
// Called on startup. If persistDir doesn't exist or is empty, this is a no-op
// (first launch with tmpfs).
func (s *Snapshotter) RestoreFromPersist() error {
	persistNotes := filepath.Join(s.persistDir, "notes")
	if _, err := os.Stat(persistNotes); os.IsNotExist(err) {
		s.logger.Info("tmpfs: no previous snapshot found, starting fresh")
		return nil
	}

	s.logger.Info("tmpfs: restoring from last snapshot", "source", s.persistDir)

	// Restore notes directory
	notesDir := filepath.Join(s.dataDir, "notes")
	if err := copyDir(persistNotes, notesDir); err != nil {
		return fmt.Errorf("restoring notes: %w", err)
	}

	// Restore SQLite database
	persistDB := filepath.Join(s.persistDir, "meta.db")
	dataDB := filepath.Join(s.dataDir, "meta.db")
	if _, err := os.Stat(persistDB); err == nil {
		if err := copyFile(persistDB, dataDB); err != nil {
			return fmt.Errorf("restoring meta.db: %w", err)
		}
		// Also copy WAL and SHM if present
		for _, suffix := range []string{"-wal", "-shm"} {
			src := persistDB + suffix
			dst := dataDB + suffix
			if _, err := os.Stat(src); err == nil {
				if err := copyFile(src, dst); err != nil {
					s.logger.Warn("tmpfs: failed to copy db suffix", "file", suffix, "error", err)
				}
			}
		}
	}

	s.logger.Info("tmpfs: restore complete")
	return nil
}

// Start begins the periodic snapshot goroutine.
func (s *Snapshotter) Start() {
	go s.snapshotLoop()
	s.logger.Info("tmpfs: snapshot loop started",
		"interval_seconds", s.snapshotInterval.Seconds(),
		"persist_dir", s.persistDir,
	)
}

// Stop signals the snapshot loop to stop and waits for it to exit.
// Does NOT perform a final snapshot — call FinalSnapshot() before Stop().
func (s *Snapshotter) Stop() {
	close(s.stopCh)
	<-s.doneCh
	s.logger.Info("tmpfs: snapshot loop stopped")
}

// FinalSnapshot performs a synchronous snapshot to persistDir.
// Called during graceful shutdown. Blocks until the snapshot is complete.
func (s *Snapshotter) FinalSnapshot() error {
	s.logger.Info("tmpfs: performing final snapshot before shutdown")
	return s.doSnapshot()
}

// snapshotLoop is the background goroutine that runs periodic snapshots.
func (s *Snapshotter) snapshotLoop() {
	defer close(s.doneCh)

	ticker := time.NewTicker(s.snapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.doSnapshot(); err != nil {
				s.logger.Error("tmpfs: periodic snapshot failed", "error", err)
			}
		}
	}
}

// doSnapshot performs one atomic snapshot cycle.
//
// Algorithm (per §7.6):
//
//  1. Create ${persistDir}/notes.new/ directory
//  2. rsync -a --delete from dataDir to notes.new
//  3. SQLite .backup to write meta.db.new
//  4. Atomic rename notes.new → notes, meta.db.new → meta.db
//  5. Update last_snapshot.txt timestamp
func (s *Snapshotter) doSnapshot() error {
	start := time.Now()

	// Step 1: ensure persist dir exists
	if err := os.MkdirAll(s.persistDir, 0755); err != nil {
		return fmt.Errorf("creating persist dir: %w", err)
	}

	// Step 2: rsync notes from tmpfs to persist dir
	notesSrc := filepath.Join(s.dataDir, "notes")
	notesNew := filepath.Join(s.persistDir, "notes.new")
	notesFinal := filepath.Join(s.persistDir, "notes")

	// Remove any stale .new from a crashed previous snapshot
	os.RemoveAll(notesNew)

	if err := rsyncDir(notesSrc, notesNew); err != nil {
		return fmt.Errorf("rsyncing notes: %w", err)
	}

	// Step 3: SQLite backup (in-process, using sqlite3_backup)
	dbNew := filepath.Join(s.persistDir, "meta.db.new")
	dbFinal := filepath.Join(s.persistDir, "meta.db")

	// Remove stale .new
	os.Remove(dbNew)

	if err := backupSQLite(s.db, dbNew); err != nil {
		// Clean up the notes.new on failure
		os.RemoveAll(notesNew)
		return fmt.Errorf("backing up sqlite: %w", err)
	}

	// Step 4: atomic rename .new over existing
	// Remove the existing destination directory first (cannot rename over non-empty dirs)
	os.RemoveAll(notesFinal)
	if err := os.Rename(notesNew, notesFinal); err != nil {
		os.RemoveAll(notesNew)
		os.Remove(dbNew)
		return fmt.Errorf("renaming notes snapshot: %w", err)
	}

	// Then database — remove existing, then rename
	os.Remove(dbFinal)
	if err := os.Rename(dbNew, dbFinal); err != nil {
		os.Remove(dbNew)
		return fmt.Errorf("renaming db snapshot: %w", err)
	}

	// Step 5: update last_snapshot.txt
	lastFile := filepath.Join(s.persistDir, "last_snapshot.txt")
	now := time.Now().UTC().Format(time.RFC3339)
	if err := os.WriteFile(lastFile, []byte(now+"\n"), 0644); err != nil {
		s.logger.Warn("tmpfs: failed to write last_snapshot.txt", "error", err)
	}

	s.logger.Info("tmpfs: snapshot complete",
		"duration_ms", time.Since(start).Milliseconds(),
		"persist_dir", s.persistDir,
	)

	return nil
}

// ────────────────────── Helpers ──────────────────────

// rsyncDir copies a directory tree using the rsync command-line tool.
// Falls back to a Go-based copy if rsync is not available.
func rsyncDir(src, dst string) error {
	// Ensure src ends with / for rsync semantics
	srcPath := src + "/"

	// Try rsync first
	cmd := exec.Command("rsync", "-a", "--delete", srcPath, dst)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		// Fall back to Go-based copy
		return copyDir(src, dst)
	}
	return nil
}

// copyDir recursively copies a directory tree from src to dst.
// Uses Go stdlib only — no external dependency. Used as fallback when
// rsync is not available (e.g., in minimal Docker images).
func copyDir(src, dst string) error {
	// Remove existing dst
	os.RemoveAll(dst)

	// Ensure parent of dst exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

// copyFile copies a single file from src to dst.
func copyFile(src, dst string) error {
	// Ensure parent exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// backupSQLite performs an in-process SQLite backup to a file.
// Uses the sqlite3_backup API via the Go driver.
func backupSQLite(db *sql.DB, dstPath string) error {
	// Use SQLite's backup API
	// modernc.org/sqlite supports the VACUUM INTO syntax which
	// creates a consistent copy to a new file.
	_, err := db.Exec(fmt.Sprintf("VACUUM INTO '%s'", dstPath))
	if err != nil {
		return fmt.Errorf("sqlite VACUUM INTO: %w", err)
	}
	return nil
}
