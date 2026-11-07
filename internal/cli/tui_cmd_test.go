package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// tui_cmd_test.go checks the parts that do not need a real terminal:
// routing, the TTY guard, and the help text. The editor itself is
// covered by internal/tui tests and a PTY smoke test in development.

func TestTUIRouteGuardsNonTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"tui"}, strings.NewReader(""), false, &stdout, &stderr)
	if code != ExitSystemErr {
		t.Fatalf("exit = %d, want %d", code, ExitSystemErr)
	}
	if !strings.Contains(stderr.String(), "interactive terminal") {
		t.Fatalf("stderr = %q, want TTY error", stderr.String())
	}
}

func TestTUIUsageIsDiscoverable(t *testing.T) {
	var top, editor bytes.Buffer
	printUsage(&top)
	printTUIUsage(&editor)

	if !strings.Contains(top.String(), "tui") {
		t.Error("top-level usage does not mention tui")
	}
	if !strings.Contains(editor.String(), "jwt tui") {
		t.Error("tui usage missing examples")
	}
	if !strings.Contains(editor.String(), "ctrl+s") {
		t.Error("tui usage missing key hints")
	}
}

func TestTUIHelpWorksWithoutTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runTUI(context.Background(), []string{"-h"}, strings.NewReader(""), false, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stderr.String(), "Usage: jwt tui") {
		t.Fatalf("stderr = %q, want tui usage", stderr.String())
	}
}

func TestTUIBadThemeBeforeTTY(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runTUI(context.Background(), []string{"--theme", "neon"}, strings.NewReader(""), false, &stdout, &stderr)
	if code != ExitSystemErr {
		t.Fatalf("exit = %d, want %d", code, ExitSystemErr)
	}
	if !strings.Contains(stderr.String(), "invalid --theme") {
		t.Fatalf("stderr = %q, want theme error", stderr.String())
	}
}
