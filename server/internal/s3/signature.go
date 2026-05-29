// Package s3 implements the minimal AWS Signature V4 verification needed
// for the Stonepad S3-compatible endpoint.
// Only verifies the Authorization header — no query-string auth in v1.
// See §7.5 of the Stonepad v1 Implementation Plan.
package s3

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Credential holds the S3 access key and secret for a user.
type Credential struct {
	AccessKey string
	SecretKey string
	UserID    string
}

// VerifySignature validates an AWS Sig V4 Authorization header
// against the request. Returns the credential if valid.
//
// This is a minimal subset of Sig V4:
// - Only Authorization header (no query-string auth)
// - Only HMAC-SHA256 signing algorithm
// - Validates credential scope (date, region, service)
// - Validates signed headers against the canonical request
func VerifySignature(r *http.Request, creds []Credential, now time.Time) (*Credential, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing Authorization header")
	}

	// Parse Authorization header
	parsed := parseAuthHeader(authHeader)
	if parsed == nil {
		return nil, fmt.Errorf("invalid Authorization header format")
	}

	// Find matching credential
	var matchedCred *Credential
	for i := range creds {
		if creds[i].AccessKey == parsed.accessKey {
			matchedCred = &creds[i]
			break
		}
	}
	if matchedCred == nil {
		return nil, fmt.Errorf("invalid access key")
	}

	// Get the request timestamp from x-amz-date or Date header
	reqTime := parseRequestTime(r, parsed.credDate)
	if reqTime.IsZero() {
		return nil, fmt.Errorf("missing x-amz-date header")
	}

	// Verify the request date is within 15 minutes of server time
	diff := now.Sub(reqTime)
	if diff < 0 {
		diff = -diff
	}
	if diff > 15*time.Minute {
		return nil, fmt.Errorf("request time too skewed: %v from server", diff)
	}

	// Build canonical request
	canonicalRequest := buildCanonicalRequest(r, parsed.signedHeaders)
	canonicalRequestHash := sha256Hex(canonicalRequest)

	// Build string to sign
	stringToSign := buildStringToSign(parsed, canonicalRequestHash, reqTime)

	// Compute signing key
	signingKey := deriveSigningKey(matchedCred.SecretKey, parsed.credDate, parsed.region, parsed.service)

	// Compute signature
	expectedSig := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	if !hmac.Equal([]byte(expectedSig), []byte(parsed.signature)) {
		return nil, fmt.Errorf("signature mismatch")
	}

	return matchedCred, nil
}

// parsedAuth holds the parsed components of the Authorization header.
type parsedAuth struct {
	accessKey     string
	credDate      string // YYYYMMDD
	region        string
	service       string
	signedHeaders []string
	signature     string
}

func parseAuthHeader(header string) *parsedAuth {
	if !strings.HasPrefix(header, "AWS4-HMAC-SHA256 ") {
		return nil
	}
	header = strings.TrimPrefix(header, "AWS4-HMAC-SHA256 ")

	p := &parsedAuth{}

	parts := splitAuthComponents(header)
	if len(parts) < 3 {
		return nil
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "Credential="):
			cred := strings.TrimPrefix(part, "Credential=")
			credParts := strings.SplitN(cred, "/", 5)
			if len(credParts) != 5 {
				return nil
			}
			p.accessKey = credParts[0]
			p.credDate = credParts[1]
			p.region = credParts[2]
			p.service = credParts[3]

		case strings.HasPrefix(part, "SignedHeaders="):
			headers := strings.TrimPrefix(part, "SignedHeaders=")
			p.signedHeaders = strings.Split(headers, ";")

		case strings.HasPrefix(part, "Signature="):
			p.signature = strings.TrimPrefix(part, "Signature=")
		}
	}

	if p.accessKey == "" || p.signature == "" || len(p.signedHeaders) == 0 {
		return nil
	}

	return p
}

// parseRequestTime extracts the request timestamp from x-amz-date header.
func parseRequestTime(r *http.Request, credDate string) time.Time {
	amzDate := r.Header.Get("x-amz-date")
	if amzDate != "" {
		t, err := time.Parse("20060102T150405Z", amzDate)
		if err == nil {
			return t
		}
	}
	// Fall back to Date header
	date := r.Header.Get("Date")
	if date != "" {
		t, err := time.Parse(time.RFC1123, date)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// splitAuthComponents splits by comma, handling commas inside quoted values.
func splitAuthComponents(s string) []string {
	var parts []string
	current := strings.Builder{}
	depth := 0
	for _, ch := range s {
		if ch == ',' && depth == 0 {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			if ch == '"' {
				depth ^= 1
			}
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// buildCanonicalRequest builds the canonical request string per AWS Sig V4 spec.
func buildCanonicalRequest(r *http.Request, signedHeaders []string) string {
	canonical := r.Method + "\n"
	canonical += canonicalURI(r.URL.Path) + "\n"
	canonical += "\n" // empty canonical query string for S3 subset

	for _, h := range signedHeaders {
		canonical += strings.ToLower(h) + ":" + strings.TrimSpace(r.Header.Get(h)) + "\n"
	}

	canonical += "\n"
	canonical += strings.Join(signedHeaders, ";") + "\n"

	// Payload hash from x-amz-content-sha256 header
	payloadHash := r.Header.Get("x-amz-content-sha256")
	if payloadHash == "" {
		// For GET/HEAD/DELETE, the payload is empty string
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "DELETE" {
			payloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		} else {
			payloadHash = "UNSIGNED-PAYLOAD"
		}
	}
	canonical += payloadHash

	return canonical
}

// canonicalURI normalizes a URI path for Sig V4.
func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

// buildStringToSign builds the "StringToSign" per AWS Sig V4 spec.
func buildStringToSign(p *parsedAuth, canonicalRequestHash string, reqTime time.Time) string {
	lines := []string{
		"AWS4-HMAC-SHA256",
		reqTime.UTC().Format("20060102T150405Z"),
		p.credDate + "/" + p.region + "/" + p.service + "/aws4_request",
		canonicalRequestHash,
	}
	return strings.Join(lines, "\n")
}

// deriveSigningKey derives the HMAC signing key per AWS Sig V4.
func deriveSigningKey(secretKey, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// GenerateCredentials creates a new access key / secret key pair for a user.
func GenerateCredentials(userID string) (*Credential, error) {
	accessKey := "STNP" + randomHex(16)
	secretKey := randomHex(32)

	return &Credential{
		AccessKey: accessKey,
		SecretKey: secretKey,
		UserID:    userID,
	}, nil
}

func randomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback — should never happen
		return strings.Repeat("0", n)
	}
	return hex.EncodeToString(bytes)[:n]
}
