package tui

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOSC52Sequence(t *testing.T) {
	got := osc52Sequence("abc")
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("abc")) + "\a"
	if got != want {
		t.Fatalf("sequence = %q, want %q", got, want)
	}
}

func TestCopyTokenFallsBackToOSC52(t *testing.T) {
	// An empty PATH means no native clipboard command can be found, so
	// the fallback must be the terminal escape.
	t.Setenv("PATH", t.TempDir())
	clearRemoteClipboardEnv(t)

	var buf bytes.Buffer
	if err := copyToken(&buf, "abc"); err != nil {
		t.Fatalf("copyToken: %v", err)
	}
	if got, want := buf.String(), osc52Sequence("abc"); got != want {
		t.Fatalf("fallback = %q, want %q", got, want)
	}
}

func TestCopyTokenPrefersOSC52OnRemote(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("SSH_CONNECTION", "192.0.2.1 1234 192.0.2.2 22")
	t.Setenv("SSH_TTY", "")
	t.Setenv("MOSH_CONNECTION", "")

	var buf bytes.Buffer
	if err := copyToken(&buf, "abc"); err != nil {
		t.Fatalf("copyToken: %v", err)
	}
	if got, want := buf.String(), osc52Sequence("abc"); got != want {
		t.Fatalf("remote copy = %q, want %q", got, want)
	}
}

func TestCopyTokenUsesNativeClipboard(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the fake wl-copy shim is Linux-only")
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "clipboard.txt")
	script := "#!/bin/sh\ncat > \"$FAKE_CLIPBOARD_OUT\"\n"
	if err := os.WriteFile(filepath.Join(dir, "wl-copy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_CLIPBOARD_OUT", out)
	clearRemoteClipboardEnv(t)

	var buf bytes.Buffer
	if err := copyToken(&buf, "abc123"); err != nil {
		t.Fatalf("copyToken: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read clipboard: %v", err)
	}
	if string(data) != "abc123" {
		t.Fatalf("clipboard = %q, want abc123", data)
	}
	if buf.Len() != 0 {
		t.Fatalf("native copy also wrote OSC 52: %q", buf.String())
	}
}

func clearRemoteClipboardEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"SSH_CONNECTION", "SSH_TTY", "MOSH_CONNECTION", "WAYLAND_DISPLAY", "DISPLAY"} {
		t.Setenv(key, "")
	}
}
