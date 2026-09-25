package verify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/yourpwnguy/jwt/internal/jwt"
)

// test helpers mint real tokens so verify tests exercise actual crypto,
// not fixtures that could drift from the implementation.

func b64urlJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func mintHMAC(t *testing.T, header, payload map[string]any, secret string) string {
	t.Helper()
	h := b64urlJSON(t, header)
	p := b64urlJSON(t, payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(h + "." + p))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return h + "." + p + "." + sig
}

func mustParseTok(t *testing.T, raw string) jwt.Token {
	t.Helper()
	tok, err := jwt.Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return tok
}

func TestHMACValidInvalid(t *testing.T) {
	ctx := context.Background()
	raw := mintHMAC(t, map[string]any{"alg": "HS256"}, map[string]any{"sub": "1"}, "s3cr3t")
	tok := mustParseTok(t, raw)
	res, err := Verify(ctx, tok, Options{Secret: "s3cr3t"})
	if err != nil || !res.SignatureValid {
		t.Errorf("valid: res=%+v err=%v", res, err)
	}
	res, err = Verify(ctx, tok, Options{Secret: "wrong"})
	if err != nil {
		t.Fatalf("wrong secret should not error, just invalid: %v", err)
	}
	if res.SignatureValid {
		t.Error("wrong secret should be invalid")
	}
}

func TestHMACGuards(t *testing.T) {
	ctx := context.Background()
	raw := mintHMAC(t, map[string]any{"alg": "HS256"}, map[string]any{"sub": "1"}, "s3cr3t")
	tok := mustParseTok(t, raw)
	if _, err := Verify(ctx, tok, Options{Secret: ""}); !errors.Is(err, ErrNoKey) {
		t.Errorf("empty secret: err=%v want ErrNoKey", err)
	}
	pemLike := "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----"
	if _, err := Verify(ctx, tok, Options{Secret: pemLike}); err == nil {
		t.Error("PEM as HMAC secret must be rejected")
	}
}

func TestNoneNeverVerifies(t *testing.T) {
	tok := mustParseTok(t, "eyJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.")
	if _, err := Verify(context.Background(), tok, Options{Secret: "x"}); !errors.Is(err, ErrNone) {
		t.Errorf("none: err=%v want ErrNone", err)
	}
}

func TestClaimsAudIssExp(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1_700_000_000, 0)
	// Valid exp + matching aud/iss.
	raw := mintHMAC(t, map[string]any{"alg": "HS256"},
		map[string]any{"sub": "1", "exp": now.Unix() + 3600, "aud": "api", "iss": "auth"}, "s")
	tok := mustParseTok(t, raw)
	res, err := Verify(ctx, tok, Options{Secret: "s", Aud: "api", Iss: "auth", Now: now})
	if err != nil || !res.ClaimsValid {
		t.Errorf("claims: %+v %v", res, err)
	}
	// Aud mismatch.
	res, _ = Verify(ctx, tok, Options{Secret: "s", Aud: "other", Now: now})
	if res.ClaimsValid {
		t.Error("aud mismatch should be invalid")
	}
	// Expired, no leeway.
	past := mintHMAC(t, map[string]any{"alg": "HS256"}, map[string]any{"exp": now.Unix() - 100}, "s")
	res, _ = Verify(ctx, mustParseTok(t, past), Options{Secret: "s", Now: now})
	if res.ClaimsValid {
		t.Error("expired should be invalid")
	}
	// Leeway forgives.
	res, _ = Verify(ctx, mustParseTok(t, past), Options{Secret: "s", Now: now, Leeway: 5 * time.Minute})
	if !res.ClaimsValid {
		t.Errorf("leeway should forgive: %+v", res)
	}
}

func BenchmarkHMACVerify(b *testing.B) {
	// Pre-mint once; benchmark the verify path only.
	data, _ := json.Marshal(map[string]any{"alg": "HS256"})
	h := base64.RawURLEncoding.EncodeToString(data)
	pdata, _ := json.Marshal(map[string]any{"sub": "1"})
	p := base64.RawURLEncoding.EncodeToString(pdata)
	mac := hmac.New(sha256.New, []byte("s3cr3t"))
	mac.Write([]byte(h + "." + p))
	raw := h + "." + p + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	tok, _ := jwt.Parse(raw)
	ctx := context.Background()
	opts := Options{Secret: "s3cr3t"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Verify(ctx, tok, opts); err != nil {
			b.Fatal(err)
		}
	}
}
