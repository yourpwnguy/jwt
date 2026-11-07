package tui

// render_help.go is the manual page. The first block is the one thing
// people need to discover: Vim-style movement without leaving insert
// mode. Below it the keymap splits into two columns on wide terminals,
// like a printed reference card. On narrow terminals it becomes one
// column and scrolls.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) helpContent(width int) []string {
	lines := m.helpMotions(width)
	lines = append(lines, "")
	if width >= 96 {
		colW := (width - 4) / 2
		left := m.helpLeftColumn(colW)
		right := m.helpRightColumn(width - colW - 4)
		lines = append(lines, joinHelpColumns(left, right, width)...)
	} else {
		lines = append(lines, m.helpLeftColumn(width)...)
		lines = append(lines, "")
		lines = append(lines, m.helpRightColumn(width)...)
	}
	lines = append(lines, m.theme.mutedStyle().Render("Exact bytes. HMAC signing only. Nothing leaves this process."))
	return lines
}

// helpMotions is a printed callout, not another list row. If the
// motions are disabled, it explains how to get them back instead.
func (m model) helpMotions(width int) []string {
	accent := m.theme.sectionColor(sectionHeader)
	bar := m.theme.fg(accent).Render("┃")
	textW := max(8, width-4)
	seq := bgSequence(m.theme.surface)

	type blockLine struct {
		text  string
		style lipgloss.Style
	}
	var body []blockLine
	if m.vimKeys {
		keys := "ctrl+h ←  ctrl+j ↓  ctrl+k ↑  ctrl+l →"
		if width >= 76 {
			keys = "ctrl+h left   ·   ctrl+j down   ·   ctrl+k up   ·   ctrl+l right"
		}
		body = []blockLine{
			{"MOVE WITHOUT LEAVING INSERT MODE", m.theme.bold(m.theme.ink)},
			{keys, m.theme.bold(accent)},
			{"Arrows always work. ctrl+i is Tab, so Tab still changes sections.", m.theme.mutedStyle()},
		}
	} else {
		body = []blockLine{
			{"VIM MOTIONS ARE OFF", m.theme.bold(m.theme.warning)},
			{"Start jwt tui without --no-vim-keys to turn ctrl+h/j/k/l back on.", m.theme.mutedStyle()},
			{"Arrow keys work in either mode.", m.theme.mutedStyle()},
		}
	}

	out := make([]string, 0, len(body)+1)
	for _, line := range body {
		for _, wrapped := range wrapPlain(line.text, textW) {
			out = append(out, paintBG(padLine(bar+"  "+line.style.Render(wrapped), width), seq))
		}
	}
	return out
}

func (m model) helpLeftColumn(width int) []string {
	var lines []string
	lines = append(lines,
		m.helpSection(1, "EDITING", sectionHeader, width, [][2]string{
			{"tab / shift+tab", "next or previous section"},
			{"ctrl+o / esc", "leave editing, enter command mode"},
			{"ctrl+s / ctrl+y", "sign or copy the token"},
			{"ctrl+c", "quit from anywhere"},
		})...)
	lines = append(lines, "")
	lines = append(lines,
		m.helpSection(2, "COMMAND MODE", sectionPayload, width, [][2]string{
			{"i / enter", "back to editing"},
			{"1 / 2 / 3", "header, payload, signature"},
			{"f / c", "format or compact the current JSON"},
			{"s / y", "sign or copy"},
			{"a / v", "full audit or full token"},
			{"? / q", "help or quit"},
		})...)
	return lines
}

func (m model) helpRightColumn(width int) []string {
	var lines []string
	lines = append(lines,
		m.helpSection(3, "SIGNING", sectionSignature, width, [][2]string{
			{"enter", "sign with the secret"},
			{"ctrl+r", "reveal or hide the secret"},
			{"esc", "cancel"},
		})...)
	lines = append(lines, "")
	lines = append(lines,
		m.helpSection(4, "READING", sectionHeader, width, [][2]string{
			{"↑ / ↓", "scroll one line"},
			{"pgup / pgdown", "scroll one page"},
			{"g / G", "top or bottom"},
			{"y", "copy the token"},
			{"esc", "close the page"},
		})...)
	lines = append(lines, "")
	lines = append(lines,
		m.helpSection(5, "NOTES", sectionPayload, width, [][2]string{
			{"backspace", "still deletes; Delete goes forward"},
			{"ctrl+i", "is Tab, so Tab switches sections"},
			{"--no-vim-keys", "start with motions disabled"},
		})...)
	return lines
}

func (m model) helpSection(number int, title string, s section, width int, rows [][2]string) []string {
	num := m.theme.bold(m.theme.sectionColor(s)).Render(fmt.Sprintf("%02d", number))
	head := num + "  " + m.theme.bold(m.theme.ink).Render(title)
	lines := []string{padLine(head, width), m.rule(width, m.theme.hairline)}
	for _, row := range rows {
		lines = append(lines, m.helpRow(row[0], row[1], width, m.theme.sectionColor(s))...)
	}
	return lines
}

func (m model) helpRow(keys, desc string, width int, color lipgloss.TerminalColor) []string {
	const keyCol = 17
	key := m.theme.bold(color).Render(fmt.Sprintf("%-*s", keyCol, keys))
	descWidth := max(8, width-keyCol-1)
	wrapped := wrapPlain(desc, descWidth)
	out := make([]string, 0, len(wrapped))
	out = append(out, key+m.theme.mutedStyle().Render(wrapped[0]))
	for _, line := range wrapped[1:] {
		out = append(out, strings.Repeat(" ", keyCol)+m.theme.mutedStyle().Render(line))
	}
	return out
}

func joinHelpColumns(left, right []string, width int) []string {
	const gap = 4
	colW := (width - gap) / 2
	rightW := width - colW - gap
	height := max(len(left), len(right))
	out := make([]string, height)
	for i := 0; i < height; i++ {
		l := ""
		if i < len(left) {
			l = left[i]
		}
		r := ""
		if i < len(right) {
			r = right[i]
		}
		out[i] = padLine(l, colW) + strings.Repeat(" ", gap) + padLine(r, rightW)
	}
	return out
}
