// Package http provides S3-compatible HTTP handlers for the Stonepad server.
// Implements the minimal S3 subset: ListBuckets, ListObjectsV2,
// HeadObject, GetObject, PutObject, DeleteObject.
// See §7.5 of the Stonepad v1 Implementation Plan.
package http

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hermes-carpet/stonepad/server/internal/s3"
	"github.com/hermes-carpet/stonepad/server/internal/storage"
)

// S3Handler holds the dependencies for S3 endpoint handlers.
type S3Handler struct {
	store       storage.Storage
	metaStore   MetadataStorer
	creds       []s3.Credential
	workspaceID string
	maxNoteSize int64
}

// MetadataStorer is the interface the S3 handler needs from the metadata store.
type MetadataStorer interface {
	GetNoteHash(workspaceID, path string) (string, error)
	UpsertNote(workspaceID, path, contentHash string, sizeBytes int64) error
	DeleteNote(workspaceID, path string) error
	RecordAudit(workspaceID, userID, action, path, contentHash string) error
	CountNotes(workspaceID string) (int, error)
}

// NewS3Handler creates a new S3 handler.
func NewS3Handler(
	store storage.Storage,
	metaStore MetadataStorer,
	creds []s3.Credential,
	workspaceID string,
	maxNoteSize int64,
) *S3Handler {
	return &S3Handler{
		store:       store,
		metaStore:   metaStore,
		creds:       creds,
		workspaceID: workspaceID,
		maxNoteSize: maxNoteSize,
	}
}

// HandleListBuckets handles GET /s3/
func (h *S3Handler) HandleListBuckets(w http.ResponseWriter, r *http.Request) {
	result := s3.NewListAllMyBucketsResult(h.workspaceID)
	writeXML(w, http.StatusOK, result)
}

// HandleListObjects handles GET /s3/{bucket}?list-type=2
func (h *S3Handler) HandleListObjects(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	if bucket != h.workspaceID {
		writeS3Error(w, r, "NoSuchBucket", "The specified bucket does not exist", bucket)
		return
	}

	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")
	continuationToken := r.URL.Query().Get("continuation-token")
	maxKeysStr := r.URL.Query().Get("max-keys")

	maxKeys := s3.DefaultMaxKeys
	if maxKeysStr != "" {
		if n, err := strconv.Atoi(maxKeysStr); err == nil && n > 0 {
			maxKeys = n
		}
	}

	metas, err := h.store.List(r.Context(), "")
	if err != nil {
		writeS3Error(w, r, "InternalError", "Failed to list objects", bucket)
		return
	}

	opts := s3.ListObjectsOpts{
		Prefix:            prefix,
		Delimiter:         delimiter,
		MaxKeys:           maxKeys,
		ContinuationToken: continuationToken,
	}

	result := s3.BuildListObjectsResult(bucket, metas, opts)
	writeXML(w, http.StatusOK, result)
}

// HandleHeadObject handles HEAD /s3/{bucket}/{key...}
func (h *S3Handler) HandleHeadObject(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key...")
	if key == "" {
		writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
		return
	}

	meta, err := h.store.Head(r.Context(), key)
	if err == storage.ErrNotFound {
		writeS3Error(w, r, "NoSuchKey", "The specified key does not exist", key)
		return
	}
	if err != nil {
		writeS3Error(w, r, "InternalError", "Failed to read object metadata", key)
		return
	}

	w.Header().Set("Content-Type", "text/markdown")
	w.Header().Set("Content-Length", strconv.FormatInt(meta.SizeBytes, 10))
	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, meta.ContentHash))
	w.Header().Set("Last-Modified", meta.ModifiedAt.UTC().Format(http.TimeFormat))
	w.Header().Set("x-amz-meta-content-hash", meta.ContentHash)
	w.Header().Set("x-amz-meta-modified-at", meta.ModifiedAt.UTC().Format(time.RFC3339))
	w.WriteHeader(http.StatusOK)
}

// HandleGetObject handles GET /s3/{bucket}/{key...}
func (h *S3Handler) HandleGetObject(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key...")
	if key == "" {
		writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
		return
	}

	reader, err := h.store.Get(r.Context(), key)
	if err == storage.ErrNotFound {
		writeS3Error(w, r, "NoSuchKey", "The specified key does not exist", key)
		return
	}
	if err != nil {
		writeS3Error(w, r, "InternalError", "Failed to read object", key)
		return
	}
	defer reader.Close()

	// Get metadata for headers
	meta, metaErr := h.store.Head(r.Context(), key)

	w.Header().Set("Content-Type", "text/markdown")
	if metaErr == nil {
		w.Header().Set("Content-Length", strconv.FormatInt(meta.SizeBytes, 10))
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, meta.ContentHash))
		w.Header().Set("Last-Modified", meta.ModifiedAt.UTC().Format(http.TimeFormat))
		w.Header().Set("x-amz-meta-content-hash", meta.ContentHash)
	}
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, reader); err != nil {
		// Client likely disconnected; not much we can do
		return
	}
}

