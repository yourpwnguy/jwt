package tui

import (
	"bytes"
	"encoding/json"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// format.go handles explicit JSON cleanup. Formatting changes the exact
// bytes, so it marks the pane dirty and the signature stale. That is
// intentional: the token you copy must match what you see.

func (m *model) formatActive() tea.Cmd {
	return m.rewriteActive(false)
}

func (m *model) compactActive() tea.Cmd {
	return m.rewriteActive(true)
}

func (m *model) rewriteActive(compact bool) tea.Cmd {
	if m.section == sectionSignature {
		return m.showToast("signature is not JSON", toastInfo)
	}

	text := m.sectionText()
	name := strings.ToLower(m.section.String())
	if err := validateObject(text, name); err != nil {
		return m.showToast(err.Error(), toastDanger)
	}

	var buf bytes.Buffer
	var err error
	if compact {
		err = json.Compact(&buf, []byte(text))
	} else {
		err = json.Indent(&buf, []byte(text), "", "  ")
	}
	if err != nil {
		return m.showToast(err.Error(), toastDanger)
	}

	m.setSectionText(buf.String())
	m.refresh()
	verb := "formatted"
	if compact {
		verb = "compacted"
	}
	return m.showToast(verb+" "+name, toastSuccess)
}

func (m model) sectionText() string {
	switch m.section {
	case sectionHeader:
		return m.header.Value()
	case sectionPayload:
		return m.payload.Value()
	case sectionSignature:
		return m.signature.Value()
	default:
		return ""
	}
}

// setSectionText is for programmatic rewrites (format, compact, tests).
// User keystrokes go through updateActive, which marks dirtiness only
// when the value actually changes.
func (m *model) setSectionText(text string) {
	switch m.section {
	case sectionHeader:
		m.header.SetValue(text)
		m.headerDirty = true
	case sectionPayload:
		m.payload.SetValue(text)
		m.payloadDirty = true
	case sectionSignature:
		m.signature.SetValue(text)
	}
}
