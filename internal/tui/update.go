package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// update.go is the keyboard router. Global keys are handled first, then
// the active mode. Editing keys are forwarded to the focused widget, so
// typing "q" in the JSON editor inserts q instead of quitting.
//
// Normal mode is intentionally close to Neovim: h/j/k/l move, w/b jump
// by word, 0/$ go to the ends of the line, f{char}/; find, i/a/o/O
// enter insert mode, and c{motion} changes a range. App actions moved
// to the ":" line so those letters stay available for editing.

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
		switch m.mode {
		case modeEdit:
			return m, m.updateActive(msg)
		case modeEx:
			var cmd tea.Cmd
			m.command, cmd = m.command.Update(msg)
			return m, cmd
		case modeSign:
			return m, m.updateSecret(msg)
		case modeAudit, modeHelp, modeToken:
			var cmd tea.Cmd
			m.overlay, cmd = m.overlay.Update(msg)
			return m, cmd
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
	case modeEx:
		return m.handleExKey(msg)
	case modeCommand:
		return m.handleCommandKey(msg)
	default:
		return m.handleEditKey(msg)
	}
}

func (m *model) handleEditKey(msg tea.KeyMsg) tea.Cmd {
	// JSON punctuation gets a partner before the textarea sees it, so
	// the cursor lands inside the pair. The signature and command
	// widgets are not JSON and are excluded inside handleAutoPair.
	if m.handleAutoPair(msg) {
		return nil
	}

	// Vim motions are checked before the widget sees the key. The
	// textarea would otherwise treat ctrl+h as delete-backward and
	// ctrl+k as delete-after-cursor. Backspace and Delete still cover
	// those jobs, so taking the aliases costs nothing important.
	//
	// ctrl+i is not here on purpose: terminals send the same byte for
	// ctrl+i and Tab, and Tab is how sections are switched. The help
	// screen says this out loud so nobody thinks it is a bug.
	if m.vimKeys {
		switch msg.String() {
		case "ctrl+h":
			return m.updateActive(tea.KeyMsg{Type: tea.KeyLeft})
		case "ctrl+j":
			return m.updateActive(tea.KeyMsg{Type: tea.KeyDown})
		case "ctrl+k":
			return m.updateActive(tea.KeyMsg{Type: tea.KeyUp})
		case "ctrl+l":
			return m.updateActive(tea.KeyMsg{Type: tea.KeyRight})
		}
	}

	switch msg.String() {
	case "esc", "ctrl+o":
		return m.enterCommandMode()
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

// handleCommandKey is normal mode. The first switch handles the second
// key of a pending f or c, so "cw" never leaks a literal c into the
// buffer and "f\"" never inserts a quote.
func (m *model) handleCommandKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if m.pending == "g" {
		m.pending = ""
		if key == "g" {
			return m.moveBufferStart()
		}
		return nil
	}
	if m.pending == "di" {
		m.pending = ""
		if key == "w" {
			return m.deleteInnerWord()
		}
		return nil
	}
	if m.pending == "da" {
		m.pending = ""
		if key == "w" {
			return m.deleteAroundWord()
		}
		return nil
	}
	if m.pending == "d" {
		m.pending = ""
		switch key {
		case "d":
			return m.deleteLine()
		case "w", "b", "0", "$":
			return m.deleteMotion(key)
		case "i":
			m.pending = "di"
			return nil
		case "a":
			m.pending = "da"
			return nil
		}
		return nil
	}
	if m.pending == "f" {
		m.pending = ""
		if key == "esc" || len(msg.Runes) == 0 {
			return nil
		}
		m.lastFind = msg.Runes[0]
		return m.findRune(m.lastFind)
	}
	if m.pending == "c" {
		m.pending = ""
		switch key {
		case "w", "b", "0", "$", "c", "h", "l", "j", "k":
			return m.changeMotion(key)
		}
		return nil
	}

	switch key {
	case "h", "left":
		return m.moveLeft()
	case "l", "right":
		return m.moveRight()
	case "j", "down":
		return m.moveDown()
	case "k", "up":
		return m.moveUp()
	case "w":
		return m.moveWordForward()
	case "b":
		return m.moveWordBackward()
	case "0":
		return m.moveLineStart()
	case "$":
		return m.moveLineEnd()
	case "f":
		m.pending = "f"
		return nil
	case ";":
		return m.repeatFind()
	case "g":
		m.pending = "g"
		return nil
	case "G":
		return m.moveBufferEnd()
	case "d":
		m.pending = "d"
		return nil
	case "i", "enter", "ctrl+o":
		m.mode = modeEdit
		return m.focusSection()
	case "a":
		return m.appendAtCursor()
	case "o":
		return m.openLine(true)
	case "O":
		return m.openLine(false)
	case "s":
		return m.changeMotion("l")
	case "c":
		m.pending = "c"
		return nil
	case ":":
		return m.startEx()
	case "tab":
		return m.nextSection(1)
	case "shift+tab":
		return m.nextSection(-1)
	case "1":
		m.section = sectionHeader
		return m.focusNormal()
	case "2":
		m.section = sectionPayload
		return m.focusNormal()
	case "3":
		m.section = sectionSignature
		return m.focusNormal()
	case "ctrl+s":
		return m.startSign()
	case "ctrl+y", "y":
		return m.copyCurrent()
	case "?":
		return m.openHelp()
	case "q":
		return tea.Quit
	case "esc":
		m.pending = ""
		return nil
	}
	return nil
}

func (m *model) nextSection(delta int) tea.Cmd {
	next := (int(m.section) + delta + 3) % 3
	m.section = section(next)
	if m.mode == modeCommand {
		m.pending = ""
		return m.focusNormal()
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
