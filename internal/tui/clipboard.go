package tui

// clipboard.go copies the token to the real clipboard.
//
// The first version used only OSC 52, the terminal clipboard escape.
// That works over SSH and inside terminals that support it, but it is
// silently ignored in common local setups. tmux, for example, defaults
// to set-clipboard=external, so the escape never reaches the system
// clipboard. The toast said "copied" while nothing had been copied.
//
// The strategy now is:
//   - local session: try a native clipboard command first (wl-copy,
//     xclip, xsel, pbcopy, or clip), because we can verify it worked
//   - remote session: try OSC 52 first, because the local clipboard is
//     behind the terminal, not the remote machine
//   - either way, fall back to the other path before reporting failure
//
// Native commands receive the token on stdin, never as an argument, so
// the token does not show up in the process list. Each command gets a
// short timeout so a broken clipboard daemon cannot hang the editor.

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const clipboardTimeout = 2 * time.Second

type clipboardCommand struct {
	name string
	args []string
}

func copyOSC52(w io.Writer, token string) error {
	if w == nil {
		w = os.Stdout
	}
	_, err := io.WriteString(w, osc52Sequence(token))
	return err
}

// osc52Sequence builds the escape sequence. BEL is the widely supported
// terminator; ST works on some terminals but not all.
func osc52Sequence(token string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(token))
	return fmt.Sprintf("\x1b]52;c;%s\a", enc)
}

// copyToken is the default copy function. It tries the path that is
// most likely to work for the session and falls back to the other one.
func copyToken(w io.Writer, token string) error {
	if w == nil {
		w = os.Stdout
	}
	if isRemoteSession() {
		if err := copyOSC52(w, token); err == nil {
			return nil
		}
		return copyNative(token)
	}
	if err := copyNative(token); err == nil {
		return nil
	}
	return copyOSC52(w, token)
}

func isRemoteSession() bool {
	for _, key := range []string{"SSH_CONNECTION", "SSH_TTY", "MOSH_CONNECTION"} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

func copyNative(token string) error {
	var lastErr error
	for _, candidate := range nativeClipboardCommands() {
		if _, err := exec.LookPath(candidate.name); err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
		cmd := exec.CommandContext(ctx, candidate.name, candidate.args...)
		cmd.Stdin = strings.NewReader(token)
		err := cmd.Run()
		cancel()
		if err == nil {
			return nil
		}
		lastErr = fmt.Errorf("%s: %w", candidate.name, err)
	}
	if lastErr == nil {
		return errors.New("no native clipboard command found")
	}
	return lastErr
}

func nativeClipboardCommands() []clipboardCommand {
	switch runtime.GOOS {
	case "darwin":
		return []clipboardCommand{{name: "pbcopy"}}
	case "windows":
		return []clipboardCommand{{name: "clip"}}
	}
	return []clipboardCommand{
		{name: "wl-copy"},
		{name: "xclip", args: []string{"-selection", "clipboard"}},
		{name: "xsel", args: []string{"--clipboard", "--input"}},
	}
}

// copyCurrent copies the token currently on screen. Warnings are based
// on the honest signature state, so a stale token never looks like a
// clean copy.
func (m *model) copyCurrent() tea.Cmd {
	if m.currentToken == "" {
		return m.showToast("fix JSON before copying", toastDanger)
	}
	if m.copyFn == nil {
		return m.showToast("copy is not available", toastDanger)
	}

	var warnings []string
	switch m.sigState {
	case sigStale:
		warnings = append(warnings, "signature stale")
	case sigCustom:
		warnings = append(warnings, "custom signature")
	case sigUnsigned:
		warnings = append(warnings, "unsigned")
	case sigInvalid:
		warnings = append(warnings, "signature invalid")
	}
	if m.tokenErr != nil {
		warnings = append(warnings, "token has errors")
	}

	token := m.currentToken
	fn := m.copyFn
	warning := strings.Join(warnings, ", ")
	return func() tea.Msg {
		return copiedMsg{err: fn(token), warning: warning, size: len(token)}
	}
}
