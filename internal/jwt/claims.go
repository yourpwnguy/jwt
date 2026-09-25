package jwt

import (
	"encoding/json"
	"strconv"
	"strings"
)

// claims.go answers "what's inside the token?" without caring about trust.
//
// All helpers are nil-safe and never panic on weird JSON types —
// real-world tokens contain numbers as float64, ints, strings, even
// json.Number depending on how they were built. We coerce patiently
// instead of failing loudly.

// Alg returns the "alg" header, e.g. "HS256". Empty if missing.
func (t Token) Alg() string {
	v, _ := t.Header["alg"].(string)
	return v
}

// Kid returns the "kid" header if present.
func (t Token) Kid() string {
	v, _ := t.Header["kid"].(string)
	return v
}

// Typ returns the "typ" header, usually "JWT".
func (t Token) Typ() string {
	v, _ := t.Header["typ"].(string)
	return v
}

// Claim fetches a (possibly nested) payload claim via dot path,
// e.g. Claim("user.role") or Claim("exp").
//
// It returns (value, true) on success. Maps are traversed; anything
// else is a miss. This is intentionally tiny — no wildcards, no
// array indexing — because 99% of CLI use is "--claim sub" or
// "--claim exp". Full JSONPath would be enterprise bullshit here.
func (t Token) Claim(path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	var cur any = map[string]any(t.Payload)
	for _, key := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// unixClaim coerces a numeric date claim (exp/iat/nbf) to int64.
//
// JWT numeric dates are seconds since epoch, but JSON decoding gives
// us float64, and hand-built tokens sometimes use strings. We accept
// all sane representations rather than erroring on trivia.
func unixClaim(m map[string]any, key string) (int64, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(n.String(), 64); err == nil {
			return int64(f), true
		}
		return 0, false
	case string:
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return i, true
		}
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return int64(f), true
		}
		return 0, false
	default:
		return 0, false
	}
}

// Exp returns the "exp" claim as unix seconds, if present and numeric.
func (t Token) Exp() (int64, bool) { return unixClaim(t.Payload, "exp") }

// Iat returns the "iat" claim as unix seconds, if present and numeric.
func (t Token) Iat() (int64, bool) { return unixClaim(t.Payload, "iat") }

// Nbf returns the "nbf" claim as unix seconds, if present and numeric.
func (t Token) Nbf() (int64, bool) { return unixClaim(t.Payload, "nbf") }
