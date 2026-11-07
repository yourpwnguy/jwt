package tui

// view_cards.go renders the editor card and the token inspector. The
// layout arithmetic lives in layout.go; these functions only fill the
// rectangles it hands them.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// renderEditorCard wraps the active widget in a border colored for its
// section. The tab bar already names the section, so the card has no
// title and gives every row to the editor.
func (m model) renderEditorCard(totalW, totalH int) string {
	if totalW < 6 || totalH < 3 {
		return ""
	}
	innerW := totalW - 4
	innerH := totalH - 2

	var content []string
	if m.section == sectionSignature {
		content = m.signatureCardContent(innerW)
	} else {
		view := m.header.View()
		if m.section == sectionPayload {
			view = m.payload.View()
		}
		content = strings.Split(strings.TrimRight(view, "\n"), "\n")
	}
	content = fitAndPad(content, innerW, innerH)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.sectionColor(m.section)).
		Width(totalW-2).
		Height(innerH).
		Padding(0, 1)
	return style.Render(strings.Join(content, "\n"))
}

func (m model) signatureCardContent(width int) []string {
	input := m.signature.View()
	lines := []string{input, ""}

	stateText := strings.ToLower(m.sigState.label())
	if m.headerErr != nil || m.payloadErr != nil {
		stateText = "unavailable, JSON is invalid"
	}
	state := m.theme.fg(m.sigColor()).Render(stateText)
	lines = append(lines, m.theme.mutedStyle().Render("status   ")+state)
	lines = append(lines, m.theme.mutedStyle().Render("bytes    "+humanBytes(len(m.signature.Value()))+" base64url"))
	lines = append(lines, m.theme.mutedStyle().Render("input    "+humanBytes(len(m.signingInput))+" signed bytes"))

	if m.sigErr != nil {
		for _, line := range wrapPlain(m.sigErr.Error(), width) {
			lines = append(lines, m.theme.fg(m.theme.danger).Render(line))
		}
	}
	lines = append(lines, "")
	lines = append(lines, m.theme.mutedStyle().Render("ctrl+s sign · or paste a base64url signature above"))
	return fitLines(lines, width, max(1, len(lines)))
}

// renderInspectorCard is the one inspector: encoded token, facts, then
// the audit summary. On short terminals the audit gives way first and
// the full report stays behind the a key.
func (m model) renderInspectorCard(totalW, totalH int) string {
	if totalW < 6 || totalH < 3 {
		return ""
	}
	innerW := totalW - 4
	innerH := totalH - 2
	content := m.inspectorContent(innerW, innerH)
	content = fitAndPad(content, innerW, innerH)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.muted).
		Width(totalW-2).
		Height(innerH).
		Padding(0, 1)
	return style.Render(strings.Join(content, "\n"))
}

func (m model) inspectorContent(width, height int) []string {
	if height <= 0 {
		return nil
	}
	label := "ENCODED TOKEN"
	if m.currentToken == "" {
		label = "LAST VALID TOKEN"
	}
	// At two or three rows there is no room for a label. The base64
	// token is more useful than the word TOKEN.
	if height <= 3 {
		return fitLines(m.tokenDisplayLines(width, height), width, height)
	}

	lines := []string{
		joinLR(
			m.theme.bold(m.theme.signature).Render(label),
			m.theme.mutedStyle().Render(humanBytes(len(m.previewToken()))),
			width,
		),
	}
	if height >= 10 {
		lines = append(lines, "")
	}
	// On a very short card, the token itself is the only useful thing.
	// The audit badge in the top bar still tells you whether anything
	// needs attention.
	if height < 11 {
		lines = append(lines, m.tokenDisplayLines(width, max(1, height-len(lines)))...)
		return fitLines(lines, width, height)
	}

	meta := m.inspectorMeta(width)
	metaH := len(meta) + 2 // blank and rule
	docs := m.propertyDocs(width)
	docsH := 0
	if len(docs) > 0 {
		docsH = len(docs) + 2 // blank and rule
	}

	if height < 17 {
		maxToken := height - len(lines) - metaH - docsH
		if maxToken < 1 {
			maxToken = 1
		}
		lines = append(lines, m.tokenDisplayLines(width, maxToken)...)
		lines = append(lines, "", m.rule(width))
		lines = append(lines, meta...)
		if docsH > 0 {
			lines = append(lines, "", m.rule(width))
			lines = append(lines, docs...)
		}
		return fitLines(lines, width, height)
	}

	// Reserve three lines for the audit header plus one finding. The
	// real token is usually shorter than the cap, so the audit gets the
	// leftover room.
	maxToken := height - len(lines) - metaH - docsH - 3
	if maxToken < 1 {
		maxToken = 1
	}
	lines = append(lines, m.tokenDisplayLines(width, maxToken)...)
	lines = append(lines, "", m.rule(width))
	lines = append(lines, meta...)
	if docsH > 0 {
		lines = append(lines, "", m.rule(width))
		lines = append(lines, docs...)
	}

	remaining := height - len(lines)
	if remaining >= 2 {
		lines = append(lines, "", m.rule(width))
		remaining = height - len(lines)
		lines = append(lines, m.renderAuditInline(width, remaining)...)
	}
	return fitLines(lines, width, height)
}

func (m model) inspectorMeta(width int) []string {
	alg := m.alg
	if alg == "" {
		alg = "missing"
	}
	state := strings.ToLower(m.sigState.label())
	if m.headerErr != nil || m.payloadErr != nil {
		state = "unavailable"
	}
	if m.sigErr != nil {
		state = "invalid encoding"
	}
	exp := m.expMessage
	if exp == "" {
		exp = "not available"
	}
	rows := [][2]string{
		{"signature", state},
		{"alg", alg},
		{"exp", exp},
		{"size", humanBytes(len(m.previewToken()))},
	}
	colors := []lipgloss.TerminalColor{
		m.sigColor(), m.theme.header, m.expColor(), m.theme.muted,
	}
	out := make([]string, 0, len(rows))
	for i, row := range rows {
		label := fmt.Sprintf("%-10s", row[0])
		value := ansi.Truncate(row[1], max(4, width-10), "…")
		out = append(out, m.theme.mutedStyle().Render(label)+m.theme.fg(colors[i]).Render(value))
	}
	return out
}

func (m model) expColor() lipgloss.TerminalColor {
	switch m.expStatus {
	case jwt.TemporalExpired, jwt.TemporalNotYetValid:
		return m.theme.danger
	case jwt.TemporalNoExp:
		return m.theme.warning
	default:
		return m.theme.success
	}
}
