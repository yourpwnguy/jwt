package verify

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
	"github.com/yourpwnguy/jwt/v2/internal/secret"
)

// hmac.go verifies HS256/384/512. Small file on purpose: HMAC logic
// is three lines, the rest is security boundaries around secrets.

// loadHMACSecret resolves --secret into bytes and adds CLI-specific
// hints to the shared loader's refusals. The TUI uses the same loader
// without the hints because it has no --pubkey/--jwks flags.
func loadHMACSecret(spec string) ([]byte, error) {
	raw, err := secret.LoadHMAC(spec)
	if err == nil {
		return raw, nil
	}
	if errors.Is(err, secret.ErrEmpty) {
		return nil, fmt.Errorf("%w: provide --secret", err)
	}
	if errors.Is(err, secret.ErrPEMKey) || errors.Is(err, secret.ErrCertificate) {
		return nil, fmt.Errorf("%w; use --pubkey/--jwks for RSA tokens", err)
	}
	return nil, err
}

// verifyHMAC recomputes the MAC over tok.SigningInput and compares.
func verifyHMAC(tok jwt.Token, alg string, secret []byte) (bool, error) {
	var h func() hash.Hash
	switch alg {
	case "HS256":
		h = sha256.New
	case "HS384":
		h = sha512.New384
	case "HS512":
		h = sha512.New
	default:
		return false, fmt.Errorf("unknown HMAC alg %q", alg)
	}
	mac := hmac.New(h, secret)
	mac.Write([]byte(tok.SigningInput))
	expected := mac.Sum(nil)
	sig, err := decodeSignature(tok)
	if err != nil {
		return false, err
	}
	return subtleEqual(expected, sig), nil
}
