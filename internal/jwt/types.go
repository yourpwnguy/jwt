// Package jwt is the core domain: what a JWT is and how to decode it.
//
// The package deliberately does NOT verify signatures or fetch keys.
// Decoding is safe and offline; trust decisions live in internal/verify
// and policy decisions live in internal/audit. That split keeps this
// package tiny, testable, and free of network or crypto side effects.
//
// Invariants:
//   - Parse expects a clean "header.payload.signature" string.
//     Trimming "Bearer " prefixes, quotes, and whitespace is the job
//     of internal/input, not here. If you hand Parse garbage, you get
//     a clear sentinel error back.
//   - Header and Payload are always JSON objects (map[string]any).
//     A JWT whose header/payload isn't an object is rejected — that's
//     never a valid token, just a confusing base64 string.
package jwt

import "errors"

// Sentinel errors so callers can use errors.Is instead of string matching.
// We keep the set small: format vs. header vs. payload is all a CLI
// needs to print a useful "your token is broken here" message.
var (
	// ErrInvalidFormat means the token didn't split into exactly 3 parts.
	// Common causes: pasting only payload, JWE (5 parts), or trailing newline issues.
	ErrInvalidFormat = errors.New("invalid token format")

	// ErrDecodeHeader means the first segment isn't valid base64url JSON object.
	ErrDecodeHeader = errors.New("failed to decode header")

	// ErrDecodePayload means the second segment isn't valid base64url JSON object.
	ErrDecodePayload = errors.New("failed to decode payload")

	// ErrTooLarge is a DoS guard: we refuse absurdly large tokens before allocating.
	ErrTooLarge = errors.New("token too large")
)

// Limits exist so a malicious log line can't OOM us.
// A real JWT is a few KB at most; 32KB is already generous.
const (
	// MaxTokenSize caps the whole "h.p.s" string.
	MaxTokenSize = 32 * 1024
	// MaxSegmentSize caps a single base64 segment before decoding.
	MaxSegmentSize = 16 * 1024
)

// Token is a decoded JWT.
//
// Header and Payload are maps so unknown/custom claims survive untouched —
// we never want to drop a claim just because we didn't anticipate it.
// Signature is kept as the raw base64url string (it's binary, not JSON).
// SigningInput is the exact "header_b64.payload_b64" bytes verifiers must
// re-sign; keeping it avoids re-encoding and subtle canonicalization bugs.
type Token struct {
	Header       map[string]any `json:"header"`
	Payload      map[string]any `json:"payload"`
	Signature    string         `json:"signature"`
	SigningInput string         `json:"-"`
	Raw          string         `json:"-"`
}
