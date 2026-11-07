package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// editor.go builds the three editable widgets. The textareas share the
// same behavior and differ only in accent color; the signature field is
// a single line because base64url has no newlines.
//
// CharLimit and MaxHeight stay unlimited on purpose. A 16 KB compact
// segment can pretty-print to more than 16 KB and past 99 lines, and
// Bubbles would silently truncate the loaded text if those defaults
// were left in place. Validation in refresh.go rejects an oversized
// segment when it is time to encode.

func newTextarea(t theme, s section) textarea.Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.Placeholder = "{}"
	ta.ShowLineNumbers = true
	ta.EndOfBufferCharacter = ' '
	ta.CharLimit = 0
	ta.MaxHeight = 0

	accent := t.sectionColor(s)
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(accent)

	focused := textarea.Style{
		Base:             lipgloss.NewStyle(),
		CursorLine:       lipgloss.NewStyle().Background(t.cursorLine(s)),
		CursorLineNumber: lipgloss.NewStyle().Foreground(accent).Bold(true),
		EndOfBuffer:      t.mutedStyle(),
		LineNumber:       t.mutedStyle(),
		Placeholder:      t.mutedStyle(),
		Prompt:           t.mutedStyle(),
		Text:             lipgloss.NewStyle(),
	}
	blurred := focused
	blurred.CursorLine = lipgloss.NewStyle()
	blurred.CursorLineNumber = t.mutedStyle()

	ta.FocusedStyle = focused
	ta.BlurredStyle = blurred
	return ta
}

func newSignatureInput(t theme) textinput.Model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "(empty)"
	ti.CharLimit = 0
	ti.PromptStyle = t.bold(t.signature)
	ti.TextStyle = lipgloss.NewStyle()
	ti.PlaceholderStyle = t.mutedStyle()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(t.signature)
	ti.Cursor.TextStyle = lipgloss.NewStyle()
	return ti
}

func newSecretInput(t theme) textinput.Model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "secret or @file"
	ti.CharLimit = 0
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.PromptStyle = t.bold(t.header)
	ti.TextStyle = lipgloss.NewStyle()
	ti.PlaceholderStyle = t.mutedStyle()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(t.header)
	ti.Cursor.TextStyle = lipgloss.NewStyle()
	return ti
}

// newCommandInput is the ":" line. It has the same quiet styling as
// the other inputs, with the signature accent on the prompt and cursor.
func newCommandInput(t theme) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ":"
	ti.CharLimit = 0
	ti.PromptStyle = t.bold(t.signature)
	ti.TextStyle = lipgloss.NewStyle()
	ti.PlaceholderStyle = t.mutedStyle()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(t.signature)
	ti.Cursor.TextStyle = lipgloss.NewStyle()
	return ti
}
