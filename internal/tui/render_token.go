package tui

// render_token.go is the full-token page. The main screen already shows
// the token in miniature; this page is the specimen sheet. Each segment
// gets its own heading and color, then the facts underneath. It answers
// "what exactly will be copied?" without forcing the reader to find the
// dots in a long wrapped string.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) tokenPageContent(width int) []string {
	header, payload, sig := m.tokenSegments()
	var lines []string
	if m.currentToken == "" {
		lines = append(lines,
			m.theme.fg(m.theme.warning).Render("LAST VALID TOKEN  ·  fix the JSON to return to the live bytes"),
			"",
		)
	}
	lines = append(lines, m.tokenSegment("HEADER", header, m.theme.header, width)...)
	lines = append(lines, "")
	lines = append(lines, m.tokenSegment("PAYLOAD", payload, m.theme.payload, width)...)
	lines = append(lines, "")
	lines = append(lines, m.tokenSegment("SIGNATURE", sig, m.theme.signature, width)...)
	lines = append(lines, "")
	lines = append(lines, m.rule(width))
	lines = append(lines, "")
	lines = append(lines, m.tokenMetaLines(width)...)
	return lines
}

func (m model) tokenSegment(label, value string, color lipgloss.TerminalColor, width int) []string {
	head := joinLR(
		m.theme.bold(color).Render(label),
		m.theme.mutedStyle().Render(fmt.Sprintf("%d chars", len(value))),
		width,
	)
	lines := []string{head}
	if value == "" {
		lines = append(lines, m.theme.mutedStyle().Render("(empty)"))
		return lines
	}
	lines = append(lines, wrapRuns([]tokenRun{{text: value, color: color}}, width)...)
	return lines
}

func (m model) tokenMetaLines(width int) []string {
	alg := m.alg
	if alg == "" {
		alg = "missing"
	}
	_, _, sig := m.tokenSegments()
	state := strings.ToLower(m.sigLabel())
	stateColor := m.sigColor()
	if m.headerErr != nil || m.payloadErr != nil {
		state = "unavailable"
		stateColor = m.theme.warning
	}
	rows := []struct {
		label string
		value string
		color lipgloss.TerminalColor
	}{
		{"ALG", alg, m.theme.header},
		{"SIGNATURE", state, stateColor},
		{"SIZE", humanBytes(len(m.previewToken())), m.theme.muted},
		{"SIG CHARS", fmt.Sprintf("%d", len(sig)), m.theme.muted},
	}
	out := make([]string, 0, len(rows)+2)
	for _, row := range rows {
		label := m.theme.mutedStyle().Render(fmt.Sprintf("%-12s", row.label))
		value := m.theme.fg(row.color).Render(row.value)
		out = append(out, label+value)
	}
	if m.currentToken == "" {
		out = append(out, "",
			m.theme.mutedStyle().Render("Fix the JSON to make this token copyable again."))
		return out
	}
	out = append(out, "",
		m.theme.mutedStyle().Render("y copies the exact encoded token, dots included."))
	return out
}
