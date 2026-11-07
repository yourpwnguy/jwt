package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// commandline.go is the ":" line. App actions used to live on single
// letters, which made normal mode impossible to keep Neovim-compatible:
// c, f, s, a, and v all had jobs. Moving those actions here gives the
// letters back to editing and gives the app a place to grow commands
// without another key.
//
// The command line is small on purpose. It accepts the handful of
// actions the editor actually needs and reports unknown words instead
// of guessing.

func (m *model) startEx() tea.Cmd {
	m.mode = modeEx
	m.pending = ""
	m.blurAll()
	m.command.SetValue("")
	m.command.CursorEnd()
	return m.command.Focus()
}

func (m *model) handleExKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		return m.cancelEx()
	case "enter":
		return m.runEx()
	default:
		var cmd tea.Cmd
		m.command, cmd = m.command.Update(msg)
		return cmd
	}
}

func (m *model) cancelEx() tea.Cmd {
	m.command.Blur()
	m.command.SetValue("")
	m.mode = modeCommand
	return m.focusNormal()
}

func (m *model) runEx() tea.Cmd {
	text := strings.TrimSpace(m.command.Value())
	m.command.Blur()
	m.command.SetValue("")
	m.mode = modeCommand

	switch text {
	case "":
		return m.focusNormal()
	case "q", "quit":
		return tea.Quit
	case "sign":
		// startSign can refuse (invalid JSON, unsupported alg). If it
		// refuses, normal mode has to get its visible cursor back.
		cmd := m.startSign()
		if m.mode == modeCommand {
			return tea.Batch(m.focusNormal(), cmd)
		}
		return cmd
	case "audit":
		return m.openAudit()
	case "token":
		return m.openToken()
	case "help":
		return m.openHelp()
	case "format":
		return tea.Batch(m.focusNormal(), m.formatActive())
	case "compact":
		return tea.Batch(m.focusNormal(), m.compactActive())
	case "header":
		m.section = sectionHeader
		return m.focusNormal()
	case "payload":
		m.section = sectionPayload
		return m.focusNormal()
	case "signature":
		m.section = sectionSignature
		return m.focusNormal()
	default:
		return tea.Batch(m.focusNormal(), m.showToast("unknown command: "+text, toastDanger))
	}
}
