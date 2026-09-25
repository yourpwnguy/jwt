package jwt

import (
	"errors"
	"strings"
	"testing"
)

// Well-known jwt.io example: HS256, secret "your-256-bit-secret".
const exampleHS = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

func TestParseValid(t *testing.T) {
	tok, err := Parse(exampleHS)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if tok.Alg() != "HS256" {
		t.Errorf("Alg=%q want HS256", tok.Alg())
	}
	if sub, ok := tok.Claim("sub"); !ok || sub != "1234567890" {
		t.Errorf("sub=%v want 1234567890", sub)
	}
	if !strings.Contains(tok.SigningInput, ".") {
		t.Error("SigningInput should contain dot")
	}
}

func TestParseBadFormat(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		wantErr   error
	}{
		{"empty", "", ErrInvalidFormat},
		{"one part", "abc", ErrInvalidFormat},
		{"two parts", "a.b", ErrInvalidFormat},
		{"jwe", "a.b.c.d.e", ErrInvalidFormat},
		{"bad header", "!!!.e30.e30", ErrDecodeHeader},
		{"bad payload", "e30.!!!.e30", ErrDecodePayload},
		{"null header", "bnVsbA.e30.e30", ErrDecodeHeader}, // "null"
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Parse(tc.raw); !errors.Is(err, tc.wantErr) {
				t.Errorf("err=%v want %v", err, tc.wantErr)
			}
		})
	}
}

func TestParseTooLarge(t *testing.T) {
	big := strings.Repeat("a", MaxTokenSize+1)
	if _, err := Parse(big); !errors.Is(err, ErrTooLarge) {
		t.Errorf("err=%v want ErrTooLarge", err)
	}
}

func TestClaimDotPath(t *testing.T) {
	tok, _ := Parse(exampleHS)
	// Nested miss returns false, never panics.
	if _, ok := tok.Claim("user.role"); ok {
		t.Error("expected miss for user.role")
	}
	if _, ok := tok.Claim(""); ok {
		t.Error("expected miss for empty path")
	}
}

func TestUnixCoercion(t *testing.T) {
	// float64 (encoding/json default), int, and string forms all work.
	cases := []struct {
		val  any
		want int64
		ok   bool
	}{
		{float64(1716239022), 1716239022, true},
		{int(42), 42, true},
		{int64(99), 99, true},
		{"1716239022", 1716239022, true},
		{"not-a-number", 0, false},
		{nil, 0, false},
		{true, 0, false},
	}
	for _, c := range cases {
		got, ok := unixClaim(map[string]any{"exp": c.val}, "exp")
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("val=%v got=%d,%v want=%d,%v", c.val, got, ok, c.want, c.ok)
		}
	}
}
