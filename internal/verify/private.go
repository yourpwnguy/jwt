package verify

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
)

// private.go converts private keys to public for verification.
// Accepting private files is deliberate UX: people point --pubkey at
// whatever they have, and verification only needs the public half.

// parsePrivate handles PKCS8, PKCS1, SEC1, and Ed25519 seeds.
func parsePrivate(der []byte, typ string) (crypto.PrivateKey, error) {
	if typ == "PRIVATE KEY" {
		// PKCS8 covers RSA/EC/Ed25519 in one envelope.
		if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
			return k, nil
		}
	}
	if typ == "RSA PRIVATE KEY" {
		if k, err := x509.ParsePKCS1PrivateKey(der); err == nil {
			return k, nil
		}
	}
	if typ == "EC PRIVATE KEY" {
		if k, err := x509.ParseECPrivateKey(der); err == nil {
			return k, nil
		}
	}
	// Fallbacks: try everything PKCS8-style before giving up.
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		return k, nil
	}
	if k, err := x509.ParseECPrivateKey(der); err == nil {
		return k, nil
	}
	return nil, fmt.Errorf("unrecognized private key (%s)", typ)
}

// publicFromPrivate extracts the public half. Unsupported types error
// instead of returning nil — a nil key would panic deep in crypto.
func publicFromPrivate(priv crypto.PrivateKey) (crypto.PublicKey, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey), nil
	default:
		return nil, fmt.Errorf("unsupported private key type %T", priv)
	}
}
