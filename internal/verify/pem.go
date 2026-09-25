package verify

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

// pem.go parses PEM/DER/certs. Split from keys.go because PEM has its
// own quirks (encrypted blocks, SSH pastes, cert envelopes) worth
// isolating and testing separately.

// parsePEMOrDER handles PKIX/PKCS1/EC/Ed25519 PEM, X.509 certs, and raw DER.
// Encrypted PEMs are rejected clearly; SSH keys get a conversion hint.
func parsePEMOrDER(data []byte) (crypto.PublicKey, error) {
	// Raw DER (no armor) — try PKIX, then cert.
	// min is builtin since Go 1.21, no helper needed.
	if !strings.Contains(string(data[:min(len(data), 64)]), "BEGIN") {
		if pub, err := x509.ParsePKIXPublicKey(data); err == nil {
			return pub, nil
		}
		if cert, err := x509.ParseCertificate(data); err == nil {
			return cert.PublicKey, nil
		}
	}
	block, _ := pem.Decode(data)
	if block == nil {
		if strings.Contains(string(data), "ssh-") {
			return nil, fmt.Errorf("SSH public key detected, convert with: ssh-keygen -e -m PKCS8 -f <file>")
		}
		return nil, fmt.Errorf("unrecognized key format (want PEM, DER, cert, or JWK)")
	}
	if strings.Contains(block.Type, "ENCRYPTED") {
		return nil, fmt.Errorf("encrypted private keys are not supported")
	}
	switch block.Type {
	case "PUBLIC KEY":
		return x509.ParsePKIXPublicKey(block.Bytes)
	case "CERTIFICATE":
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid certificate: %w", err)
		}
		return cert.PublicKey, nil
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "EC PUBLIC KEY":
		return x509.ParsePKIXPublicKey(block.Bytes)
	case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY", "OPENSSH PRIVATE KEY":
		priv, err := parsePrivate(block.Bytes, block.Type)
		if err != nil {
			return nil, err
		}
		return publicFromPrivate(priv)
	default:
		if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
			return pub, nil
		}
		return nil, fmt.Errorf("unsupported PEM type %q", block.Type)
	}
}
