// Package verify answers "should I trust this token?" — signature plus
// claims. It is the only package that touches crypto secrets or network
// (JWKS fetch), so its boundaries are deliberately paranoid:
//
//   - alg=none is NEVER valid here, even with an empty secret.
//   - HMAC secrets that look like PEM public keys are rejected —
//     that's the classic RS256->HS256 confusion attack, and accepting
//     it would turn a public value into a signing key.
//   - Empty secrets are rejected. An empty HMAC key is known to everyone.
//   - All network fetches take ctx with timeout and size caps.
package verify

import (
	"time"
)

// Options selects how to verify. Zero value verifies nothing — callers
// must provide Secret, PubKeyPath, or JWKS. Aud/Iss are optional
// assertions; Leeway tolerates clock skew.
type Options struct {
	// Secret is a literal HMAC secret. Prefix "@path" loads from file.
	Secret string
	// PubKeyPath loads a PEM/DER cert or JWK from file. "@" optional.
	PubKeyPath string
	// JWKS is a file path or https URL for a JWK Set.
	JWKS string
	// Aud requires this audience (token aud may be string or list).
	Aud string
	// Iss requires this issuer.
	Iss string
	// Leeway tolerates exp/nbf skew. 0 is strict.
	Leeway time.Duration
	// Now anchors claim checks; zero means time.Now().
	Now time.Time
}

// Result separates the two trust questions so UIs can show both:
// "is the math right?" (SignatureValid) and "is it usable now, for me?"
// (ClaimsValid + Reason). A token can be correctly signed yet expired.
type Result struct {
	Alg            string `json:"alg"`
	SignatureValid bool   `json:"signatureValid"`
	ClaimsValid    bool   `json:"claimsValid"`
	ClaimsReason   string `json:"claimsReason,omitempty"`
}

// HasKey reports whether any key material was configured.
func (o Options) HasKey() bool {
	return o.Secret != "" || o.PubKeyPath != "" || o.JWKS != ""
}
