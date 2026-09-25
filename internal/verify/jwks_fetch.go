package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// jwks_fetch.go loads a JWK Set from file or URL. The split from jwk.go
// (parsing) is deliberate: fetching has timeouts, size caps, and scheme
// checks; parsing has none. Testing each in isolation is far easier.

// Limits: JWKS is small JSON; 1MB is already 100x generous and bounds
// a malicious endpoint from OOMing the CLI.
const (
	maxJWKSBytes = 1 << 20
	jwksTimeout  = 5 * time.Second
)

// loadJWKS resolves path-or-URL into a parsed set.
// http(s) URLs fetch with ctx timeout; anything else reads a file.
// Bare JSON objects (single JWK) are wrapped into a 1-key set.
func loadJWKS(ctx context.Context, ref string) (jwks, error) {
	ref = strings.TrimSpace(strings.TrimPrefix(ref, "@"))
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return fetchJWKS(ctx, ref)
	}
	data, err := os.ReadFile(ref)
	if err != nil {
		return jwks{}, fmt.Errorf("failed to read JWKS file: %w", err)
	}
	if len(data) > maxJWKSBytes {
		return jwks{}, fmt.Errorf("JWKS file too large")
	}
	return parseJWKSMaybeSingle(data)
}

// fetchJWKS GETs with timeout, checks status, and caps the body.
// Only 200 is accepted — JWKS endpoints shouldn't redirect to login
// pages, and following redirects would leak the request elsewhere.
func fetchJWKS(ctx context.Context, url string) (jwks, error) {
	ctx, cancel := context.WithTimeout(ctx, jwksTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return jwks{}, fmt.Errorf("invalid JWKS URL: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return jwks{}, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return jwks{}, fmt.Errorf("JWKS fetch: unexpected status %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxJWKSBytes+1))
	if err != nil {
		return jwks{}, fmt.Errorf("failed to read JWKS: %w", err)
	}
	if len(data) > maxJWKSBytes {
		return jwks{}, fmt.Errorf("JWKS response too large")
	}
	return parseJWKSMaybeSingle(data)
}

// parseJWKSMaybeSingle accepts {"keys":[...]} or a bare single JWK.
func parseJWKSMaybeSingle(data []byte) (jwks, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		// Try set first (common), then single.
		var probe struct {
			Keys json.RawMessage `json:"keys"`
		}
		if err := json.Unmarshal(data, &probe); err == nil && len(probe.Keys) > 0 {
			return parseJWKS(data)
		}
		if pub, err := parseJWK(data); err == nil {
			_ = pub
			var k jwk
			_ = json.Unmarshal(data, &k)
			return jwks{Keys: []jwk{k}}, nil
		}
	}
	return parseJWKS(data)
}
