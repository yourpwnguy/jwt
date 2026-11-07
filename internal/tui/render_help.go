package tui

import (
	"fmt"
	"strings"
)

// render_help.go is the keymap screen. It is plain text plus a little
// color on the keys, so it stays readable when the terminal can only do
// 16 colors or the user passed --no-color.

func (m model) helpContent(width int) []string {
	lines := []string{
		m.theme.bold(m.theme.signature).Render("JWT TOKEN STUDIO"),
		m.theme.mutedStyle().Render("Edit JSON, watch the token and audit update live."),
		"",
	}
	sections := []struct {
		title string
		rows  [][2]string
	}{
		{
			title: "EDITING",
			rows: [][2]string{
				{"tab / shift+tab", "next or previous section"},
				{"ctrl+o / esc", "leave editing, enter command mode"},
				{"ctrl+s", "sign the current token"},
				{"ctrl+y", "copy the encoded token"},
				{"ctrl+c", "quit from anywhere"},
			},
		},
		{
			title: "COMMAND MODE",
			rows: [][2]string{
				{"i / enter / ctrl+o", "back to editing"},
				{"1 / 2 / 3", "header, payload, signature"},
				{"f / c", "format or compact the current JSON"},
				{"s / y", "sign or copy"},
				{"a / v", "full audit or full token"},
				{"? / q", "help or quit"},
			},
		},
		{
			title: "SIGNING",
			rows: [][2]string{
				{"enter", "sign with the current secret"},
				{"ctrl+r", "reveal or hide the secret"},
				{"esc", "cancel"},
			},
		},
		{
			title: "READING",
			rows: [][2]string{
				{"up / down", "scroll one line"},
				{"pgup / pgdown", "scroll one page"},
				{"g / G", "top or bottom"},
				{"y", "copy the token"},
			},
		},
	}
	for _, sec := range sections {
		lines = append(lines, m.theme.bold(m.theme.header).Render(sec.title))
		for _, row := range sec.rows {
			lines = append(lines, m.helpRow(row[0], row[1], width)...)
		}
		lines = append(lines, "")
	}
	lines = append(lines,
		m.theme.mutedStyle().Render("Exact bytes are encoded. Untouched panes keep their original segments."),
		m.theme.mutedStyle().Render("Signing is HMAC only in this build. Nothing leaves this process."),
	)
	return lines
}

func (m model) helpRow(keys, desc string, width int) []string {
	const keyCol = 22
	key := fmt.Sprintf("%-*s  ", keyCol, keys)
	descWidth := max(8, width-keyCol-2)
	wrapped := wrapPlain(desc, descWidth)
	out := make([]string, 0, len(wrapped))
	out = append(out, m.theme.bold(m.theme.header).Render(key)+m.theme.mutedStyle().Render(wrapped[0]))
	for _, line := range wrapped[1:] {
		out = append(out, strings.Repeat(" ", keyCol+2)+m.theme.mutedStyle().Render(line))
	}
	return out
}
