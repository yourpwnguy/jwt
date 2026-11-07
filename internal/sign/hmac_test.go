package sign

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
	"github.com/yourpwnguy/jwt/v2/internal/verify"
)

// sign_test.go checks the output against a known vector and then runs
// the result through the real verifier. A signature that only agrees
// with itself is not evidence.

func TestHMACKnownVector(t *testing.T) {
	input := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ"
	const want = "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	got, err := HMAC(input, "HS256", []byte("your-256-bit-secret"))
	if err != nil {
		t.Fatalf("HMAC: %v", err)
	}
	if got != want {
		t.Fatalf("HMAC() = %q, want %q", got, want)
	}
}

func TestHMACRoundTripThroughVerify(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		alg    string
		secret string
	}{
		{"HS256", "s3cr3t"},
		{"HS384", "s3cr3t"},
		{"HS512", "s3cr3t"},
	}

	for _, tt := range tests {
		t.Run(tt.alg, func(t *testing.T) {
			header := []byte(`{"alg":"` + tt.alg + `"}`)
			payload := []byte(`{"sub":"1"}`)
			input := jwt.SigningInput(header, payload)

			sig, err := HMAC(input, tt.alg, []byte(tt.secret))
			if err != nil {
				t.Fatalf("HMAC: %v", err)
			}
			raw := jwt.Encode(header, payload, sig)
			tok, err := jwt.Parse(raw)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			res, err := verify.Verify(ctx, tok, verify.Options{Secret: tt.secret})
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if !res.SignatureValid {
				t.Fatal("Verify reported an invalid signature")
			}
		})
	}
}

func TestHMACSignatureShapes(t *testing.T) {
	tests := []struct {
		alg  string
		size int
	}{
		{"HS256", 32},
		{"HS384", 48},
		{"HS512", 64},
	}
	for _, tt := range tests {
		t.Run(tt.alg, func(t *testing.T) {
			sig, err := HMAC("a.b", tt.alg, []byte("secret"))
			if err != nil {
				t.Fatalf("HMAC: %v", err)
			}
			raw, err := base64.RawURLEncoding.DecodeString(sig)
			if err != nil {
				t.Fatalf("signature is not base64url: %v", err)
			}
			if len(raw) != tt.size {
				t.Fatalf("%s signature is %d bytes, want %d", tt.alg, len(raw), tt.size)
			}
		})
	}
}

func TestHMACErrors(t *testing.T) {
	if _, err := HMAC("a.b", "HS256", nil); err == nil {
		t.Fatal("empty secret must fail")
	}
	if _, err := HMAC("a.b", "RS256", []byte("secret")); err == nil {
		t.Fatal("non-HMAC alg must fail")
	}
}

func TestHMACDeterministic(t *testing.T) {
	first, err := HMAC("a.b", "HS256", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := HMAC("a.b", "HS256", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("same input and key produced different signatures: %q vs %q", first, second)
	}
	if len(first) < 10 {
		t.Fatalf("unexpected signature shape: %q", first)
	}
}
