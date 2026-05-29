// Package tmpfs tests — snapshot restore, periodic snapshot, and final snapshot.
package tmpfs

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSnapshotter_RestoreFromPersist_EmptyDir(t *testing.T) {
	dataDir := t.TempDir()
	persistDir := t.TempDir()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create schema
	db.Exec(`CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, value TEXT)`)

	s := New(dataDir, persistDir, 300*time.Second, db, slog.Default())

	// No previous snapshot — should be a no-op
	if err := s.RestoreFromPersist(); err != nil {
		t.Fatalf("restore from empty persist dir should not fail: %v", err)
	}
}

func TestSnapshotter_DoSnapshot_CreatesPersistFiles(t *testing.T) {
	dataDir := t.TempDir()
	persistDir := t.TempDir()

	// Create a note in the data directory (simulating tmpfs)
	notesDir := filepath.Join(dataDir, "notes", "default")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notesDir, "test.md"), []byte("# Hello tmpfs"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a SQLite DB in memory with schema
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, value TEXT)`)
	db.Exec(`INSERT INTO test (id, value) VALUES (1, 'persisted')`)

	s := New(dataDir, persistDir, 300*time.Second, db, slog.Default())

	// Perform snapshot
	if err := s.doSnapshot(); err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// Verify persist dir has the expected files
	checkFileExists(t, persistDir, "notes/default/test.md")
	checkFileExists(t, persistDir, "meta.db")
	checkFileExists(t, persistDir, "last_snapshot.txt")

	// Verify no stale .new directories
	if _, err := os.Stat(filepath.Join(persistDir, "notes.new")); !os.IsNotExist(err) {
		t.Error("notes.new should not exist after successful snapshot")
	}
	if _, err := os.Stat(filepath.Join(persistDir, "meta.db.new")); !os.IsNotExist(err) {
		t.Error("meta.db.new should not exist after successful snapshot")
	}

	// Verify notes content
	content, err := os.ReadFile(filepath.Join(persistDir, "notes", "default", "test.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# Hello tmpfs" {
		t.Errorf("expected '# Hello tmpfs', got '%s'", content)
	}
}

func TestSnapshotter_RestoreFromPersist_DataPresent(t *testing.T) {
	// Phase 1: create a snapshot via doSnapshot
	dataDir1 := t.TempDir()
	persistDir := t.TempDir()

	notesDir1 := filepath.Join(dataDir1, "notes", "default")
	os.MkdirAll(notesDir1, 0755)
	os.MkdirAll(filepath.Join(notesDir1, "sub"), 0755)
	os.WriteFile(filepath.Join(notesDir1, "alpha.md"), []byte("# Alpha"), 0644)
	os.WriteFile(filepath.Join(notesDir1, "sub", "beta.md"), []byte("# Beta"), 0644)

	db1, err := sql.Open("sqlite", filepath.Join(dataDir1, "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	db1.Exec(`CREATE TABLE IF NOT EXISTS kv (k TEXT PRIMARY KEY, v TEXT)`)
	db1.Exec(`INSERT INTO kv (k, v) VALUES ('key1', 'value1')`)
	db1.Close()

	db1, _ = sql.Open("sqlite", filepath.Join(dataDir1, "meta.db"))
	s1 := New(dataDir1, persistDir, 300*time.Second, db1, slog.Default())
	if err := s1.doSnapshot(); err != nil {
		t.Fatal(err)
	}
	db1.Close()

	// Phase 2: new data dir, restore from persistDir
	dataDir2 := t.TempDir()
	db2, err := sql.Open("sqlite", filepath.Join(dataDir2, "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	db2.Exec(`CREATE TABLE IF NOT EXISTS kv (k TEXT PRIMARY KEY, v TEXT)`)

	s2 := New(dataDir2, persistDir, 300*time.Second, db2, slog.Default())
	if err := s2.RestoreFromPersist(); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// Verify restored content
	checkFileExists(t, dataDir2, "notes/default/alpha.md")
	checkFileExists(t, dataDir2, "notes/default/sub/beta.md")

	content, _ := os.ReadFile(filepath.Join(dataDir2, "notes/default/alpha.md"))
	if string(content) != "# Alpha" {
		t.Errorf("expected '# Alpha', got '%s'", content)
	}
}

func TestSnapshotter_FinalSnapshot_CreatesPersistData(t *testing.T) {
	dataDir := t.TempDir()
	persistDir := t.TempDir()

	notesDir := filepath.Join(dataDir, "notes", "default")
	os.MkdirAll(notesDir, 0755)
	os.WriteFile(filepath.Join(notesDir, "final.md"), []byte("# Final"), 0644)

	db, err := sql.Open("sqlite", filepath.Join(dataDir, "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY)`)

	s := New(dataDir, persistDir, 300*time.Second, db, slog.Default())
	if err := s.FinalSnapshot(); err != nil {
		t.Fatalf("final snapshot failed: %v", err)
	}

	checkFileExists(t, persistDir, "notes/default/final.md")
	content, _ := os.ReadFile(filepath.Join(persistDir, "notes/default/final.md"))
	if string(content) != "# Final" {
		t.Errorf("expected '# Final', got '%s'", content)
	}
}

