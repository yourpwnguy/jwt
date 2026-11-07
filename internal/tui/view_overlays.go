package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// view_overlays.go renders the two full-screen treatments: the signing
// dialog and the scrollable audit/token/help screens. Both are centered
// and bounded, so a small terminal clips the card instead of the whole
// layout.

func (m model) renderSign() string {
	dialogW := min(64, m.width-8)
	if dialogW < 40 {
		dialogW = max(30, m.width-4)
	}
	textW := dialogW - 6 // border 2 + horizontal padding 4

	alg := strings.ToUpper(m.alg)
	title := m.theme.bold(m.theme.header).Render("SIGN TOKEN")
	head := joinLR(title, m.theme.pill(alg, fillHeader), textW)

	lines := []string{
		head,
		"",
		m.theme.mutedStyle().Render("HMAC secret"),
		m.secret.View(),
	}
	if m.revealSecret {
		lines = append(lines, m.theme.fg(m.theme.warning).Render("secret is visible"))
	}
	if m.signErr != nil {
		for _, line := range wrapPlain(m.signErr.Error(), textW) {
			lines = append(lines, m.theme.fg(m.theme.danger).Render(line))
		}
	}
	lines = append(lines, "")
	for _, line := range wrapPlain("The signature is recomputed over the exact header and payload bytes. The secret stays in memory only.", textW) {
		lines = append(lines, m.theme.mutedStyle().Render(line))
	}
	lines = append(lines,
		"",
		m.rule(textW),
		m.theme.mutedStyle().Render("enter sign · ctrl+r reveal · esc cancel"),
	)

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.header).
		Width(dialogW-2).
		Padding(1, 2)
	dialog := style.Render(strings.Join(fitLines(lines, textW, len(lines)), "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

func (m model) renderOverlay() string {
	cardW := max(40, m.width-4)
	textW := cardW - 6
	title := "HELP"
	var fill lipgloss.TerminalColor = fillSignature
	switch m.mode {
	case modeAudit:
		title = "AUDIT"
		fill = m.auditFill()
	case modeToken:
		title = "TOKEN"
	}
	head := joinLR(
		m.theme.pill(title, fill),
		m.theme.mutedStyle().Render("esc close"),
		textW,
	)

	content := []string{
		head,
		"",
		m.rule(textW),
	}
	content = append(content, strings.Split(m.overlay.View(), "\n")...)
	footer := m.theme.mutedStyle().Render("↑↓ scroll · g/G top/bottom · y copy · esc close")
	right := ""
	if m.toast != "" {
		right = m.theme.fg(m.toastColor()).Render(m.toast)
	}
	content = append(content, m.rule(textW), joinLR(footer, right, textW))
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.muted).
		Width(cardW-2).
		Padding(1, 2)
	card := style.Render(strings.Join(fitLines(content, textW, len(content)), "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
}
