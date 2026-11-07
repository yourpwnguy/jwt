package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overlays.go opens the full-screen audit, token, and help screens. The
// overlay always returns to command mode on close, so the user can press
// i to edit or q to quit without guessing where they came from.

func (m *model) openAudit() tea.Cmd {
	width := m.overlayWidth()
	m.mode = modeAudit
	return m.setOverlay(m.auditFullContent(width))
}

func (m *model) openToken() tea.Cmd {
	width := m.overlayWidth()
	m.mode = modeToken
	return m.setOverlay(m.tokenPageContent(width))
}

func (m *model) openHelp() tea.Cmd {
	width := m.overlayWidth()
	m.mode = modeHelp
	return m.setOverlay(m.helpContent(width))
}

func (m *model) setOverlay(lines []string) tea.Cmd {
	m.overlay.Width = m.overlayWidth()
	m.overlay.Height = m.overlayHeight()
	m.overlay.SetContent(strings.Join(lines, "\n"))
	m.overlay.GotoTop()
	return nil
}

func (m model) overlayWidth() int {
	return max(20, m.layout().contentW)
}

func (m model) overlayHeight() int {
	return max(4, m.layout().overlayH)
}

func (m *model) handleOverlayKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q":
		m.mode = modeCommand
		return nil
	case "a":
		return m.openAudit()
	case "v":
		return m.openToken()
	case "?":
		return m.openHelp()
	case "y":
		return m.copyCurrent()
	case "g":
		m.overlay.GotoTop()
		return nil
	case "G":
		m.overlay.GotoBottom()
		return nil
	default:
		var cmd tea.Cmd
		m.overlay, cmd = m.overlay.Update(msg)
		return cmd
	}
}

func (m model) sigColor() lipgloss.TerminalColor {
	switch m.sigState {
	case sigSigned:
		return m.theme.success
	case sigStale, sigInvalid:
		return m.theme.danger
	case sigUnsigned, sigCustom:
		return m.theme.warning
	default:
		return m.theme.signature
	}
}