// HandlePutObject handles PUT /s3/{bucket}/{key...}
func (h *S3Handler) HandlePutObject(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key...")
	if key == "" {
		writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
		return
	}

	// Reject dot-prefixed paths
	if storage.IsDotPrefixed(key) {
		writeS3Error(w, r, "InvalidArgument", "Key must not be dot-prefixed", key)
		return
	}

	// Enforce max note size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxNoteSize)

	// Read body into memory for hash verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			writeS3Error(w, r, "EntityTooLarge", "Object exceeds maximum size", key)
			return
		}
		writeS3Error(w, r, "InternalError", "Failed to read request body", key)
		return
	}

	// Verify Content-MD5 if provided
	contentMD5 := r.Header.Get("Content-MD5")
	if contentMD5 != "" {
		expectedMD5, err := base64.StdEncoding.DecodeString(contentMD5)
		if err == nil {
			actualHash := md5.Sum(bodyBytes)
			if !hmacEq(expectedMD5, actualHash[:]) {
				writeS3Error(w, r, "BadDigest", "The Content-MD5 you specified did not match", key)
				return
			}
		}
	}

	// Store the note
	contentHash, err := h.store.Put(r.Context(), key, strings.NewReader(string(bodyBytes)))
	if err != nil {
		if strings.Contains(err.Error(), "invalid path") {
			writeS3Error(w, r, "InvalidArgument", err.Error(), key)
			return
		}
		writeS3Error(w, r, "InternalError", "Failed to store object", key)
		return
	}

	// Get size
	meta, metaErr := h.store.Head(r.Context(), key)
	sizeBytes := int64(len(bodyBytes))
	if metaErr == nil {
		sizeBytes = meta.SizeBytes
	}

	// Update metadata
	h.metaStore.UpsertNote(h.workspaceID, key, contentHash, sizeBytes)

	// Record audit
	userID := "owner" // S3 auth maps to the credential's user
	h.metaStore.RecordAudit(h.workspaceID, userID, "update", key, contentHash)

	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, contentHash))
	w.Header().Set("x-amz-meta-content-hash", contentHash)
	w.WriteHeader(http.StatusOK)
}

// HandleDeleteObject handles DELETE /s3/{bucket}/{key...}
func (h *S3Handler) HandleDeleteObject(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key...")
	if key == "" {
		writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
		return
	}

	if err := h.store.Delete(r.Context(), key); err != nil {
		writeS3Error(w, r, "InternalError", "Failed to delete object", key)
		return
	}

	h.metaStore.DeleteNote(h.workspaceID, key)
	userID := "owner"
	h.metaStore.RecordAudit(h.workspaceID, userID, "delete", key, "")

	w.WriteHeader(http.StatusNoContent)
}

// authS3Request validates the Sig V4 signature on the request.
func (h *S3Handler) authS3Request(r *http.Request) (*s3.Credential, error) {
	return s3.VerifySignature(r, h.creds, time.Now())
}

// --- Parameterized handlers for catch-all route (no PathValue available) ---

// ListObjectsForBucket returns a handler for listing objects in the given bucket.
func (h *S3Handler) ListObjectsForBucket(bucket string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if bucket != h.workspaceID {
			writeS3Error(w, r, "NoSuchBucket", "The specified bucket does not exist", bucket)
			return
		}
		prefix := r.URL.Query().Get("prefix")
		maxKeysStr := r.URL.Query().Get("max-keys")
		maxKeys := s3.DefaultMaxKeys
		if maxKeysStr != "" {
			if n, err := strconv.Atoi(maxKeysStr); err == nil && n > 0 {
				maxKeys = n
			}
		}
		metas, err := h.store.List(r.Context(), "")
		if err != nil {
			writeS3Error(w, r, "InternalError", "Failed to list objects", bucket)
			return
		}
		opts := s3.ListObjectsOpts{
			Prefix:            prefix,
			MaxKeys:           maxKeys,
			ContinuationToken: r.URL.Query().Get("continuation-token"),
		}
		result := s3.BuildListObjectsResult(bucket, metas, opts)
		writeXML(w, http.StatusOK, result)
	}
}

