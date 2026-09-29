package verify

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// sig.go dispatches to the right primitive by alg. It owns the
// "which key for which alg" policy so callers can't mix them up.

// verifySignature selects HMAC vs. asymmetric verification from the
// token's alg. Mismatched key types produce explicit errors, not
// silent failures — e.g. an RS256 token with only --secret errors
// with "needs --pubkey/--jwks" instead of pretending HMAC.
func verifySignature(ctx context.Context, tok jwt.Token, opts Options) (bool, error) {
	alg := strings.ToUpper(tok.Alg())
	switch alg {
	case "HS256", "HS384", "HS512":
		secret, err := loadHMACSecret(opts.Secret)
		if err != nil {
			return false, err
		}
		return verifyHMAC(tok, alg, secret)
	case "RS256", "RS384", "RS512",
		"PS256", "PS384", "PS512",
		"ES256", "ES384", "ES512",
		"EDDSA", "ED25519":
		pub, err := loadPublicKey(ctx, tok, opts)
		if err != nil {
			return false, err
		}
		return verifyAsymmetric(tok, alg, pub)
	default:
		return false, fmt.Errorf("%w %q", ErrUnsupportedAlg, tok.Alg())
	}
}

// decodeSignature base64url-decodes the third segment. Empty signatures
// (alg=none with stripped sig) fail here with a clear message.
func decodeSignature(tok jwt.Token) ([]byte, error) {
	if tok.Signature == "" {
		return nil, fmt.Errorf("empty signature")
	}
	sig, err := base64.RawURLEncoding.DecodeString(tok.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}
	return sig, nil
}

// subtleEqual compares HMAC outputs in constant time. Early-exit
// comparison would leak prefix info via timing — boring but correct
// to always use ConstantTimeCompare for secrets.
func subtleEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
