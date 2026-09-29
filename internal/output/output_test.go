package output

import (
	"strings"
	"testing"
	"time"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

func testToken(t *testing.T) jwt.Token {
	t.Helper()
	const hs = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	tok, err := jwt.Parse(hs)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func TestFormatOneModes(t *testing.T) {
	tok := testToken(t)
	now := time.Unix(1_700_000_000, 0)
	if s, err := FormatOne(tok, Options{Format: FormatJSON, Now: now, NoColor: true}); err != nil || !strings.Contains(s, `"sub"`) {
		t.Errorf("json: %v %s", err, s)
	}
	if s, err := FormatOne(tok, Options{Format: FormatCompact, Now: now, NoColor: true}); err != nil || strings.Contains(s, "\n") {
		t.Errorf("compact should be single line: %q", s)
	}
	if s, err := FormatOne(tok, Options{HeaderOnly: true, Now: now, NoColor: true}); err != nil || !strings.Contains(s, "HS256") || strings.Contains(s, "John") {
		t.Errorf("headerOnly: %s", s)
	}
	if s, err := FormatOne(tok, Options{Claim: "sub", Now: now}); err != nil || s != "1234567890" {
		t.Errorf("claim: got %q", s)
	}
	if _, err := FormatOne(tok, Options{Claim: "missing", Now: now}); err == nil {
		t.Error("expected claim miss error")
	}
}

func TestFormatOneQuietDropsFactsRow(t *testing.T) {
	tok := testToken(t)
	now := time.Unix(1_700_000_000, 0)
	loud, err := FormatOne(tok, Options{Format: FormatPretty, Now: now, NoColor: true})
	if err != nil || !strings.Contains(loud, "HS256") {
		t.Fatalf("pretty: %v %s", err, loud)
	}
	quiet, err := FormatOne(tok, Options{Format: FormatPretty, Quiet: true, Now: now, NoColor: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(quiet, "●") || strings.Contains(quiet, "sub 1234567890") {
		t.Errorf("quiet pretty should print just the data: %q", quiet)
	}
	if !strings.Contains(quiet, `"header"`) || !strings.Contains(quiet, `"signature"`) {
		t.Errorf("quiet pretty keeps the full JSON body: %q", quiet)
	}
}

func TestFormatBatchNumbering(t *testing.T) {
	tok := testToken(t)
	now := time.Unix(1_700_000_000, 0)
	two := []jwt.Token{tok, tok}
	s, err := FormatBatch(two, Options{Format: FormatJSON, Now: now, NoColor: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "Token 1") || !strings.Contains(s, "Token 2") {
		t.Errorf("batch should number: %s", s)
	}
	s, _ = FormatBatch(two, Options{Format: FormatJSON, Quiet: true, Now: now, NoColor: true})
	if strings.Contains(s, "Token 1") {
		t.Errorf("quiet should suppress headers: %s", s)
	}
	// Claim batch prints one value per line, no headers.
	s, _ = FormatBatch(two, Options{Claim: "sub", Now: now})
	if s != "1234567890\n1234567890" {
		t.Errorf("claim batch: %q", s)
	}
}

func TestTable(t *testing.T) {
	tok := testToken(t)
	s, err := FormatBatch([]jwt.Token{tok}, Options{Format: FormatTable, NoColor: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "alg") || !strings.Contains(s, "HS256") {
		t.Errorf("table: %s", s)
	}
}
