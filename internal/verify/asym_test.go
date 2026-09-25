package verify

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

// asym_test.go covers RSA/ECDSA/EdDSA with freshly generated keys so
// tests never depend on checked-in fixtures that could expire or leak.

// signRSA returns a complete RS256 token.
func signRSA(t *testing.T, priv *rsa.PrivateKey, header, payload map[string]any) string {
	t.Helper()
	h := b64urlJSON(t, header)
	p := b64urlJSON(t, payload)
	digest := sha256.Sum256([]byte(h + "." + p))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func writePubPEM(t *testing.T, pub crypto.PublicKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pub.pem")
	data := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRSAValidInvalid(t *testing.T) {
	ctx := context.Background()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	raw := signRSA(t, priv, map[string]any{"alg": "RS256"}, map[string]any{"sub": "1"})
	tok := mustParseTok(t, raw)
	pubPath := writePubPEM(t, &priv.PublicKey)
	res, err := Verify(ctx, tok, Options{PubKeyPath: pubPath})
	if err != nil || !res.SignatureValid {
		t.Errorf("rsa valid: %+v %v", res, err)
	}
	// Wrong key invalid, not error.
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	res, err = Verify(ctx, tok, Options{PubKeyPath: writePubPEM(t, &other.PublicKey)})
	if err != nil || res.SignatureValid {
		t.Errorf("rsa wrong key: %+v %v", res, err)
	}
	// Private file also works (public extracted).
	der, _ := x509.MarshalPKCS8PrivateKey(priv)
	privPath := filepath.Join(t.TempDir(), "priv.pem")
	_ = os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600)
	res, err = Verify(ctx, tok, Options{PubKeyPath: privPath})
	if err != nil || !res.SignatureValid {
		t.Errorf("rsa via private: %+v %v", res, err)
	}
}

func TestECDSA(t *testing.T) {
	ctx := context.Background()
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	header, payload := map[string]any{"alg": "ES256"}, map[string]any{"sub": "1"}
	h := b64urlJSON(t, header)
	p := b64urlJSON(t, payload)
	digest := sha256.Sum256([]byte(h + "." + p))
	r, s, _ := ecdsa.Sign(rand.Reader, priv, digest[:])
	// JWS raw R||S, each 32 bytes for P-256.
	rb, sb := r.Bytes(), s.Bytes()
	raw := make([]byte, 64)
	copy(raw[32-len(rb):32], rb)
	copy(raw[64-len(sb):], sb)
	token := h + "." + p + "." + base64.RawURLEncoding.EncodeToString(raw)
	tok := mustParseTok(t, token)
	res, err := Verify(ctx, tok, Options{PubKeyPath: writePubPEM(t, &priv.PublicKey)})
	if err != nil || !res.SignatureValid {
		t.Errorf("ecdsa: %+v %v", res, err)
	}
}

func TestEdDSAAndJWKS(t *testing.T) {
	ctx := context.Background()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	h := b64urlJSON(t, map[string]any{"alg": "EdDSA", "kid": "k1"})
	p := b64urlJSON(t, map[string]any{"sub": "1"})
	sig := ed25519.Sign(priv, []byte(h+"."+p))
	token := h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig)
	tok := mustParseTok(t, token)
	// JWKS file with matching kid.
	jwksJSON, _ := json.Marshal(map[string]any{"keys": []any{map[string]any{
		"kty": "OKP", "crv": "Ed25519", "kid": "k1",
		"x": base64.RawURLEncoding.EncodeToString(pub),
	}}})
	jwksPath := filepath.Join(t.TempDir(), "jwks.json")
	_ = os.WriteFile(jwksPath, jwksJSON, 0o600)
	res, err := Verify(ctx, tok, Options{JWKS: jwksPath})
	if err != nil || !res.SignatureValid {
		t.Errorf("eddsa jwks: %+v %v", res, err)
	}
	// Unknown kid errors (never silently use wrong key).
	h2 := b64urlJSON(t, map[string]any{"alg": "EdDSA", "kid": "nope"})
	tok2 := mustParseTok(t, h2+"."+p+"."+base64.RawURLEncoding.EncodeToString(sig))
	if _, err := Verify(ctx, tok2, Options{JWKS: jwksPath}); err == nil {
		t.Error("unknown kid should error")
	}
}
