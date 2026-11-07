package tui

// autopair.go is the syntax-aware part of insert mode. JSON keys and
// values are full of quotes and brackets, so typing the opening half
// inserts the closing half and leaves the cursor in the middle. Typing
// the closing half over an auto-inserted one steps over it instead of
// producing "" or }}.
//
// This only runs in the header and payload editors. The signature field
// is base64url, not JSON, and the command line is a command, not text,
// so neither should get automatic punctuation.

import tea "github.com/charmbracelet/bubbletea"

// handleAutoPair returns true when it consumed the key. A false return
// means the textarea should handle the character normally.
func (m *model) handleAutoPair(msg tea.KeyMsg) bool {
	if m.mode != modeEdit || (m.section != sectionHeader && m.section != sectionPayload) {
		return false
	}
	if len(msg.Runes) != 1 {
		return false
	}

	r := msg.Runes[0]
	line := m.activeLine()
	runes := m.activeLineRunes()
	col := m.activeCol()

	switch r {
	case '"':
		if escapedAt(runes, col) {
			return false
		}
		if col < len(runes) && runes[col] == '"' {
			m.setActiveCursor(line, col+1)
			return true
		}
		m.insertPair(line, col, '"', '"')
		return true
	case '{', '[', '(':
		if insideJSONString(runes, col) {
			return false
		}
		close := matchingClose(r)
		if col < len(runes) && runes[col] == close {
			m.setActiveCursor(line, col+1)
			return true
		}
		m.insertPair(line, col, r, close)
		return true
	case '}', ']', ')':
		if insideJSONString(runes, col) {
			return false
		}
		if col < len(runes) && runes[col] == r {
			m.setActiveCursor(line, col+1)
			return true
		}
	}
	return false
}

// insertPair writes both characters in one edit and then puts the
// cursor between them. One update means one refresh instead of a
// transient invalid-JSON flash between the two halves.
func (m *model) insertPair(line, col int, open, close rune) {
	m.updateActive(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{open, close}})
	m.setActiveCursor(line, col+1)
}

// escapedAt reports whether the cursor sits after an odd number of
// backslashes, which means the next quote is escaped and must not get a
// partner.
func escapedAt(runes []rune, col int) bool {
	count := 0
	for i := col - 1; i >= 0 && runes[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}

// insideJSONString scans the current line up to the cursor. JSON strings
// cannot cross a line, so the current line is enough context.
func insideJSONString(runes []rune, col int) bool {
	if col > len(runes) {
		col = len(runes)
	}
	inString := false
	escape := false
	for i := 0; i < col; i++ {
		switch {
		case escape:
			escape = false
		case inString && runes[i] == '\\':
			escape = true
		case runes[i] == '"':
			inString = !inString
		}
	}
	return inString
}

func matchingClose(r rune) rune {
	switch r {
	case '{':
		return '}'
	case '[':
		return ']'
	case '(':
		return ')'
	default:
		return r
	}
}
