package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// update.go is the keyboard router. Global keys are handled first, then
// the active mode. Editing keys are forwarded to the focused widget, so
// typing "q" in the JSON editor inserts q instead of quitting.

type tickMsg time.Time

type copiedMsg struct {
	err     error
	warning string
	size    int
}

func (m model) Init() tea.Cmd {
	if m.startCmd == nil {
		return tickCmd()
	}
	return tea.Batch(m.startCmd, tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		// Overlay content is wrapped to the viewport width, so it has
		// to be rebuilt when the window changes underneath it.
		switch m.mode {
		case modeAudit:
			return m, m.openAudit()
		case modeHelp:
			return m, m.openHelp()
		case modeToken:
			return m, m.openToken()
		}
		return m, nil
	case tickMsg:
		m.tick(time.Time(msg))
		return m, tickCmd()
	case toastClearMsg:
		if msg.id == m.toastID {
			m.toast = ""
		}
		return m, nil
	case copiedMsg:
		cmd := m.finishCopy(msg)
		return m, cmd
	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		return m, cmd
	default:
		// Cursor blink and paste messages belong to the focused widget.
		if m.mode == modeEdit {
			return m, m.updateActive(msg)
		}
		if m.mode == modeSign {
			return m, m.updateSecret(msg)
		}
		return m, nil
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Ctrl-C always quits, even mid-edit or inside a dialog. Esc is the
	// non-destructive close key.
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}
	switch m.mode {
	case modeSign:
		return m.handleSignKey(msg)
	case modeAudit, modeHelp, modeToken:
		return m.handleOverlayKey(msg)
	case modeCommand:
		return m.handleCommandKey(msg)
	default:
		return m.handleEditKey(msg)
	}
}

func (m *model) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+o":
		m.mode = modeCommand
		m.blurAll()
		return nil
	case "tab":
		return m.nextSection(1)
	case "shift+tab":
		return m.nextSection(-1)
	case "ctrl+s":
		return m.startSign()
	case "ctrl+y":
		return m.copyCurrent()
	default:
		return m.updateActive(msg)
	}
}

func (m *model) handleCommandKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "i", "enter", "ctrl+o":
		m.mode = modeEdit
		return m.focusSection()
	case "tab":
		return m.nextSection(1)
	case "shift+tab":
		return m.nextSection(-1)
	case "1":
		m.section = sectionHeader
		m.mode = modeEdit
		return m.focusSection()
	case "2":
		m.section = sectionPayload
		m.mode = modeEdit
		return m.focusSection()
	case "3":
		m.section = sectionSignature
		m.mode = modeEdit
		return m.focusSection()
	case "s":
		return m.startSign()
	case "y":
		return m.copyCurrent()
	case "f":
		return m.formatActive()
	case "c":
		return m.compactActive()
	case "a":
		return m.openAudit()
	case "v":
		return m.openToken()
	case "?":
		return m.openHelp()
	case "q":
		return tea.Quit
	}
	return nil
}

func (m *model) nextSection(delta int) tea.Cmd {
	next := (int(m.section) + delta + 3) % 3
	m.section = section(next)
	if m.mode == modeCommand {
		m.mode = modeEdit
	}
	return m.focusSection()
}

func (m *model) updateActive(msg tea.Msg) tea.Cmd {
	switch m.section {
	case sectionHeader:
		before := m.header.Value()
		var cmd tea.Cmd
		m.header, cmd = m.header.Update(msg)
		if m.header.Value() != before {
			m.headerDirty = true
			m.refresh()
		}
		return cmd
	case sectionPayload:
		before := m.payload.Value()
		var cmd tea.Cmd
		m.payload, cmd = m.payload.Update(msg)
		if m.payload.Value() != before {
			m.payloadDirty = true
			m.refresh()
		}
		return cmd
	case sectionSignature:
		before := m.signature.Value()
		var cmd tea.Cmd
		m.signature, cmd = m.signature.Update(msg)
		if m.signature.Value() != before {
			m.refresh()
		}
		return cmd
	}
	return nil
}

func (m *model) updateSecret(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.secret, cmd = m.secret.Update(msg)
	return cmd
}

func (m *model) finishCopy(msg copiedMsg) tea.Cmd {
	if msg.err != nil {
		return m.showToast("copy failed: "+msg.err.Error(), toastDanger)
	}
	text := "copied " + humanBytes(msg.size)
	kind := toastSuccess
	if msg.warning != "" {
		text += ", " + msg.warning
		kind = toastWarning
	}
	return m.showToast(text, kind)
}
