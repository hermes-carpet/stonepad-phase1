package s3

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/hermes-carpet/stonepad/server/internal/storage"
)

func TestBuildListObjectsResult_Empty(t *testing.T) {
	result := BuildListObjectsResult("default", nil, ListObjectsOpts{})
	if result.Name != "default" {
		t.Errorf("bucket name = %q, want default", result.Name)
	}
	if result.IsTruncated {
		t.Error("empty result should not be truncated")
	}
	if len(result.Contents) != 0 {
		t.Errorf("expected 0 objects, got %d", len(result.Contents))
	}
}

func TestBuildListObjectsResult_WithNotes(t *testing.T) {
	metas := []storage.NoteMeta{
		{Path: "hello.md", ContentHash: "abc", SizeBytes: 100, ModifiedAt: time.Now()},
		{Path: "work/todo.md", ContentHash: "def", SizeBytes: 200, ModifiedAt: time.Now()},
	}

	result := BuildListObjectsResult("default", metas, ListObjectsOpts{})
	if result.KeyCount != 2 {
		t.Errorf("KeyCount = %d, want 2", result.KeyCount)
	}
	if len(result.Contents) != 2 {
		t.Errorf("expected 2 objects, got %d", len(result.Contents))
	}
	if result.Contents[0].Key != "hello.md" {
		t.Errorf("first key = %q, want hello.md", result.Contents[0].Key)
	}
	if result.Contents[1].Key != "work/todo.md" {
		t.Errorf("second key = %q, want work/todo.md", result.Contents[1].Key)
	}
}

func TestBuildListObjectsResult_Prefix(t *testing.T) {
	metas := []storage.NoteMeta{
		{Path: "work/a.md", ContentHash: "a", SizeBytes: 10, ModifiedAt: time.Now()},
		{Path: "work/b.md", ContentHash: "b", SizeBytes: 10, ModifiedAt: time.Now()},
		{Path: "personal/c.md", ContentHash: "c", SizeBytes: 10, ModifiedAt: time.Now()},
	}

	result := BuildListObjectsResult("default", metas, ListObjectsOpts{Prefix: "work/"})
	if result.KeyCount != 2 {
		t.Errorf("with prefix 'work/': KeyCount = %d, want 2", result.KeyCount)
	}
}

func TestBuildListObjectsResult_Delimiter(t *testing.T) {
	metas := []storage.NoteMeta{
		{Path: "work/a.md", ContentHash: "a", SizeBytes: 10, ModifiedAt: time.Now()},
		{Path: "work/b.md", ContentHash: "b", SizeBytes: 10, ModifiedAt: time.Now()},
		{Path: "root.md", ContentHash: "c", SizeBytes: 10, ModifiedAt: time.Now()},
	}

	result := BuildListObjectsResult("default", metas, ListObjectsOpts{Delimiter: "/"})
	// root.md is at top level, work/* are under "work/"
	if result.KeyCount != 1 {
		t.Errorf("with delimiter /: KeyCount = %d, want 1 (root.md)", result.KeyCount)
	}
	if len(*result.CommonPrefixes) != 1 {
		t.Errorf("expected 1 CommonPrefix (work/), got %d", len(*result.CommonPrefixes))
	}
}

func TestListAllMyBucketsResult_XML(t *testing.T) {
	result := NewListAllMyBucketsResult("default")
	xmlBytes, err := xml.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("xml.Marshal: %v", err)
	}
	xmlStr := string(xmlBytes)
	if !strings.Contains(xmlStr, "ListAllMyBucketsResult") {
		t.Error("XML missing ListAllMyBucketsResult element")
	}
	if !strings.Contains(xmlStr, "<Name>default</Name>") {
		t.Error("XML missing bucket name")
	}
}

func TestErrorResponse_XML(t *testing.T) {
	err := ErrorResponse{
		Code:     "NoSuchKey",
		Message:  "The specified key does not exist",
		Resource: "hello.md",
	}
	xmlBytes, _ := xml.MarshalIndent(err, "", "  ")
	xmlStr := string(xmlBytes)
	if !strings.Contains(xmlStr, "<Code>NoSuchKey</Code>") {
		t.Error("XML missing error code")
	}
}
