// Package storage tests the filesystem-backed storage implementation.
package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemStorage_PutGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFilesystemStorage(dir, "test")
	if err != nil {
		t.Fatalf("NewFilesystemStorage: %v", err)
	}

	ctx := context.Background()
	content := "# Hello, World!\n\nThis is a test note."

	// Put
	hash, err := store.Put(ctx, "hello.md", strings.NewReader(content))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(hash))
	}

	// Get
	reader, err := store.Get(ctx, "hello.md")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer reader.Close()
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != content {
		t.Errorf("Get: expected %q, got %q", content, string(got))
	}

	// Verify file exists on disk
	expectedPath := filepath.Join(dir, "notes", "test", "hello.md")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("file not on disk at %s: %v", expectedPath, err)
	}
}

func TestFilesystemStorage_NestedDirs(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFilesystemStorage(dir, "test")
	ctx := context.Background()

	paths := []string{
		"work/meetings/2026-04-22.md",
		"personal/journal/entry.md",
		"projects/stonepad/design.md",
	}

	for _, p := range paths {
		content := "note at " + p
		_, err := store.Put(ctx, p, strings.NewReader(content))
		if err != nil {
			t.Fatalf("Put %q: %v", p, err)
		}

		reader, err := store.Get(ctx, p)
		if err != nil {
			t.Fatalf("Get %q: %v", p, err)
		}
		got, _ := io.ReadAll(reader)
		reader.Close()
		if string(got) != content {
			t.Errorf("Get %q: expected %q, got %q", p, content, string(got))
		}
	}
}

func TestFilesystemStorage_PutOverwrite(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFilesystemStorage(dir, "test")
	ctx := context.Background()

	hash1, _ := store.Put(ctx, "note.md", strings.NewReader("version 1"))
	hash2, err := store.Put(ctx, "note.md", strings.NewReader("version 2"))
	if err != nil {
		t.Fatalf("Put overwrite: %v", err)
	}
	if hash1 == hash2 {
		t.Error("hash should change on overwrite")
	}

	reader, _ := store.Get(ctx, "note.md")
	got, _ := io.ReadAll(reader)
	reader.Close()
	if string(got) != "version 2" {
		t.Errorf("overwrite: expected 'version 2', got %q", string(got))
	}
}

func TestFilesystemStorage_Delete(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFilesystemStorage(dir, "test")
	ctx := context.Background()

	store.Put(ctx, "note.md", strings.NewReader("content"))

	// Delete
	if err := store.Delete(ctx, "note.md"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify gone
	_, err := store.Get(ctx, "note.md")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	// Deleting non-existent should not error
	if err := store.Delete(ctx, "missing.md"); err != nil {
		t.Errorf("Delete missing should not error: %v", err)
	}
}

func TestFilesystemStorage_Head(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFilesystemStorage(dir, "test")
	ctx := context.Background()

	store.Put(ctx, "note.md", strings.NewReader("hello"))

	meta, err := store.Head(ctx, "note.md")
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if meta.Path != "note.md" {
		t.Errorf("Head path: got %q, want 'note.md'", meta.Path)
	}
	if meta.SizeBytes != 5 {
		t.Errorf("Head size: got %d, want 5", meta.SizeBytes)
	}
	if meta.ContentHash == "" {
		t.Error("Head hash is empty")
	}

	// Head on missing
	_, err = store.Head(ctx, "missing.md")
	if err != ErrNotFound {
		t.Errorf("Head missing: expected ErrNotFound, got %v", err)
	}
}

func TestFilesystemStorage_List(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFilesystemStorage(dir, "test")
	ctx := context.Background()

	paths := []string{"a.md", "b.md", "work/c.md", "work/d.md"}
	for _, p := range paths {
		store.Put(ctx, p, strings.NewReader("content"))
	}

	// List all
	metas, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 4 {
		t.Errorf("List all: expected 4 notes, got %d", len(metas))
	}

	// List with prefix
	metas, err = store.List(ctx, "work")
	if err != nil {
		t.Fatalf("List prefix: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("List prefix 'work': expected 2 notes, got %d", len(metas))
	}

	// Verify sort order
	for i := 1; i < len(metas); i++ {
		if metas[i-1].Path > metas[i].Path {
			t.Errorf("list not sorted: %s > %s", metas[i-1].Path, metas[i].Path)
		}
	}
}

func TestFilesystemStorage_InvalidPaths(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewFilesystemStorage(dir, "test")
	ctx := context.Background()

	invalidPaths := []string{
		"",
		"/leading.md",
		"../escape.md",
	}

	for _, p := range invalidPaths {
		_, err := store.Put(ctx, p, strings.NewReader("x"))
		if err == nil {
			t.Errorf("expected error for invalid path %q", p)
		}
	}
}
