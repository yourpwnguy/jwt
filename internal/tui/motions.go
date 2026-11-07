package tui

// motions.go implements the Neovim-style motions that normal mode
// exposes. The widgets already know how to move by character and line,
// but word motion, find, and the c operator need to reason about the
// text as a whole, so they live here instead of being wedged into the
// key router.
//
// The textarea stores its value as lines of runes and its cursor as a
// rune offset inside the current line. ColumnOffset is a display
// offset, so the logical position is StartColumn+ColumnOffset; that
// pair also works when a long line soft-wraps.
//
// Word motion uses three classes: whitespace, keyword characters
// (letters, digits, underscore), and punctuation. A run of punctuation
// is one word, which matches Vim closely enough for JSON editing and
// keeps "w" predictable around quotes, braces, and commas.

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

func runeClass(r rune) int {
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return 1
	default:
		return 2
	}
}

func (m model) activeText() string {
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

func (m model) activeLine() int {
	switch m.section {
	case sectionHeader:
		return m.header.Line()
	case sectionPayload:
		return m.payload.Line()
	default:
		return 0
	}
}

func (m model) activeCol() int {
	switch m.section {
	case sectionHeader:
		li := m.header.LineInfo()
		return li.StartColumn + li.ColumnOffset
	case sectionPayload:
		li := m.payload.LineInfo()
		return li.StartColumn + li.ColumnOffset
	case sectionSignature:
		return m.signature.Position()
	default:
		return 0
	}
}

func (m model) activeLineRunes() []rune {
	lines := strings.Split(m.activeText(), "\n")
	line := m.activeLine()
	if line < 0 || line >= len(lines) {
		return nil
	}
	return []rune(lines[line])
}

// setActiveCursor moves the logical cursor and then nudges the widget
// with an empty key message. The nudge is not a hack around a missing
// API: the textarea only repositions its viewport inside Update, so a
// direct SetCursor would happily move the cursor off screen.
func (m *model) setActiveCursor(line, col int) {
	lines := strings.Split(m.activeText(), "\n")
	if len(lines) == 0 {
		return
	}
	line = clamp(line, 0, len(lines)-1)
	col = clamp(col, 0, len([]rune(lines[line])))

	switch m.section {
	case sectionHeader:
		for m.header.Line() < line {
			m.header.CursorDown()
		}
		for m.header.Line() > line {
			m.header.CursorUp()
		}
		m.header.SetCursor(col)
	case sectionPayload:
		for m.payload.Line() < line {
			m.payload.CursorDown()
		}
		for m.payload.Line() > line {
			m.payload.CursorUp()
		}
		m.payload.SetCursor(col)
	case sectionSignature:
		m.signature.SetCursor(col)
	}
	m.settleActive()
}

// settleActive forces the widget to reposition its viewport without
// changing the value. An empty rune message falls through the textarea
// switch to "insert nothing" and then runs repositionView.
func (m *model) settleActive() {
	m.updateActive(tea.KeyMsg{Type: tea.KeyRunes})
}

func (m *model) moveLeft() tea.Cmd {
	if m.activeCol() > 0 {
		m.setActiveCursor(m.activeLine(), m.activeCol()-1)
	}
	return nil
}

func (m *model) moveRight() tea.Cmd {
	line := m.activeLine()
	runes := m.activeLineRunes()
	// Normal mode stops on the last character, like Vim. Positions past
	// the end belong to insert mode, which a and i enter explicitly.
	if len(runes) > 0 && m.activeCol() < len(runes)-1 {
		m.setActiveCursor(line, m.activeCol()+1)
	}
	return nil
}

// moveDown and moveUp step by logical lines, not display rows. A long
// wrapped JSON line should still be one line to j and k, the way it is
// in Neovim.
func (m *model) moveDown() tea.Cmd {
	line := m.activeLine()
	lines := strings.Split(m.activeText(), "\n")
	if line+1 >= len(lines) {
		return nil
	}
	m.setActiveCursor(line+1, m.activeCol())
	return nil
}

func (m *model) moveUp() tea.Cmd {
	line := m.activeLine()
	if line <= 0 {
		return nil
	}
	m.setActiveCursor(line-1, m.activeCol())
	return nil
}

func (m *model) moveLineStart() tea.Cmd {
	m.setActiveCursor(m.activeLine(), 0)
	return nil
}

// moveLineEnd lands on the last character, like Vim's $, rather than
// one cell past it the way an insert-mode cursor would.
func (m *model) moveLineEnd() tea.Cmd {
	line := m.activeLine()
	runes := m.activeLineRunes()
	col := 0
	if len(runes) > 0 {
		col = len(runes) - 1
	}
	m.setActiveCursor(line, col)
	return nil
}

func (m *model) moveBufferStart() tea.Cmd {
	m.setActiveCursor(0, 0)
	return nil
}

func (m *model) moveBufferEnd() tea.Cmd {
	lines := strings.Split(m.activeText(), "\n")
	if len(lines) == 0 {
		return nil
	}
	m.setActiveCursor(len(lines)-1, 0)
	return nil
}

func (m *model) moveWordForward() tea.Cmd {
	text := []rune(m.activeText())
	if len(text) == 0 {
		return nil
	}
	starts := lineStarts(text)
	index := posToIndex(text, starts, m.activeLine(), m.activeCol())
	start := index

	if index < len(text) && runeClass(text[index]) != 0 {
		class := runeClass(text[index])
		for index < len(text) && runeClass(text[index]) == class {
			index++
		}
	}
	for index < len(text) && runeClass(text[index]) == 0 {
		index++
	}
	if index >= len(text) {
		if start >= len(text)-1 {
			return nil
		}
		index = len(text) - 1
	}

	line, col := indexToPos(starts, index)
	m.setActiveCursor(line, col)
	return nil
}

func (m *model) moveWordBackward() tea.Cmd {
	text := []rune(m.activeText())
	if len(text) == 0 {
		return nil
	}
	starts := lineStarts(text)
	index := posToIndex(text, starts, m.activeLine(), m.activeCol())
	if index <= 0 {
		m.setActiveCursor(0, 0)
		return nil
	}

	index--
	for index > 0 && runeClass(text[index]) == 0 {
		index--
	}
	class := runeClass(text[index])
	for index > 0 && runeClass(text[index-1]) == class {
		index--
	}

	line, col := indexToPos(starts, index)
	m.setActiveCursor(line, col)
	return nil
}

func (m *model) findRune(r rune) tea.Cmd {
	line := m.activeLine()
	runes := m.activeLineRunes()
	for i := m.activeCol() + 1; i < len(runes); i++ {
		if runes[i] == r {
			m.setActiveCursor(line, i)
			return nil
		}
	}
	return nil
}

func (m *model) repeatFind() tea.Cmd {
	if m.lastFind == 0 {
		return nil
	}
	return m.findRune(m.lastFind)
}

// appendAtCursor is a: insert after the character under the cursor.
// At the end of a line it stays at the end instead of wrapping.
func (m *model) appendAtCursor() tea.Cmd {
	line := m.activeLine()
	col := m.activeCol()
	if col < len(m.activeLineRunes()) {
		col++
	}
	m.setActiveCursor(line, col)
	m.mode = modeEdit
	return m.focusSection()
}

// openLine is o and O. The new line is left empty; JSON indentation is
// the user's business, exactly like a plain Neovim buffer.
func (m *model) openLine(below bool) tea.Cmd {
	if m.section == sectionSignature {
		return m.showToast("signature is one line", toastInfo)
	}

	switch m.section {
	case sectionHeader:
		if below {
			m.header.CursorEnd()
		} else {
			m.header.CursorStart()
		}
	case sectionPayload:
		if below {
			m.payload.CursorEnd()
		} else {
			m.payload.CursorStart()
		}
	}

	m.updateActive(tea.KeyMsg{Type: tea.KeyEnter})
	if !below {
		// splitLine always leaves the cursor on the tail line. For O
		// the empty line is above, so step back onto it.
		switch m.section {
		case sectionHeader:
			m.header.CursorUp()
			m.header.SetCursor(0)
		case sectionPayload:
			m.payload.CursorUp()
			m.payload.SetCursor(0)
		}
		m.settleActive()
	}

	m.mode = modeEdit
	return m.focusSection()
}

// changeMotion implements c{motion}. It computes the range on the
// current logical line, rewrites that line, and leaves the cursor at
// the start of the change. Multi-line motions are limited to cj and ck
// so the common cases stay exact and the code stays readable.
func (m *model) changeMotion(motion string) tea.Cmd {
	lines := strings.Split(m.activeText(), "\n")
	line := m.activeLine()
	if line < 0 || line >= len(lines) {
		return nil
	}
	runes := []rune(lines[line])
	col := clamp(m.activeCol(), 0, len(runes))

	switch motion {
	case "j":
		if line+1 >= len(lines) {
			return m.changeLine(line, lines)
		}
		next := strings.Split(m.activeText(), "\n")
		next = append(next[:line], append([]string{""}, next[line+2:]...)...)
		m.replaceActiveText(strings.Join(next, "\n"), line, 0)
		m.mode = modeEdit
		return m.focusSection()
	case "k":
		if line == 0 {
			return m.changeLine(line, lines)
		}
		next := strings.Split(m.activeText(), "\n")
		next = append(next[:line-1], append([]string{""}, next[line+1:]...)...)
		m.replaceActiveText(strings.Join(next, "\n"), line-1, 0)
		m.mode = modeEdit
		return m.focusSection()
	}

	start, end, ok := lineMotionRange(runes, col, motion)
	if !ok {
		return nil
	}
	if start > end {
		start, end = end, start
	}

	lines[line] = string(runes[:start]) + string(runes[end:])
	m.replaceActiveText(strings.Join(lines, "\n"), line, start)
	m.mode = modeEdit
	return m.focusSection()
}

func (m *model) changeLine(line int, lines []string) tea.Cmd {
	lines[line] = ""
	m.replaceActiveText(strings.Join(lines, "\n"), line, 0)
	m.mode = modeEdit
	return m.focusSection()
}

// lineMotionRange is the single-line half of the c and d operators. It
// returns the half-open rune range [start, end) for the motion.
func lineMotionRange(runes []rune, col int, motion string) (int, int, bool) {
	col = clamp(col, 0, len(runes))
	switch motion {
	case "l", "s":
		return col, min(len(runes), col+1), true
	case "h":
		return max(0, col-1), col, true
	case "0":
		return 0, col, true
	case "$":
		return col, len(runes), true
	case "c":
		return 0, len(runes), true
	case "w":
		start := col
		end := col
		switch {
		case col < len(runes) && runeClass(runes[col]) == 0:
			end = nextWordStart(runes, col)
		case col < len(runes):
			class := runeClass(runes[col])
			end = col
			for end < len(runes) && runeClass(runes[end]) == class {
				end++
			}
			if end == col {
				end = nextWordStart(runes, col)
			}
		}
		return start, end, true
	case "b":
		return prevWordStart(runes, col), col, true
	default:
		return 0, 0, false
	}
}

// deleteMotion is d{motion} for the motions that stay inside one line.
// Unlike c{motion}, it stays in normal mode after the edit.
func (m *model) deleteMotion(motion string) tea.Cmd {
	lines, line, runes, ok := m.activeLineState()
	if !ok {
		return nil
	}
	start, end, ok := lineMotionRange(runes, m.activeCol(), motion)
	if !ok {
		return nil
	}
	// Vim's dw eats the trailing whitespace so the next word moves into
	// place. cw does not, which is why the two share a range but not
	// this small extension.
	if motion == "w" {
		for end < len(runes) && runeClass(runes[end]) == 0 {
			end++
		}
	}
	m.deleteRange(lines, line, runes, start, end)
	return nil
}

// deleteLine is dd. A middle line disappears and the next line moves
// up. The last line takes the newline before it, and a single-line
// buffer is left empty rather than removed.
func (m *model) deleteLine() tea.Cmd {
	lines := strings.Split(m.activeText(), "\n")
	line := m.activeLine()
	if line < 0 || line >= len(lines) {
		return nil
	}
	switch {
	case len(lines) == 1:
		m.replaceActiveText("", 0, 0)
	case line == len(lines)-1:
		lines = lines[:line]
		m.replaceActiveText(strings.Join(lines, "\n"), line-1, 0)
	default:
		lines = append(lines[:line], lines[line+1:]...)
		m.replaceActiveText(strings.Join(lines, "\n"), line, 0)
	}
	return nil
}

// deleteInnerWord is diw: the word under the cursor, without touching
// the whitespace around it.
func (m *model) deleteInnerWord() tea.Cmd {
	lines, line, runes, ok := m.activeLineState()
	if !ok {
		return nil
	}
	start, end := wordRun(runes, m.activeCol())
	m.deleteRange(lines, line, runes, start, end)
	return nil
}

// deleteAroundWord is daw: the word plus one side of whitespace, so the
// surrounding text closes up the way it does in Vim.
func (m *model) deleteAroundWord() tea.Cmd {
	lines, line, runes, ok := m.activeLineState()
	if !ok {
		return nil
	}
	start, end := wordRun(runes, m.activeCol())
	if start == end {
		return nil
	}
	if end < len(runes) && runeClass(runes[end]) == 0 {
		for end < len(runes) && runeClass(runes[end]) == 0 {
			end++
		}
	} else {
		for start > 0 && runeClass(runes[start-1]) == 0 {
			start--
		}
	}
	m.deleteRange(lines, line, runes, start, end)
	return nil
}

func (m *model) activeLineState() ([]string, int, []rune, bool) {
	lines := strings.Split(m.activeText(), "\n")
	line := m.activeLine()
	if line < 0 || line >= len(lines) {
		return nil, 0, nil, false
	}
	return lines, line, []rune(lines[line]), true
}

func (m *model) deleteRange(lines []string, line int, runes []rune, start, end int) {
	if start > end {
		start, end = end, start
	}
	if start == end {
		return
	}
	lines[line] = string(runes[:start]) + string(runes[end:])
	m.replaceActiveText(strings.Join(lines, "\n"), line, start)
}

// wordRun returns the word under the cursor. If the cursor is on
// whitespace, it returns the next word, which is what diw and daw do in
// Vim.
func wordRun(runes []rune, col int) (int, int) {
	col = clamp(col, 0, len(runes))
	if col < len(runes) && runeClass(runes[col]) != 0 {
		class := runeClass(runes[col])
		start := col
		for start > 0 && runeClass(runes[start-1]) == class {
			start--
		}
		end := col
		for end < len(runes) && runeClass(runes[end]) == class {
			end++
		}
		return start, end
	}
	start := nextWordStart(runes, col)
	if start >= len(runes) {
		return col, col
	}
	class := runeClass(runes[start])
	end := start
	for end < len(runes) && runeClass(runes[end]) == class {
		end++
	}
	return start, end
}

func nextWordStart(runes []rune, from int) int {
	i := from
	for i < len(runes) && runeClass(runes[i]) == 0 {
		i++
	}
	return i
}

func prevWordStart(runes []rune, from int) int {
	if from <= 0 {
		return 0
	}
	i := from - 1
	for i > 0 && runeClass(runes[i]) == 0 {
		i--
	}
	class := runeClass(runes[i])
	for i > 0 && runeClass(runes[i-1]) == class {
		i--
	}
	return i
}

// replaceActiveText is the bulk edit used by c{motion}. It only marks
// the pane dirty when the text actually changed, so a no-op change
// motion cannot re-encode an untouched segment and silently alter the
// token.
func (m *model) replaceActiveText(text string, line, col int) {
	switch m.section {
	case sectionHeader:
		before := m.header.Value()
		m.header.SetValue(text)
		if m.header.Value() != before {
			m.headerDirty = true
		}
	case sectionPayload:
		before := m.payload.Value()
		m.payload.SetValue(text)
		if m.payload.Value() != before {
			m.payloadDirty = true
		}
	case sectionSignature:
		m.signature.SetValue(text)
	}
	m.setActiveCursor(line, col)
	m.refresh()
}

// lineStarts returns the rune index of the first character on each
// line. The newline itself belongs to the line before it.
func lineStarts(text []rune) []int {
	starts := []int{0}
	for i, r := range text {
		if r == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func posToIndex(text []rune, starts []int, line, col int) int {
	line = clamp(line, 0, len(starts)-1)
	index := starts[line] + col
	if index < 0 {
		return 0
	}
	if index >= len(text) {
		return len(text)
	}
	return index
}

func indexToPos(starts []int, index int) (int, int) {
	if index < 0 {
		return 0, 0
	}
	line := 0
	for i := len(starts) - 1; i >= 0; i-- {
		if starts[i] <= index {
			line = i
			break
		}
	}
	return line, index - starts[line]
}
