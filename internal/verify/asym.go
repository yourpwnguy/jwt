package verify

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"math/big"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// asym.go verifies RSA/ECDSA/EdDSA/PS signatures with stdlib only.
//
// The subtle part is ECDSA: JWS uses raw R||S (fixed-width big-endian),
// while Go's ecdsa.VerifyASN1 wants DER. We split manually and call
// ecdsa.Verify — getting this wrong silently rejects every ES256 token,
// so it's covered by dedicated tests with RFC 7515 vectors.

// verifyAsymmetric dispatches by alg family. pub must be the matching
// public type; mismatches error instead of panicking.
func verifyAsymmetric(tok jwt.Token, alg string, pub crypto.PublicKey) (bool, error) {
	sig, err := decodeSignature(tok)
	if err != nil {
		return false, err
	}
	digest, h, err := hashSigningInput(tok.SigningInput, alg)
	if err != nil {
		return false, err
	}
	switch {
	case isRSA(alg):
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return false, fmt.Errorf("alg %s needs RSA public key, got %T", alg, pub)
		}
		if isPSS(alg) {
			opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto, Hash: h}
			if err := rsa.VerifyPSS(rsaPub, h, digest, sig, opts); err != nil {
				return false, nil
			}
			return true, nil
		}
		if err := rsa.VerifyPKCS1v15(rsaPub, h, digest, sig); err != nil {
			return false, nil
		}
		return true, nil
	case isECDSA(alg):
		ecPub, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return false, fmt.Errorf("alg %s needs ECDSA public key, got %T", alg, pub)
		}
		return verifyECDSARaw(ecPub, digest, sig), nil
	case isEdDSA(alg):
		edPub, ok := pub.(ed25519.PublicKey)
		if !ok {
			return false, fmt.Errorf("alg %s needs Ed25519 public key, got %T", alg, pub)
		}
		if len(sig) != ed25519.SignatureSize {
			return false, nil
		}
		return ed25519.Verify(edPub, []byte(tok.SigningInput), sig), nil
	default:
		return false, fmt.Errorf("unsupported asymmetric alg %q", alg)
	}
}

// hashSigningInput hashes "header.payload" per alg. EdDSA signs raw
// bytes (no pre-hash); we return nil digest for it and let the caller
// use SigningInput directly.
func hashSigningInput(input, alg string) ([]byte, crypto.Hash, error) {
	var h crypto.Hash
	var fn func() hash.Hash
	switch alg {
	case "RS256", "ES256", "PS256":
		h, fn = crypto.SHA256, sha256.New
	case "RS384", "ES384", "PS384":
		h, fn = crypto.SHA384, sha512.New384
	case "RS512", "ES512", "PS512":
		h, fn = crypto.SHA512, sha512.New
	case "EDDSA", "ED25519":
		return nil, 0, nil
	default:
		return nil, 0, fmt.Errorf("unknown hash for alg %q", alg)
	}
	digest := fn()
	digest.Write([]byte(input))
	return digest.Sum(nil), h, nil
}

// verifyECDSARaw splits JWS raw R||S and verifies. Length must be even
// and match the curve byte size; anything else is "invalid", not an error.
func verifyECDSARaw(pub *ecdsa.PublicKey, digest, sig []byte) bool {
	byteLen := (pub.Curve.Params().BitSize + 7) / 8
	if len(sig) != 2*byteLen {
		return false
	}
	r := new(big.Int).SetBytes(sig[:byteLen])
	s := new(big.Int).SetBytes(sig[byteLen:])
	return ecdsa.Verify(pub, digest, r, s)
}

func isRSA(alg string) bool { return len(alg) > 0 && (alg[0] == 'R' || alg[0] == 'P') }
func isPSS(alg string) bool { return len(alg) > 0 && alg[0] == 'P' }
func isECDSA(alg string) bool {
	return len(alg) > 1 && alg[0] == 'E' && alg != "EDDSA" && alg != "ED25519"
}
func isEdDSA(alg string) bool { return alg == "EDDSA" || alg == "ED25519" }
