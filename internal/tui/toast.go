package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// toast.go is the short-lived status line in the footer. Toasts never
// block input and never stack: the newest one wins, and a stale timer
// cannot clear a newer message.
type toastClearMsg struct {
	id int
}

func (m *model) showToast(text string, kind toastKind) tea.Cmd {
	m.toastID++
	m.toast = toastPrefix(kind) + text
	m.toastK = kind
	id := m.toastID
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return toastClearMsg{id: id}
	})
}

func toastPrefix(kind toastKind) string {
	switch kind {
	case toastSuccess:
		return "✓ "
	case toastWarning:
		return "! "
	case toastDanger:
		return "✗ "
	default:
		return "● "
	}
}
