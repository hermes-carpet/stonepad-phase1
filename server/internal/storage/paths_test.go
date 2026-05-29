// Package storage tests path validation rules.
package storage

import (
	"testing"
)

func TestValidateNotePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		// Valid paths
		{"hello.md", false},
		{"work/meeting-notes.md", false},
		{"projects/2026/May 15 notes.md", false},
		{"a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p.md", false}, // 16 components (max)

		// Invalid paths
		{"", true},                        // empty
		{"/leading/slash.md", true},       // leading slash
		{"../escape.md", true},            // ..
		{"foo//bar.md", true},             // empty component
		{"foo/../bar.md", true},           // .. in middle
		{string(make([]byte, 513)), true}, // too long
	}
	// Build a 513-char path
	longPath := ""
	for i := 0; i < 513; i++ {
		longPath += "a"
	}
	tests = append(tests, struct {
		path    string
		wantErr bool
	}{longPath, true})

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if len(tt.path) > 50 {
				t.Logf("testing long path (%d bytes)", len(tt.path))
			}
			err := ValidateNotePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNotePath(%q) error = %v, wantErr = %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestIsDotPrefixed(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{"hello.md", false},
		{".hidden.md", true},
		{"work/.hidden/note.md", true},
		{"normal/path.md", false},
		{".git/HEAD", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := IsDotPrefixed(tt.path); got != tt.expect {
				t.Errorf("IsDotPrefixed(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

func TestValidateNotePath_InvalidChars(t *testing.T) {
	invalidChars := []string{"note!.md", "hello@world.md", "path#test.md", "file$.md", "test%note.md"}
	for _, p := range invalidChars {
		err := ValidateNotePath(p)
		if err == nil {
			t.Errorf("expected error for path with invalid chars: %q", p)
		}
	}
}
