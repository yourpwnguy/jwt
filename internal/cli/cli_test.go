package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// CLI tests drive Run with buffers — no subprocesses, no real stdin files.

const testHS = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

func run(t *testing.T, args []string, stdin string, isTTY bool) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	var in *strings.Reader
	if stdin == "" {
		code := Run(context.Background(), args, nil, isTTY, &stdout, &stderr)
		return code, stdout.String(), stderr.String()
	}
	in = strings.NewReader(stdin)
	code := Run(context.Background(), args, in, isTTY, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestDecodeExitCodes(t *testing.T) {
	if code, out, _ := run(t, []string{"-t", testHS}, "", true); code != ExitOK || !strings.Contains(out, "1234567890") {
		t.Errorf("decode: code=%d out=%s", code, out)
	}
	if code, _, _ := run(t, []string{"-t", "bad.token.here"}, "", true); code != ExitSystemErr {
		t.Errorf("bad token: code=%d want 1", code)
	}
	if code, _, _ := run(t, []string{}, "", true); code != ExitSystemErr {
		t.Errorf("no input: code=%d want 1", code)
	}
	if code, out, _ := run(t, []string{"--claim", "sub", "-t", testHS}, "", true); code != ExitOK || strings.TrimSpace(out) != "1234567890" {
		t.Errorf("claim: code=%d out=%q", code, out)
	}
}

func TestAuditExitCodes(t *testing.T) {
	// Fixture has missing-exp (high): default fail-on high breaches.
	if code, _, _ := run(t, []string{"audit", "-t", testHS}, "", true); code != ExitValidation {
		t.Errorf("audit breach: code=%d want 2", code)
	}
	if code, _, _ := run(t, []string{"audit", "-t", testHS, "--fail-on", "critical"}, "", true); code != ExitOK {
		t.Errorf("audit crit gate: code=%d want 0", code)
	}
}

func TestVerifyExitCodes(t *testing.T) {
	if code, _, _ := run(t, []string{"verify", "-t", testHS, "--secret", "your-256-bit-secret"}, "", true); code != ExitOK {
		t.Errorf("verify ok: code=%d", code)
	}
	if code, _, _ := run(t, []string{"verify", "-t", testHS, "--secret", "wrong"}, "", true); code != ExitValidation {
		t.Errorf("verify wrong: code=%d want 2", code)
	}
	if code, _, _ := run(t, []string{"verify", "-t", testHS}, "", true); code != ExitSystemErr {
		t.Errorf("verify no key: code=%d want 1", code)
	}
}

func TestFormatFlags(t *testing.T) {
	// -format alias works everywhere -o does.
	if code, _, _ := run(t, []string{"-t", testHS, "-format", "json"}, "", true); code != ExitOK {
		t.Errorf("decode -format: code=%d want 0", code)
	}
	if code, _, _ := run(t, []string{"audit", "-t", testHS, "-format", "table", "--fail-on", "critical"}, "", true); code != ExitOK {
		t.Errorf("audit -format table: code=%d want 0", code)
	}
	if code, _, _ := run(t, []string{"verify", "-t", testHS, "--secret", "your-256-bit-secret", "-format", "table"}, "", true); code != ExitOK {
		t.Errorf("verify -format table: code=%d want 0", code)
	}
	// Bogus formats error instead of silently defaulting.
	if code, _, _ := run(t, []string{"-t", testHS, "-o", "yaml"}, "", true); code != ExitSystemErr {
		t.Errorf("bad format: code=%d want 1", code)
	}
	if code, _, _ := run(t, []string{"audit", "-t", testHS, "-o", "yaml"}, "", true); code != ExitSystemErr {
		t.Errorf("audit bad format: code=%d want 1", code)
	}
}
