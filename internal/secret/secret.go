// Package secret loads HMAC secrets from literals or files.
//
// The important part is what it refuses. A common JWT attack is to take
// a service's RSA public key, use it as an HMAC secret, and sign an
// HS256 token with it. The service then verifies the HMAC using the
// public key and accepts a token anyone could forge. This loader
// rejects anything that looks like PEM key material or a certificate,
// so verify and the TUI cannot accidentally honor that trick.
package secret

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Sentinel errors let callers add their own context. The CLI says
// "use --pubkey/--jwks"; the TUI just prints the generic reason.
var (
	// ErrEmpty means the secret was blank after loading and trimming.
	ErrEmpty = errors.New("empty HMAC secret")
	// ErrPEMKey means the secret looks like PEM key material.
	ErrPEMKey = errors.New("secret looks like a PEM key (RS256 to HS256 confusion attack)")
	// ErrCertificate means the secret looks like an X.509 certificate.
	ErrCertificate = errors.New("secret looks like a certificate")
)

// LoadHMAC resolves spec into raw secret bytes.
//
// Accepted forms:
//   - "literal"       use the bytes as-is
//   - "raw:literal"   force literal, even if a file with that name exists
//   - "@path"         read the file at path
//   - "path"          read the file if it exists, else treat as literal
//
// Text files are trimmed of trailing CR/LF because `echo secret > file`
// is the common way people create them. Binary files are left alone.
func LoadHMAC(spec string) ([]byte, error) {
	if spec == "" {
		return nil, ErrEmpty
	}

	raw := spec
	if strings.HasPrefix(spec, "raw:") {
		raw = strings.TrimPrefix(spec, "raw:")
	} else if strings.HasPrefix(spec, "@") || isExistingFile(spec) {
		path := strings.TrimPrefix(spec, "@")
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
		return nil, ErrEmpty
	}
	// Check certificates before the broader KEY check so the error is
	// specific. Both end in a refusal.
	if strings.Contains(raw, "BEGIN CERTIFICATE") {
		return nil, ErrCertificate
	}
	if strings.Contains(raw, "BEGIN ") && strings.Contains(raw, "KEY") {
		return nil, ErrPEMKey
	}
	return []byte(raw), nil
}

// isExistingFile reports whether path exists and is not a directory.
// It lets bare file paths work without mistaking secrets for files.
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
