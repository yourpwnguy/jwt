package verify

import (
	"context"
	"crypto"
	"fmt"
	"os"
	"strings"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// keys.go resolves --pubkey/--jwks into a public key. --pubkey wins
// when both are set (explicit file beats set lookup). kid selection
// uses the token's kid, falling back to the first key.

// loadPublicKey resolves key sources. It takes ctx because JWKS may fetch.
func loadPublicKey(ctx context.Context, tok jwt.Token, opts Options) (crypto.PublicKey, error) {
	if opts.PubKeyPath != "" {
		return loadKeyFile(strings.TrimPrefix(opts.PubKeyPath, "@"))
	}
	if opts.JWKS != "" {
		set, err := loadJWKS(ctx, opts.JWKS)
		if err != nil {
			return nil, err
		}
		return set.Select(tok.Kid())
	}
	return nil, fmt.Errorf("alg %s needs asymmetric key: provide --pubkey or --jwks", tok.Alg())
}

// loadKeyFile tries JWK(S) JSON first, then PEM/DER.
// JSON first because a JWK is ASCII that PEM would reject with noise.
func loadKeyFile(path string) (crypto.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	if len(data) > 1<<20 {
		return nil, fmt.Errorf("key file too large")
	}
	if trimmed := strings.TrimSpace(string(data)); strings.HasPrefix(trimmed, "{") {
		if pub, err := parseJWK([]byte(trimmed)); err == nil {
			return pub, nil
		}
		if set, err := parseJWKS([]byte(trimmed)); err == nil {
			return set.Select("")
		}
	}
	return parsePEMOrDER(data)
}