// GetObjectForBucketKey returns a handler for GET on a specific bucket/key.
func (h *S3Handler) GetObjectForBucketKey(bucket, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if bucket != h.workspaceID {
			writeS3Error(w, r, "NoSuchBucket", "The specified bucket does not exist", bucket)
			return
		}
		if key == "" {
			writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
			return
		}
		reader, err := h.store.Get(r.Context(), key)
		if err == storage.ErrNotFound {
			writeS3Error(w, r, "NoSuchKey", "The specified key does not exist", key)
			return
		}
		if err != nil {
			writeS3Error(w, r, "InternalError", "Failed to read object", key)
			return
		}
		defer reader.Close()
		meta, metaErr := h.store.Head(r.Context(), key)
		w.Header().Set("Content-Type", "text/markdown")
		if metaErr == nil {
			w.Header().Set("Content-Length", strconv.FormatInt(meta.SizeBytes, 10))
			w.Header().Set("ETag", fmt.Sprintf(`"%s"`, meta.ContentHash))
			w.Header().Set("Last-Modified", meta.ModifiedAt.UTC().Format(http.TimeFormat))
			w.Header().Set("x-amz-meta-content-hash", meta.ContentHash)
		}
		w.WriteHeader(http.StatusOK)
		io.Copy(w, reader)
	}
}

// HeadObjectForBucketKey returns a handler for HEAD on a specific bucket/key.
func (h *S3Handler) HeadObjectForBucketKey(bucket, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if bucket != h.workspaceID {
			writeS3Error(w, r, "NoSuchBucket", "The specified bucket does not exist", bucket)
			return
		}
		if key == "" {
			writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
			return
		}
		meta, err := h.store.Head(r.Context(), key)
		if err == storage.ErrNotFound {
			writeS3Error(w, r, "NoSuchKey", "The specified key does not exist", key)
			return
		}
		if err != nil {
			writeS3Error(w, r, "InternalError", "Failed to read object metadata", key)
			return
		}
		w.Header().Set("Content-Type", "text/markdown")
		w.Header().Set("Content-Length", strconv.FormatInt(meta.SizeBytes, 10))
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, meta.ContentHash))
		w.Header().Set("Last-Modified", meta.ModifiedAt.UTC().Format(http.TimeFormat))
		w.Header().Set("x-amz-meta-content-hash", meta.ContentHash)
		w.Header().Set("x-amz-meta-modified-at", meta.ModifiedAt.UTC().Format(time.RFC3339))
		w.WriteHeader(http.StatusOK)
	}
}

// PutObjectForBucketKey returns a handler for PUT on a specific bucket/key.
func (h *S3Handler) PutObjectForBucketKey(bucket, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if bucket != h.workspaceID {
			writeS3Error(w, r, "NoSuchBucket", "The specified bucket does not exist", bucket)
			return
		}
		if key == "" {
			writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
			return
		}
		if storage.IsDotPrefixed(key) {
			writeS3Error(w, r, "InvalidArgument", "Key must not be dot-prefixed", key)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, h.maxNoteSize)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			if strings.Contains(err.Error(), "http: request body too large") {
				writeS3Error(w, r, "EntityTooLarge", "Object exceeds maximum size", key)
				return
			}
			writeS3Error(w, r, "InternalError", "Failed to read request body", key)
			return
		}
		contentHash, err := h.store.Put(r.Context(), key, strings.NewReader(string(bodyBytes)))
		if err != nil {
			writeS3Error(w, r, "InternalError", "Failed to store object", key)
			return
		}
		sizeBytes := int64(len(bodyBytes))
		h.metaStore.UpsertNote(h.workspaceID, key, contentHash, sizeBytes)
		h.metaStore.RecordAudit(h.workspaceID, "owner", "update", key, contentHash)
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, contentHash))
		w.WriteHeader(http.StatusOK)
	}
}

// DeleteObjectForBucketKey returns a handler for DELETE on a specific bucket/key.
func (h *S3Handler) DeleteObjectForBucketKey(bucket, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if bucket != h.workspaceID {
			writeS3Error(w, r, "NoSuchBucket", "The specified bucket does not exist", bucket)
			return
		}
		if key == "" {
			writeS3Error(w, r, "InvalidArgument", "Key is required", h.workspaceID)
			return
		}
		if err := h.store.Delete(r.Context(), key); err != nil {
			writeS3Error(w, r, "InternalError", "Failed to delete object", key)
			return
		}
		h.metaStore.DeleteNote(h.workspaceID, key)
		h.metaStore.RecordAudit(h.workspaceID, "owner", "delete", key, "")
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- S3 error helpers ---

func writeS3Error(w http.ResponseWriter, r *http.Request, code, message, resource string) {
	resp := s3.ErrorResponse{
		Code:      code,
		Message:   message,
		Resource:  resource,
		RequestID: GenerateRequestID(),
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(s3ErrorToHTTP(code))
	xml.NewEncoder(w).Encode(resp)
}

// s3ErrorToHTTP maps S3 error codes to HTTP status codes.
func s3ErrorToHTTP(code string) int {
	switch code {
	case "NoSuchBucket", "NoSuchKey":
		return http.StatusNotFound
	case "AccessDenied":
		return http.StatusForbidden
	case "BadDigest", "InvalidArgument", "EntityTooLarge":
		return http.StatusBadRequest
	case "NotImplemented":
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func writeXML(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)
	xml.NewEncoder(w).Encode(data)
}

// hmacEq is a constant-time byte comparison helper.
func hmacEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
