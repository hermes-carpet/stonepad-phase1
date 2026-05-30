package sync

import (
	"os"
	"testing"
)

func TestVacuumIfNeeded_SmallDB(t *testing.T) {
	// Create a minimal in-memory DB
	dbPath := t.TempDir() + "/small.db"
	store, err := OpenMetadataStore(dbPath)
	if err != nil {
		t.Fatalf("OpenMetadataStore: %v", err)
	}
	defer store.Close()

	// DB is empty (<10MB), vacuum should be a no-op
	err = store.VacuumIfNeeded(dbPath, 10*1024*1024)
	if err != nil {
		t.Fatalf("VacuumIfNeeded on small DB should not error: %v", err)
	}
}

func TestVacuumIfNeeded_NonexistentFile(t *testing.T) {
	store, err := OpenMetadataStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("OpenMetadataStore: %v", err)
	}
	defer store.Close()

	// Stat on nonexistent file should error
	err = store.VacuumIfNeeded("/nonexistent/path.db", 10*1024*1024)
	if err == nil {
		t.Fatal("Expected error for nonexistent DB file path")
	}
}

func TestVacuumIfNeeded_ZeroThreshold(t *testing.T) {
	dbPath := t.TempDir() + "/zerothreshold.db"
	store, err := OpenMetadataStore(dbPath)
	if err != nil {
		t.Fatalf("OpenMetadataStore: %v", err)
	}
	defer store.Close()

	// Zero threshold means always vacuum (file > 0 bytes)
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Skip("DB file is empty — zero-threshold test requires non-empty DB")
	}

	err = store.VacuumIfNeeded(dbPath, 0)
	if err != nil {
		t.Fatalf("VacuumIfNeeded with zero threshold: %v", err)
	}
}
