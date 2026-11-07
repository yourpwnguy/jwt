package tui

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// clipboard.go copies through OSC 52, the terminal clipboard escape.
//
// Why not xclip or pbcopy? Shelling out fails over SSH, needs an X or
// Wayland session, and adds platform branches. OSC 52 is handled by the
// terminal itself, so it works in the same places the TUI does. If a
// terminal does not support it, the token is still on screen and can be
// selected by hand.
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
