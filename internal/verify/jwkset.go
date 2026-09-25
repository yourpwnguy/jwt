package verify

import (
	"crypto"
	"encoding/json"
	"fmt"
)

// jwkset.go handles {"keys":[...]} sets and kid selection.
// Split from jwk.go (single-key conversion) because selection has its
// own security rule: unknown kids error, never fall back silently.

// jwks is a JSON Web Key Set.
type jwks struct {
	Keys []jwk `json:"keys"`
}

// Select picks by kid, or the first key when kid is empty (single-key
// file case). Unknown kids error — using the wrong key would be a hole.
func (s jwks) Select(kid string) (crypto.PublicKey, error) {
	if len(s.Keys) == 0 {
		return nil, fmt.Errorf("empty JWK Set")
	}
	if kid == "" {
		return s.Keys[0].publicKey()
	}
	for _, k := range s.Keys {
		if k.Kid == kid {
			return k.publicKey()
		}
	}
	return nil, fmt.Errorf("kid %q not found in JWK Set (%d keys)", kid, len(s.Keys))
}

// parseJWKS handles {"keys":[...]}.
func parseJWKS(data []byte) (jwks, error) {
	var s jwks
	if err := json.Unmarshal(data, &s); err != nil {
		return jwks{}, err
	}
	if len(s.Keys) == 0 {
		return jwks{}, fmt.Errorf("not a JWKS (no keys)")
	}
	return s, nil
}
