package input

import (
	"encoding/base64"
	"strings"
)

// clean.go normalizes one user-supplied string into a JWT candidate.
//
// Real pastes are messy: "Bearer eyJ...", '"eyJ..."', whitespace,
// newlines from copy-paste. We strip all of that patiently instead of
// erroring on trivia — a decoder should be forgiving on input,
// strict on structure (that strictness lives in jwt.Parse).

// Normalize cleans s and reports whether anything remains.
// It never fails: empty stays empty, garbage passes through so
// jwt.Parse can produce the precise "here's what's wrong" error.
func Normalize(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	// Strip "Bearer " case-insensitively (copied from Authorization headers).
	// We check prefix on the trimmed string only — cheap and predictable.
	if len(s) > 7 && strings.EqualFold(s[:7], "Bearer ") {
		s = strings.TrimSpace(s[7:])
	}
	// Strip surrounding quotes from shell/JSON copy-paste.
	s = strings.Trim(s, `"'`+"`")
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	// Double-encoded tokens (base64 of the whole JWT) contain no dots,
	// so they can't be confused with a real JWT. Only then do we try
	// unwrapping — this avoids corrupting a valid token that happens
	// to also be valid StdEncoding (the old code's bug).
	if !strings.Contains(s, ".") {
		if unwrapped, ok := tryUnwrap(s); ok {
			return unwrapped, true
		}
	}
	return s, true
}

// tryUnwrap attempts standard-base64 decode and keeps the result only
// if it looks like a JWT (exactly 2 dots). Both padded and unpadded
// variants are tried because transport encoders differ.
func tryUnwrap(s string) (string, bool) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding} {
		// Guard length before allocating: base64 output is ~4/3 input.
		if len(s) > 64*1024 {
			return "", false
		}
		decoded, err := enc.DecodeString(s)
		if err != nil {
			continue
		}
		candidate := strings.TrimSpace(string(decoded))
		// Must look like header.payload.signature to be accepted.
		// We don't fully validate here — jwt.Parse owns that —
		// just enough to avoid false positives.
		if strings.Count(candidate, ".") == 2 && len(candidate) < 64*1024 {
			return candidate, true
		}
	}
	return "", false
}
