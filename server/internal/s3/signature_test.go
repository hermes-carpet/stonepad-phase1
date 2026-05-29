package s3

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseAuthHeader(t *testing.T) {
	header := "AWS4-HMAC-SHA256 Credential=AKID123/20260422/us-east-1/s3/aws4_request, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=abcd1234ef567890"

	p := parseAuthHeader(header)
	if p == nil {
		t.Fatal("parseAuthHeader returned nil")
	}
	if p.accessKey != "AKID123" {
		t.Errorf("accessKey = %q, want AKID123", p.accessKey)
	}
	if p.credDate != "20260422" {
		t.Errorf("credDate = %q, want 20260422", p.credDate)
	}
	if p.region != "us-east-1" {
		t.Errorf("region = %q, want us-east-1", p.region)
	}
	if p.service != "s3" {
		t.Errorf("service = %q, want s3", p.service)
	}
	if len(p.signedHeaders) != 3 {
		t.Errorf("signedHeaders count = %d, want 3", len(p.signedHeaders))
	}
	if p.signature != "abcd1234ef567890" {
		t.Errorf("signature = %q", p.signature)
	}
}

func TestParseAuthHeader_Invalid(t *testing.T) {
	tests := []string{
		"",
		"Basic abc123",
		"AWS4-HMAC-SHA256 Credential=badformat, Signature=abc",
		"AWS4-HMAC-SHA256 SignedHeaders=host, Signature=abc",
	}

	for _, h := range tests {
		if p := parseAuthHeader(h); p != nil {
			t.Errorf("expected nil for %q, got %+v", h, p)
		}
	}
}

// Test signing key derivation using the known AWS test suite example.
// Reference: https://docs.aws.amazon.com/general/latest/gr/signature-v4-test-suite.html
func TestSigningKeyDerivation(t *testing.T) {
	secretKey := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	date := "20120215"
	region := "us-east-1"
	service := "iam"

	key := deriveSigningKey(secretKey, date, region, service)
	expected := []byte{
		0xf4, 0x78, 0x0e, 0x2d, 0x9f, 0x65, 0xfa, 0x89,
		0x5f, 0x9c, 0x67, 0xb3, 0x2c, 0xe1, 0xba, 0xf0,
		0xb0, 0xd8, 0xa4, 0x35, 0x05, 0xa0, 0x00, 0xa1,
		0xa9, 0xe0, 0x90, 0xd4, 0x14, 0xdb, 0x40, 0x4d,
	}

	if !bytes.Equal(key, expected) {
		t.Errorf("signing key mismatch:\n got: %x\nwant: %x", key, expected)
	}
}

func TestGenerateCredentials(t *testing.T) {
	cred, err := GenerateCredentials("testuser")
	if err != nil {
		t.Fatalf("GenerateCredentials: %v", err)
	}
	if len(cred.AccessKey) < 20 {
		t.Errorf("access key too short: %d chars", len(cred.AccessKey))
	}
	if !strings.HasPrefix(cred.AccessKey, "STNP") {
		t.Errorf("access key should start with STNP: %s", cred.AccessKey)
	}
	if len(cred.SecretKey) != 32 {
		t.Errorf("secret key should be 32 chars hex, got %d", len(cred.SecretKey))
	}
	if cred.UserID != "testuser" {
		t.Errorf("userID = %q, want testuser", cred.UserID)
	}
}

func TestSha256Hex(t *testing.T) {
	if got := sha256Hex(""); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("sha256Hex of empty string: got %s", got)
	}
}

func TestSplitAuthComponents(t *testing.T) {
	input := "Credential=abc, SignedHeaders=host;x-amz-date, Signature=123"
	parts := splitAuthComponents(input)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %v", len(parts), parts)
	}
}
