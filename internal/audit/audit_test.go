package audit

import (
	"context"
	"testing"
	"time"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// Helpers build tokens from raw JWT strings (audit never checks
// signatures, so placeholder sigs are fine).

func fixture(t *testing.T, raw string) jwt.Token {
	t.Helper()
	tok, err := jwt.Parse(raw)
	if err != nil {
		t.Fatalf("fixture Parse: %v", err)
	}
	return tok
}

func TestNoneAlgCritical(t *testing.T) {
	none := fixture(t, "eyJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.")
	rep := Run(context.Background(), none, time.Now())
	var found Finding
	for _, f := range rep.Findings {
		if f.ID == "none-alg" {
			found = f
		}
	}
	if found.Pass || found.Severity != SeverityCritical {
		t.Errorf("none-alg: %+v", found)
	}
	if !rep.FailedAtOrAbove(SeverityHigh) {
		t.Error("should breach high threshold")
	}
}

func TestMissingExpHigh(t *testing.T) {
	// jwt.io HS256 example has no exp.
	tok := fixture(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	rep := Run(context.Background(), tok, time.Now())
	for _, f := range rep.Findings {
		if f.ID == "missing-exp" && f.Pass {
			t.Error("missing-exp should fail")
		}
	}
}

func TestSensitiveAndKid(t *testing.T) {
	// Payload {"password":"x"} + header kid. Craft via jwt.Parse of
	// precomputed segments: header {"alg":"HS256","kid":"1"}, payload {"password":"x"}.
	// base64url: header=eyJhbGciOiJIUzI1NiIsImtpZCI6IjEifQ, payload=eyJwYXNzd29yZCI6IngifQ
	tok := fixture(t, "eyJhbGciOiJIUzI1NiIsImtpZCI6IjEifQ.eyJwYXNzd29yZCI6IngifQ.c2ln")
	rep := Run(context.Background(), tok, time.Now())
	byID := map[string]Finding{}
	for _, f := range rep.Findings {
		byID[f.ID] = f
	}
	if byID["kid"].Pass {
		t.Error("kid should fail (review)")
	}
	if byID["sensitive-claim"].Pass {
		t.Error("sensitive should fail")
	}
}

func TestFailOnThreshold(t *testing.T) {
	tok := fixture(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	rep := Run(context.Background(), tok, time.Now())
	// Only missing-exp (high) fails on this fixture.
	if !rep.FailedAtOrAbove(SeverityHigh) {
		t.Error("should breach high")
	}
	if rep.FailedAtOrAbove(SeverityCritical) {
		t.Error("should not breach critical")
	}
	if rep.FailedCount() != 1 {
		t.Errorf("FailedCount=%d want 1", rep.FailedCount())
	}
}

func BenchmarkAudit(b *testing.B) {
	tok, _ := jwt.Parse("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	ctx := context.Background()
	now := time.Now()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		Run(ctx, tok, now)
	}
}
