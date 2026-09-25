package jwt

import (
	"testing"
	"time"
)

// Time tests use fixed clocks — never time.Now() — so they're deterministic.

func TestCheckTime(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	mk := func(payload string) Token {
		// Build via Parse of hand-made segments for realism.
		tok, err := Parse("eyJhbGciOiJIUzI1NiJ9." + payload + ".sig")
		if err != nil {
			t.Fatalf("setup Parse: %v", err)
		}
		return tok
	}
	// exp in future -> valid; exp in past -> expired; nbf future -> not yet.
	_ = mk
	// Use direct maps for clarity (avoids hand-encoding base64 in test).
	valid := Token{Payload: map[string]any{"exp": float64(now.Unix() + 3600)}}
	if st, _ := valid.CheckTime(now, 0); st != TemporalValid {
		t.Errorf("valid: got %v", st)
	}
	expired := Token{Payload: map[string]any{"exp": float64(now.Unix() - 10)}}
	if st, msg := expired.CheckTime(now, 0); st != TemporalExpired || msg == "" {
		t.Errorf("expired: got %v %q", st, msg)
	}
	// Leeway forgives 10s skew with 30s tolerance.
	if st, _ := expired.CheckTime(now, 30*time.Second); st != TemporalValid {
		t.Errorf("leeway: got %v want valid", st)
	}
	future := Token{Payload: map[string]any{"nbf": float64(now.Unix() + 600)}}
	if st, _ := future.CheckTime(now, 0); st != TemporalNotYetValid {
		t.Errorf("nbf: got %v", st)
	}
	noExp := Token{Payload: map[string]any{}}
	if st, _ := noExp.CheckTime(now, 0); st != TemporalNoExp {
		t.Errorf("noExp: got %v", st)
	}
}

func TestHumanize(t *testing.T) {
	for _, tc := range []struct {
		sec  int64
		want string
	}{
		{5, "5s"}, {90, "1m"}, {7200, "2h"}, {90000, "1d"},
	} {
		if got := Humanize(tc.sec); got != tc.want {
			t.Errorf("Humanize(%d)=%q want %q", tc.sec, got, tc.want)
		}
	}
}

func TestFormatUnix(t *testing.T) {
	if got := FormatUnix(0); got != "1970-01-01T00:00:00Z" {
		t.Errorf("got %q", got)
	}
	if got := FormatUnix(-1); got != "invalid time" {
		t.Errorf("negative: got %q", got)
	}
}

func BenchmarkParse(b *testing.B) {
	const tok = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Parse(tok); err != nil {
			b.Fatal(err)
		}
	}
}
