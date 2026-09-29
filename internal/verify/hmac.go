package verify

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"os"
	"strings"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// hmac.go verifies HS256/384/512. Small file on purpose: HMAC logic
// is three lines, the rest is security boundaries around secrets.

// loadHMACSecret resolves --secret into bytes.
// "@path" loads from file (trailing newline trimmed for text secrets);
// "raw:" forces literal; bare values are literal. Empty is rejected —
// an empty HMAC key is public knowledge, not a secret.
func loadHMACSecret(secret string) ([]byte, error) {
	if secret == "" {
		return nil, fmt.Errorf("empty HMAC secret: provide --secret")
	}
	raw := secret
	if strings.HasPrefix(secret, "raw:") {
		raw = strings.TrimPrefix(secret, "raw:")
	} else if strings.HasPrefix(secret, "@") || isExistingFile(secret) {
		path := strings.TrimPrefix(secret, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read secret file: %w", err)
		}
		raw = string(data)
		// Text secrets from echo often carry "\n"; binary keys must not
		// be trimmed, so only trim printable ASCII with newline endings.
		if isPrintableASCII(raw) {
			raw = strings.TrimRight(raw, "\r\n")
		}
	}
	if raw == "" {
		return nil, fmt.Errorf("empty HMAC secret: provide --secret")
	}
	// Key-confusion guard: a PEM block is a PUBLIC key. Using it as an
	// HMAC secret would let anyone forge tokens. Fail loudly.
	if strings.Contains(raw, "BEGIN ") && strings.Contains(raw, "KEY") {
		return nil, fmt.Errorf("secret looks like a PEM public key, refusing (RS256 to HS256 confusion attack); use --pubkey/--jwks for RSA tokens")
	}
	if strings.Contains(raw, "BEGIN CERTIFICATE") {
		return nil, fmt.Errorf("secret looks like a certificate, refusing; use --pubkey/--jwks for RSA tokens")
	}
	return []byte(raw), nil
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

// isExistingFile reports whether path exists and is not a directory.
// It keeps "@"-less file paths working without mistaking secrets for files.
func isExistingFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

// isPrintableASCII allows safe newline-trimming only for text secrets.
func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\t' || c == '\r' || c == '\n' {
			continue
		}
		if c < 0x20 || c > 0x7e {
			return false
		}
	}
	return true
}
