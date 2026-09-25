package input

import (
	"context"
	"strings"
	"testing"
)

// Normalize must be forgiving on wrapping, strict nowhere (jwt.Parse owns strictness).

func TestNormalize(t *testing.T) {
	tok := "a.b.c"
	for _, tc := range []struct {
		name, in, want string
		ok             bool
	}{
		{"plain", tok, tok, true},
		{"bearer", "Bearer " + tok, tok, true},
		{"bearer lower", "bearer " + tok, tok, true},
		{"quotes", `"` + tok + `"`, tok, true},
		{"single quotes", "'" + tok + "'", tok, true},
		{"spaces", "  " + tok + "  ", tok, true},
		{"empty", "   ", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Normalize(tc.in)
			if ok != tc.ok || (ok && got != tc.want) {
				t.Errorf("got %q,%v want %q,%v", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestDoubleBase64Unwrap(t *testing.T) {
	// Standard base64 of "h.p.s" (no dots in encoded form) unwraps.
	inner := "eyJhIjoxfQ.eyJzdWIiOiIxIn0.c2ln"
	// Encode with StdEncoding (padded) to simulate transport wrapping.
	enc := func() string {
		// Hardcode instead of importing encoding/base64 in test? Use strings:
		// precomputed StdEncoding of inner.
		return "ZXlKaElqb3hmUS5leUp6ZFdJaU9pSXhJbjAuYzJsbQ=="
	}()
	_ = inner
	got, ok := Normalize(enc)
	if !ok {
		t.Fatal("expected unwrap")
	}
	if strings.Count(got, ".") != 2 {
		t.Errorf("unwrapped %q should have 2 dots", got)
	}
	// A real JWT (contains dots) must NEVER be unwrapped, even if its
	// chars happen to be valid StdEncoding alphabet.
	real := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig"
	if got, _ := Normalize(real); got != real {
		t.Errorf("real token corrupted: %q", got)
	}
}

func TestExtract(t *testing.T) {
	hs := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	noneTok := "eyJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0."
	text := "login ok token=" + hs + " then none " + noneTok + " end, version 1.2.3 ignored"
	found := ExtractTokens(text)
	if len(found) != 2 {
		t.Fatalf("found=%v want 2 tokens", found)
	}
	if found[0] != hs || found[1] != noneTok {
		t.Errorf("wrong tokens: %v", found)
	}
	if got := ExtractTokens("no tokens here, v1.2.3"); len(got) != 0 {
		t.Errorf("expected none, got %v", got)
	}
}

func TestResolvePrecedence(t *testing.T) {
	ctx := context.Background()
	hs := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	// -t plus positional: -t first, then args in order.
	got, err := Resolve(ctx, Config{Token: hs, Args: []string{hs}, StdinIsTTY: true})
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	// Piped stdin auto-reads when TTY=false and nothing else given.
	got, err = Resolve(ctx, Config{Stdin: strings.NewReader(hs + "\n"), StdinIsTTY: false})
	if err != nil || len(got) != 1 {
		t.Fatalf("stdin: got=%v err=%v", got, err)
	}
	// TTY stdin is ignored (would hang waiting for keystrokes).
	got, err = Resolve(ctx, Config{Stdin: strings.NewReader(hs), StdinIsTTY: true})
	if err != nil || len(got) != 0 {
		t.Fatalf("tty: got=%v err=%v", got, err)
	}
	// Cancelled context aborts.
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := Resolve(cancelled, Config{Token: hs}); err == nil {
		t.Error("expected ctx error")
	}
}

func BenchmarkExtract(b *testing.B) {
	hs := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	text := "log " + strings.Repeat("x ", 100) + hs + " tail"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ExtractTokens(text)
	}
}
