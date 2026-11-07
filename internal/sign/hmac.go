// Package sign computes JWS signatures for tokens being edited.
//
// It is the write side of internal/verify, not a trust decision. The
// caller decides which algorithm and key to use; this package only does
// the math and returns base64url text. Keeping it separate means the
// TUI never has to import verification logic just to save a token.
package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"strings"
)

// HMAC signs signingInput with HS256, HS384, or HS512.
//
// The secret must be non-empty. An empty HMAC key is public knowledge,
// not a secret, so refusing it here matches the verify side.
func HMAC(signingInput, alg string, secret []byte) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("empty HMAC secret")
	}

	var h func() hash.Hash
	switch strings.ToUpper(alg) {
	case "HS256":
		h = sha256.New
	case "HS384":
		h = sha512.New384
	case "HS512":
		h = sha512.New
	default:
		return "", fmt.Errorf("unsupported HMAC alg %q", alg)
	}

	mac := hmac.New(h, secret)
	mac.Write([]byte(signingInput))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
