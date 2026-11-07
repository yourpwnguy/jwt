package tui

// render_help.go is the cheat sheet behind "?". It is deliberately one
// page: five groups, the keys that matter, and a single note about the
// terminal's ctrl+i/Tab collision. The old manual tried to explain the
// whole app; this one assumes you can press : and read the command
// names.
//
// The content is plain styled text. The overlay viewport handles
// scrolling, so this file does not measure the screen height.

import (
	"fmt"
	"strings"
)

type helpRow struct {
	group string
	keys  string
	note  string
}

func (m model) helpContent(width int) []string {
	var lines []string
	if !m.vimKeys {
		lines = append(lines,
			m.theme.bold(m.theme.warning).Render("INSERT CTRL MOTIONS ARE OFF"),
			m.theme.mutedStyle().Render("Start without --no-vim-keys to turn ctrl+hjkl back on."),
			"",
		)
	}
	lines = append(lines, m.helpRows(width)...)
	lines = append(lines, "")
	for _, note := range []string{
		"The editor starts in normal mode. i or enter begins editing; esc returns.",
		"Arrows always work. ctrl+i is Tab. --no-vim-keys disables ctrl+hjkl while editing.",
		"Quotes and brackets auto-close in JSON; type the closer again to step over it.",
		"The inspector explains the JSON key under the cursor.",
	} {
		for _, line := range wrapPlain(note, width) {
			lines = append(lines, m.theme.mutedStyle().Render(line))
		}
	}
	return lines
}

func (m model) helpRows(width int) []string {
	rows := []helpRow{
		{"MOVE", "h j k l", "left, down, up, right"},
		{"", "w / b", "next / previous word"},
		{"", "0 / $", "start / end of line"},
		{"", "f{char} / ;", "find / repeat find"},
		{"", "gg / G", "first / last line"},
		{"EDIT", "i / enter", "insert at the cursor"},
		{"", "a", "append after the cursor"},
		{"", "o / O", "open line below / above"},
		{"", "c{motion}", "cw cb c$ cc"},
		{"", "d{motion}", "dd dw diw daw"},
		{"", "s", "change character"},
		{"TOKEN", "y", "copy the encoded token"},
		{"", ":sign", "sign with a secret"},
		{"", ":token / :audit", "full token / audit"},
		{"VIEW", "1 2 3", "header / payload / signature"},
		{"", "tab / shift+tab", "next / previous section"},
		{"", "? / :q", "help / quit"},
		{"INSERT", "esc", "back to normal mode"},
		{"", "ctrl+hjkl", "move while editing"},
		{"", "ctrl+s", "sign"},
	}

	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, m.helpRow(row, width)...)
	}
	return out
}

func (m model) helpRow(row helpRow, width int) []string {
	const groupW = 8
	const keyW = 20
	noteW := max(8, width-groupW-keyW)

	group := m.theme.mutedStyle().Render(fmt.Sprintf("%-*s", groupW, row.group))
	keys := m.theme.bold(m.theme.header).Render(fmt.Sprintf("%-*s", keyW, row.keys))
	wrapped := wrapPlain(row.note, noteW)

	out := make([]string, 0, len(wrapped))
	out = append(out, group+keys+m.theme.mutedStyle().Render(wrapped[0]))
	for _, line := range wrapped[1:] {
		out = append(out, strings.Repeat(" ", groupW+keyW)+m.theme.mutedStyle().Render(line))
	}
	return out
}