func TestBackupSQLite_DatabaseContents(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "source.db")
	backupPath := filepath.Join(dir, "dest.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`)
	db.Exec(`INSERT INTO items (id, name) VALUES (1, 'one'), (2, 'two')`)

	if err := backupSQLite(db, backupPath); err != nil {
		t.Fatalf("backupSQLite failed: %v", err)
	}
	db.Close()

	// Verify backup has the data
	db2, err := sql.Open("sqlite", backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()

	var count int
	if err := db2.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 rows in backup, got %d", count)
	}

	var name string
	if err := db2.QueryRow("SELECT name FROM items WHERE id = 1").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "one" {
		t.Errorf("expected 'one', got '%s'", name)
	}
}

func TestCopyDir_NestedFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	os.MkdirAll(filepath.Join(src, "a", "b"), 0755)
	os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0644)
	os.WriteFile(filepath.Join(src, "a", "mid.txt"), []byte("mid"), 0644)
	os.WriteFile(filepath.Join(src, "a", "b", "deep.txt"), []byte("deep"), 0644)

	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	checkFileExists(t, dst, "root.txt")
	checkFileExists(t, dst, "a/mid.txt")
	checkFileExists(t, dst, "a/b/deep.txt")

	content, _ := os.ReadFile(filepath.Join(dst, "a/b/deep.txt"))
	if string(content) != "deep" {
		t.Errorf("expected 'deep', got '%s'", content)
	}
}

func TestSnapshotter_AtomicRename_NoStaleNewDirs(t *testing.T) {
	dataDir := t.TempDir()
	persistDir := t.TempDir()

	notesDir := filepath.Join(dataDir, "notes", "default")
	os.MkdirAll(notesDir, 0755)
	os.WriteFile(filepath.Join(notesDir, "note.md"), []byte("hello"), 0644)

	db, _ := sql.Open("sqlite", filepath.Join(dataDir, "meta.db"))
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS x (y TEXT)`)

	s := New(dataDir, persistDir, 300*time.Second, db, slog.Default())

	// Run two snapshots to ensure .new cleanup works
	if err := s.doSnapshot(); err != nil {
		t.Fatal(err)
	}
	if err := s.doSnapshot(); err != nil {
		t.Fatal(err)
	}

	// No stale .new directories
	for _, suffix := range []string{"notes.new", "meta.db.new"} {
		if _, err := os.Stat(filepath.Join(persistDir, suffix)); !os.IsNotExist(err) {
			t.Errorf("stale directory %s should not exist", suffix)
		}
	}
}

func TestSnapshotter_StartAndStop(t *testing.T) {
	dataDir := t.TempDir()
	persistDir := t.TempDir()

	notesDir := filepath.Join(dataDir, "notes", "default")
	os.MkdirAll(notesDir, 0755)
	os.WriteFile(filepath.Join(notesDir, "a.md"), []byte("a"), 0644)

	db, _ := sql.Open("sqlite", filepath.Join(dataDir, "meta.db"))
	defer db.Close()
	db.Exec(`CREATE TABLE IF NOT EXISTS t (x TEXT)`)

	s := New(dataDir, persistDir, 100*time.Millisecond, db, slog.Default())
	s.Start()

	// Wait for at least one snapshot to fire
	time.Sleep(300 * time.Millisecond)

	s.Stop()

	// Verify snapshot was created
	checkFileExists(t, persistDir, "notes/default/a.md")
	checkFileExists(t, persistDir, "meta.db")
	checkFileExists(t, persistDir, "last_snapshot.txt")

	// Verify last_snapshot.txt has a valid timestamp
	data, err := os.ReadFile(filepath.Join(persistDir, "last_snapshot.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "T") {
		t.Error("last_snapshot.txt should contain ISO timestamp")
	}
}

// ── helpers ──

func checkFileExists(t *testing.T, dir, relPath string) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", fullPath)
	}
}
