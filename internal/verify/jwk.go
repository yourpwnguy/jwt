package verify

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// jwk.go converts a single JWK (RFC 7517) to a stdlib public key.
// Only verification fields are modeled: RSA (n/e), EC (crv/x/y),
// OKP Ed25519 (x). Private halves (d) are ignored on purpose.

// jwk is the verification subset of a JSON Web Key.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Crv string `json:"crv"`
	N   string `json:"n"`
	E   string `json:"e"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

// parseJWK handles a bare {"kty":"RSA",...} object.
func parseJWK(data []byte) (crypto.PublicKey, error) {
	var k jwk
	if err := json.Unmarshal(data, &k); err != nil {
		return nil, err
	}
	if k.Kty == "" {
		return nil, fmt.Errorf("not a JWK (missing kty)")
	}
	return k.publicKey()
}

// publicKey converts fields to stdlib keys. Unknown types error
// instead of returning nil (which would panic in verify).
func (k jwk) publicKey() (crypto.PublicKey, error) {
	switch k.Kty {
	case "RSA":
		n, err := b64Big(k.N)
		if err != nil {
			return nil, fmt.Errorf("invalid RSA n: %w", err)
		}
		e, err := b64Int(k.E)
		if err != nil {
			return nil, fmt.Errorf("invalid RSA e: %w", err)
		}
		return &rsa.PublicKey{N: n, E: e}, nil
	case "EC":
		return ecPublicKey(k)
	case "OKP":
		if k.Crv != "Ed25519" {
			return nil, fmt.Errorf("unsupported OKP curve %q (only Ed25519)", k.Crv)
		}
		x, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil || len(x) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid Ed25519 x")
		}
		return ed25519.PublicKey(x), nil
	default:
		return nil, fmt.Errorf("unsupported JWK kty %q", k.Kty)
	}
}

func ecPublicKey(k jwk) (crypto.PublicKey, error) {
	var curve elliptic.Curve
	switch k.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve %q", k.Crv)
	}
	x, err := b64Big(k.X)
	if err != nil {
		return nil, fmt.Errorf("invalid EC x: %w", err)
	}
	y, err := b64Big(k.Y)
	if err != nil {
		return nil, fmt.Errorf("invalid EC y: %w", err)
	}
	return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, nil
}

func b64Big(s string) (*big.Int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(b), nil
}

func b64Int(s string) (int, error) {
	n, err := b64Big(s)
	if err != nil {
		return 0, err
	}
	if !n.IsInt64() {
		return 0, fmt.Errorf("exponent too large")
	}
	return int(n.Int64()), nil
}
