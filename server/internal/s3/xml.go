// Package s3 provides XML response structures for the S3-compatible API.
// Matches AWS S3 XML schemas closely enough that rclone and aws s3 CLI work.
// See §7.5 of the Stonepad v1 Implementation Plan.
package s3

import "encoding/xml"

// --- ListBuckets ---

// ListAllMyBucketsResult is the XML response for GET /s3/ (ListBuckets).
type ListAllMyBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Xmlns   string   `xml:"xmlns,attr"`
	Owner   Owner    `xml:"Owner"`
	Buckets Buckets  `xml:"Buckets"`
}

// Owner represents the bucket/object owner.
type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

// Buckets wraps a list of Bucket entries.
type Buckets struct {
	Bucket []Bucket `xml:"Bucket"`
}

// Bucket represents a single S3 bucket.
type Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

// --- ListObjectsV2 ---

// ListBucketResult is the XML response for ListObjectsV2.
type ListBucketResult struct {
	XMLName               xml.Name        `xml:"ListBucketResult"`
	Xmlns                 string          `xml:"xmlns,attr"`
	Name                  string          `xml:"Name"`
	Prefix                string          `xml:"Prefix"`
	KeyCount              int             `xml:"KeyCount"`
	MaxKeys               int             `xml:"MaxKeys"`
	IsTruncated           bool            `xml:"IsTruncated"`
	Contents              []Object        `xml:"Contents,omitempty"`
	CommonPrefixes        *[]CommonPrefix `xml:"CommonPrefixes>Prefix"`
	ContinuationToken     string          `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string          `xml:"NextContinuationToken,omitempty"`
	Delimiter             string          `xml:"Delimiter,omitempty"`
}

// Object represents a single S3 object (note) in ListObjectsV2.
type Object struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

// CommonPrefix represents a directory-like prefix in ListObjectsV2.
type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

// --- Errors ---

// ErrorResponse is the standard S3 XML error response.
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource,omitempty"`
	RequestID string   `xml:"RequestId,omitempty"`
}
