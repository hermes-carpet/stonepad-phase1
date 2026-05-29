// Package relay implements Cloudflare R2 relay polling.
//
// The relay poller periodically fetches objects from an R2 bucket and
// synchronizes them with the local storage. This allows the home server
// to stay in sync when the Flutter app writes directly to R2 instead of
// connecting to the home server directly.
//
// See §9.5 of the Stonepad v1 Implementation Plan.
package relay

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/hermes-carpet/stonepad/server/internal/storage"
)

// Poller periodically fetches from an S3-compatible relay (Cloudflare R2)
// and syncs objects into local storage.
type Poller struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucket     string
	region     string
	interval   time.Duration
	store      storage.Storage
	httpClient *http.Client
	logger     *slog.Logger
	stopCh     chan struct{}
	doneCh     chan struct{}
}

// New creates a new relay Poller. Call Start() to begin the polling loop.
func New(
	endpoint, accessKey, secretKey, bucket, region string,
	interval time.Duration,
	store storage.Storage,
	logger *slog.Logger,
) *Poller {
	return &Poller{
		endpoint:  strings.TrimRight(endpoint, "/"),
		accessKey: accessKey,
		secretKey: secretKey,
		bucket:    bucket,
		region:    region,
		interval:  interval,
		store:     store,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start begins the background polling goroutine.
func (p *Poller) Start() {
	go p.pollLoop()
	p.logger.Info("relay: polling started",
		"endpoint", p.endpoint,
		"bucket", p.bucket,
		"interval_seconds", p.interval.Seconds(),
	)
}

// Stop signals the polling loop to stop and waits for it to exit.
func (p *Poller) Stop() {
	close(p.stopCh)
	<-p.doneCh
	p.logger.Info("relay: polling stopped")
}

// pollLoop is the background goroutine that fetches from R2 on a timer.
func (p *Poller) pollLoop() {
	defer close(p.doneCh)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Run immediately on start
	p.poll()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.poll()
		}
	}
}

// poll performs one full poll cycle:
//  1. List objects from R2
//  2. Pull new/changed objects from R2 → local
//  3. Push local changes not yet in R2
func (p *Poller) poll() {
	p.logger.Debug("relay: polling R2")

	// Step 1: list R2 objects
	remoteObjects, err := p.listObjects()
	if err != nil {
		p.logger.Error("relay: list objects failed", "error", err)
		return
	}

	// Step 2: pull new/changed objects
	pullCount := 0
	for key, remoteHash := range remoteObjects {
		localMeta, err := p.store.Head(context.Background(), key)
		if err != nil || localMeta.ContentHash != remoteHash {
			if err := p.pullObject(key); err != nil {
				p.logger.Error("relay: pull failed", "key", key, "error", err)
				continue
			}
			pullCount++
		}
	}

	// Step 3: push local changes to R2
	localMetas, err := p.store.List(context.Background(), "")
	if err != nil {
		p.logger.Error("relay: list local failed", "error", err)
		return
	}

	pushCount := 0
	for _, meta := range localMetas {
		if _, exists := remoteObjects[meta.Path]; !exists || remoteObjects[meta.Path] != meta.ContentHash {
			if err := p.pushObject(meta.Path); err != nil {
				p.logger.Error("relay: push failed", "key", meta.Path, "error", err)
				continue
			}
			pushCount++
		}
	}

	if pullCount > 0 || pushCount > 0 {
		p.logger.Info("relay: poll complete",
			"pulled", pullCount,
			"pushed", pushCount,
		)
	}
}

// ────────────────────── S3 HTTP ops (with SigV4 signing) ──────────────────────

// listObjects fetches all object keys and their ETags from the R2 bucket.
func (p *Poller) listObjects() (map[string]string, error) {
	result := make(map[string]string)
	var continuationToken string

	for {
		query := url.Values{}
		query.Set("list-type", "2")
		if continuationToken != "" {
			query.Set("continuation-token", continuationToken)
		}

		u := fmt.Sprintf("%s/%s?%s", p.endpoint, p.bucket, query.Encode())
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("building list request: %w", err)
		}

		p.signRequest(req, sha256Hex(nil))

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading list response: %w", err)
		}

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("list returned HTTP %d: %s", resp.StatusCode, string(body))
		}

		var lr listBucketResult
		if err := xml.Unmarshal(body, &lr); err != nil {
			return nil, fmt.Errorf("parsing list response: %w", err)
		}

		for _, obj := range lr.Contents {
			hash := strings.Trim(obj.ETag, "\"")
			result[obj.Key] = hash
		}

		if !lr.IsTruncated {
			break
		}
		continuationToken = lr.NextContinuationToken
	}

	return result, nil
}

// pullObject downloads an object from R2 and writes it to local storage.
func (p *Poller) pullObject(key string) error {
	u := fmt.Sprintf("%s/%s/%s", p.endpoint, p.bucket, url.PathEscape(key))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}

	p.signRequest(req, sha256Hex(nil))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull HTTP %d: %s", resp.StatusCode, string(body))
	}

	_, err = p.store.Put(context.Background(), key, resp.Body)
	return err
}

// pushObject uploads a local note to R2.
func (p *Poller) pushObject(key string) error {
	reader, err := p.store.Get(context.Background(), key)
	if err != nil {
		return err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	bodyHash := sha256Hex(data)

	u := fmt.Sprintf("%s/%s/%s", p.endpoint, p.bucket, url.PathEscape(key))
	req, err := http.NewRequest("PUT", u, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/markdown")
	p.signRequest(req, bodyHash)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("push HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ────────────────────── SigV4 Signing (client-side) ──────────────────────

// signRequest adds AWS Signature V4 headers to the request.
func (p *Poller) signRequest(req *http.Request, bodyHash string) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", bodyHash)

	canonicalHeaders, signedHeaders := p.buildCanonicalHeaders(req)
	canonicalReq := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		bodyHash,
	)
	canonicalReqHash := sha256Hex([]byte(canonicalReq))

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, p.region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate,
		credentialScope,
		canonicalReqHash,
	)

	signingKey := p.deriveSigningKey(dateStamp)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		p.accessKey,
		credentialScope,
		signedHeaders,
		signature,
	)
	req.Header.Set("Authorization", authHeader)
}

func (p *Poller) buildCanonicalHeaders(req *http.Request) (string, string) {
	var headers []string
	headerNames := []string{"host", "x-amz-content-sha256", "x-amz-date"}

	for _, name := range headerNames {
		val := req.Header.Get(name)
		if val == "" && name == "host" {
			val = req.URL.Host
		}
		headers = append(headers, fmt.Sprintf("%s:%s", name, strings.TrimSpace(val)))
	}

	sort.Strings(headers)
	return strings.Join(headers, "\n") + "\n", strings.Join(headerNames, ";")
}

func (p *Poller) deriveSigningKey(dateStamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+p.secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(p.region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

// ────────────────────── XML structs for ListObjectsV2 ──────────────────────

type listBucketResult struct {
	XMLName               xml.Name `xml:"ListBucketResult"`
	IsTruncated           bool     `xml:"IsTruncated"`
	NextContinuationToken string   `xml:"NextContinuationToken"`
	Contents              []object `xml:"Contents"`
}

type object struct {
	Key  string `xml:"Key"`
	ETag string `xml:"ETag"`
}

// ────────────────────── helpers ──────────────────────

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
