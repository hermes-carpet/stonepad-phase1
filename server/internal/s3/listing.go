// Package s3 provides listing logic for the S3-compatible endpoint.
// Implements ListObjectsV2 with prefix filtering, delimiter support,
// and pagination via continuation tokens.
// See §7.5 of the Stonepad v1 Implementation Plan.
package s3

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hermes-carpet/stonepad/server/internal/storage"
)

// ListObjectsOpts holds the parameters for a ListObjectsV2 request.
type ListObjectsOpts struct {
	Prefix            string
	Delimiter         string
	MaxKeys           int
	ContinuationToken string
}

// DefaultMaxKeys is the default page size for ListObjectsV2.
const DefaultMaxKeys = 1000

// BuildListObjectsResult constructs a ListBucketResult from filesystem metadata.
// Implements prefix filtering, delimiter-based grouping (CommonPrefixes),
// and pagination via opaque continuation tokens.
func BuildListObjectsResult(bucket string, metas []storage.NoteMeta, opts ListObjectsOpts) *ListBucketResult {
	if opts.MaxKeys <= 0 {
		opts.MaxKeys = DefaultMaxKeys
	}

	// Filter by prefix
	var filtered []storage.NoteMeta
	for _, m := range metas {
		if strings.HasPrefix(m.Path, opts.Prefix) {
			filtered = append(filtered, m)
		}
	}

	// Sort by path
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Path < filtered[j].Path
	})

	// Handle delimiter (group by common prefix)
	var objects []Object
	prefixSet := make(map[string]bool)

	for _, m := range filtered {
		// Remove the prefix portion for delimiter logic
		relativeKey := strings.TrimPrefix(m.Path, opts.Prefix)

		if opts.Delimiter != "" {
			// Check if path has the delimiter after the prefix
			idx := strings.Index(relativeKey, opts.Delimiter)
			if idx >= 0 {
				// This is a "folder" — add as CommonPrefix
				commonPrefix := opts.Prefix + relativeKey[:idx+len(opts.Delimiter)]
				prefixSet[commonPrefix] = true
				continue
			}
		}

		// Regular object
		objects = append(objects, Object{
			Key:          m.Path,
			LastModified: m.ModifiedAt.Format(time.RFC3339),
			ETag:         fmt.Sprintf(`"%s"`, m.ContentHash),
			Size:         m.SizeBytes,
			StorageClass: "STANDARD",
		})
	}

	// Build CommonPrefixes list from set
	var commonPrefixes []CommonPrefix
	for p := range prefixSet {
		commonPrefixes = append(commonPrefixes, CommonPrefix{Prefix: p})
	}
	sort.Slice(commonPrefixes, func(i, j int) bool {
		return commonPrefixes[i].Prefix < commonPrefixes[j].Prefix
	})

	// Pagination
	total := len(objects) + len(commonPrefixes)
	isTruncated := total > opts.MaxKeys
	var truncatedObjects []Object
	var truncatedPrefixes []CommonPrefix
	var nextToken string

	if isTruncated {
		// Take first MaxKeys items, distributing between objects and prefixes
		count := 0
		oi, pi := 0, 0
		for oi < len(objects) || pi < len(commonPrefixes) {
			if count >= opts.MaxKeys {
				break
			}
			// Interleave: pick the lexicographically smaller
			if oi >= len(objects) {
				truncatedPrefixes = append(truncatedPrefixes, commonPrefixes[pi])
				pi++
			} else if pi >= len(commonPrefixes) {
				truncatedObjects = append(truncatedObjects, objects[oi])
				oi++
			} else if objects[oi].Key < commonPrefixes[pi].Prefix {
				truncatedObjects = append(truncatedObjects, objects[oi])
				oi++
			} else {
				truncatedPrefixes = append(truncatedPrefixes, commonPrefixes[pi])
				pi++
			}
			count++
		}
		nextToken = generateContinuationToken()
		objects = truncatedObjects
		commonPrefixes = truncatedPrefixes
	}

	result := &ListBucketResult{
		Xmlns:         "http://s3.amazonaws.com/doc/2006-03-01/",
		Name:          bucket,
		Prefix:        opts.Prefix,
		KeyCount:      len(objects),
		MaxKeys:       opts.MaxKeys,
		IsTruncated:   isTruncated,
		Contents:      objects,
		CommonPrefixes: commonPrefixPtr(commonPrefixes),
		ContinuationToken:     opts.ContinuationToken,
		NextContinuationToken: nextToken,
		Delimiter:             opts.Delimiter,
	}

	return result
}

// NewListAllMyBucketsResult creates a ListBuckets response with a single bucket.
func NewListAllMyBucketsResult(bucketName string) *ListAllMyBucketsResult {
	return &ListAllMyBucketsResult{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Owner: Owner{
			ID:          "stonepad",
			DisplayName: "stonepad",
		},
		Buckets: Buckets{
			Bucket: []Bucket{{
				Name:         bucketName,
				CreationDate: time.Now().UTC().Format(time.RFC3339),
			}},
		},
	}
}

func generateContinuationToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// commonPrefixPtr returns nil if the slice is empty, otherwise returns a pointer to the slice.
// This ensures Go's XML marshaling correctly omits the CommonPrefixes element when empty.
func commonPrefixPtr(s []CommonPrefix) *[]CommonPrefix {
	if len(s) == 0 {
		return nil
	}
	return &s
}
