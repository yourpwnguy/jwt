package tui

// view_overlays.go renders the reading pages and the signing sheet.
// They share the masthead and footer with the main page, so moving into
// help or audit feels like turning a page, not opening a dialog. The
// signer is the one true modal: a sheet centered in the body.

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderOverlay() string {
	l := m.layout()
	lines := make([]string, 0, l.height)

	if l.topPad > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, frame(m.renderMasthead(l.contentW), l.margin, l.contentW))
	lines = append(lines, frame(m.rule(l.contentW, m.theme.hairline), l.margin, l.contentW))
	lines = append(lines, frame(m.renderPageTitle(l.contentW), l.margin, l.contentW))
	lines = append(lines, frame(m.rule(l.contentW, m.theme.hairline), l.margin, l.contentW))

	content := strings.Split(m.overlay.View(), "\n")
	content = fitAndPad(content, l.contentW, l.overlayH)
	for _, line := range content {
		lines = append(lines, frame(line, l.margin, l.contentW))
	}

	lines = append(lines, frame(m.rule(l.contentW, m.theme.hairline), l.margin, l.contentW))
	lines = append(lines, frame(m.renderFooter(l.contentW), l.margin, l.contentW))
	if l.bottomPad > 0 {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m model) renderPageTitle(width int) string {
	name := "HELP"
	sub := "KEYS"
	color := m.theme.signature
	switch m.mode {
	case modeAudit:
		name, sub, color = "AUDIT", "OFFLINE", m.auditColor()
	case modeToken:
		name, sub, color = "TOKEN", "EXACT BYTES", m.theme.signature
	}
	left := m.theme.bold(color).Render(name) + m.theme.mutedStyle().Render("  /  "+sub)
	right := m.theme.mutedStyle().Render("ESC CLOSE")
	return joinLR(left, right, width)
}

func (m model) renderSign() string {
	l := m.layout()
	sheetW := min(68, l.contentW-8)
	if sheetW < 40 {
		sheetW = max(30, l.contentW-4)
	}
	// The sheet page has the same fixed chrome as the main page,
	// minus the tab row: masthead, two rules, and the footer.
	bodyH := max(3, l.height-l.topPad-l.bottomPad-4-2*l.gap)
	sheet := m.signSheet(sheetW, bodyH)
	body := lipgloss.Place(l.contentW, bodyH, lipgloss.Center, lipgloss.Center, sheet)

	lines := make([]string, 0, l.height)
	if l.topPad > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, frame(m.renderMasthead(l.contentW), l.margin, l.contentW))
	lines = append(lines, frame(m.rule(l.contentW, m.theme.hairline), l.margin, l.contentW))
	if l.gap > 0 {
		lines = append(lines, "")
	}
	for _, line := range strings.Split(body, "\n") {
		lines = append(lines, frame(line, l.margin, l.contentW))
	}
	if l.gap > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, frame(m.rule(l.contentW, m.theme.hairline), l.margin, l.contentW))
	lines = append(lines, frame(m.renderFooter(l.contentW), l.margin, l.contentW))
	if l.bottomPad > 0 {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// signSheet is a printed form: a letterhead stripe, the secret field,
// and the three keys that matter. It has no border; the surface wash and
// the stripe do the framing.
func (m model) signSheet(sheetW, maxH int) string {
	textW := sheetW - 6
	if textW < 20 {
		textW = 20
	}
	alg := strings.ToUpper(strings.TrimSpace(m.alg))
	if alg == "" {
		alg = "NO ALG"
	}
	head := joinLR(
		m.theme.bold(m.theme.ink).Render("SIGN")+m.theme.mutedStyle().Render("  /  HMAC"),
		m.theme.bold(m.theme.header).Render(alg),
		textW,
	)

	lines := []string{head, ""}
	if maxH >= 12 {
		for _, line := range wrapPlain("A secret, or @path to a file. It stays in memory and never leaves this process.", textW) {
			lines = append(lines, m.theme.mutedStyle().Render(line))
		}
		lines = append(lines, "")
	}

	lines = append(lines, m.theme.mutedStyle().Render("SECRET"))
	secret := m.secret
	secret.Width = max(8, textW-4)
	lines = append(lines, padLine(secret.View(), textW))

	if m.revealSecret {
		lines = append(lines, m.theme.fg(m.theme.warning).Render("secret is visible"))
	}
	if m.signErr != nil {
		for _, line := range wrapPlain(m.signErr.Error(), textW) {
			lines = append(lines, m.theme.fg(m.theme.danger).Render(line))
		}
	}

	lines = append(lines, "")
	var hints string
	if textW >= 46 {
		hints = m.theme.keycap("enter") + " " + m.theme.mutedStyle().Render("sign") + "   " +
			m.theme.keycap("ctrl+r") + " " + m.theme.mutedStyle().Render("reveal") + "   " +
			m.theme.keycap("esc") + " " + m.theme.mutedStyle().Render("cancel")
	} else {
		// The footer already spells out the actions. On a narrow sheet
		// the keys alone are enough, and they fit without an ellipsis.
		hints = m.theme.keycap("enter") + "  " + m.theme.keycap("ctrl+r") + "  " + m.theme.keycap("esc")
	}
	lines = append(lines, padLine(hints, textW))

	// A form should not grow taller than the page it sits on. The
	// finished sheet adds a stripe, a blank row, and a bottom blank.
	lines = fitLines(lines, textW, max(3, maxH-3))

	surf := bgSequence(m.theme.surface)
	strip := bgSequence(fillHeader)
	blank := paintBG(strings.Repeat(" ", sheetW), surf)
	top := paintBG(strings.Repeat(" ", sheetW), strip)
	out := []string{top, blank}
	for _, line := range lines {
		out = append(out, paintBG(padLine("   "+line, sheetW), surf))
	}
	out = append(out, blank)
	return strings.Join(out, "\n")
}
