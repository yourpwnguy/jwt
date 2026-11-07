package tui

// view_chrome.go renders the masthead, the section index, and the
// footer. These are the parts that stay still while the panes change.
// The section index has two shapes: a vertical rail with a big page
// number on wide screens, and a typographic tab row on smaller ones.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) sectionError(s section) error {
	switch s {
	case sectionHeader:
		return m.headerErr
	case sectionPayload:
		return m.payloadErr
	case sectionSignature:
		return m.sigErr
	}
	return nil
}

// sigLabel avoids calling an incomplete token "invalid signature" when
// the real problem is JSON. The tabs point at the broken pane.
func (m model) sigLabel() string {
	if m.headerErr != nil || m.payloadErr != nil {
		return "JSON ERROR"
	}
	return m.sigState.label()
}

func (m model) renderMasthead(width int) string {
	brand := m.theme.bold(m.theme.ink).Render("JWT") + m.theme.fg(m.theme.payload).Render(".")
	left := brand
	if width >= 76 {
		left += "   " + m.theme.mutedStyle().Render(tracked("TOKEN PRESS"))
	}
	return joinLR(left, m.renderMastheadMeta(width), width)
}

// renderMastheadMeta is a single dateline, not a row of badges. Each
// value keeps its section or state color, while the dots make the
// three readings easy to scan.
func (m model) renderMastheadMeta(width int) string {
	alg := strings.ToUpper(strings.TrimSpace(m.alg))
	if alg == "" {
		alg = "NO ALG"
	}
	algColor := m.theme.header
	if alg == "NO ALG" || strings.EqualFold(alg, "NONE") {
		algColor = m.theme.danger
	}

	auditColor := m.auditColor()
	auditText := m.auditSummary()
	if !m.auditOK {
		auditText = "AUDIT PAUSED"
		if width < 92 {
			auditText = "PAUSED"
		}
	} else if m.report.FailedCount() > 0 {
		failed := m.report.FailedCount()
		auditText = fmt.Sprintf("%d ISSUE%s", failed, strings.ToUpper(plural(failed)))
	} else {
		auditText = fmt.Sprintf("%d/%d", len(m.report.Findings), len(m.report.Findings))
	}

	parts := []string{
		m.theme.fg(algColor).Render("●") + " " + m.theme.bold(algColor).Render(alg),
		m.theme.fg(m.sigColor()).Render("●") + " " + m.theme.bold(m.sigColor()).Render(m.sigLabel()),
		m.theme.fg(auditColor).Render("●") + " " + m.theme.bold(auditColor).Render(auditText),
	}
	sep := "   "
	if width < 92 {
		sep = "  "
	}
	return strings.Join(parts, sep)
}

type tabCell struct {
	text   string
	width  int
	active bool
	prefix int
}

func (m model) tabCells() []tabCell {
	cells := make([]tabCell, 0, 3)
	for s := sectionHeader; s <= sectionSignature; s++ {
		marker := ""
		if m.sectionError(s) != nil {
			marker = m.theme.fg(m.theme.danger).Render(" ✗")
		}
		num := fmt.Sprintf("%02d", int(s)+1)
		var text string
		if s == m.section {
			text = m.theme.bold(m.theme.sectionColor(s)).Render(num) +
				"  " + m.theme.bold(m.theme.ink).Render(s.String()) + marker
		} else {
			text = m.theme.mutedStyle().Render(num) +
				"  " + m.theme.fg(m.theme.ink).Render(s.String()) + marker
		}
		c := tabCell{text: text, width: lipgloss.Width(text), active: s == m.section}
		if len(cells) > 0 {
			c.prefix = 3
		}
		cells = append(cells, c)
	}
	return cells
}

func (m model) renderTabs(width int) string {
	cells := m.tabCells()
	var b strings.Builder
	for _, c := range cells {
		b.WriteString(strings.Repeat(" ", c.prefix))
		b.WriteString(c.text)
	}
	right := ""
	if width >= 104 {
		right = m.theme.mutedStyle().Render("EXACT BYTES  ·  NO RE-MARSHAL")
	}
	return joinLRKeepLeft(b.String(), right, width)
}

func (m model) renderTabRule(l layout) string {
	cells := m.tabCells()
	start := 0
	activeW := 0
	for _, c := range cells {
		start += c.prefix
		if c.active {
			activeW = c.width
			break
		}
		start += c.width
	}
	if activeW == 0 {
		return m.rule(l.contentW, m.theme.hairline)
	}
	return padLine(strings.Repeat(" ", start)+m.theme.bold(m.theme.sectionColor(m.section)).Render(
		strings.Repeat("━", activeW)), l.contentW)
}

// renderRail is the wide-screen section index. The active section gets
// a page number in display type; the other two stay small. This is the
// one place the interface raises its voice.
//
// Tall screens get a status block under every section. Short ones get
// a compact list with one status glyph per row, because clipping the
// last section is worse than losing a key count.
func (m model) renderRail(width, height int) []string {
	lines := m.railLines(width, true)
	if len(lines) > height {
		lines = m.railLines(width, false)
	}
	return fitAndPad(lines, width, height)
}

func (m model) railLines(width int, full bool) []string {
	const pad = 1
	inner := width - pad
	lines := make([]string, 0, 24)

	number := renderBlock(displayNumber(int(m.section)+1), m.theme.bold(m.theme.sectionColor(m.section)))
	for _, line := range number {
		lines = append(lines, strings.Repeat(" ", pad)+padLine(line, inner))
	}
	lines = append(lines, "")
	activeLabel := m.section.String()
	if m.sectionError(m.section) != nil {
		activeLabel += " ✗"
	}
	lines = append(lines, strings.Repeat(" ", pad)+m.theme.bold(m.theme.ink).Render(activeLabel))
	lines = append(lines, strings.Repeat(" ", pad)+m.theme.fg(m.theme.sectionColor(m.section)).Render(
		strings.Repeat("━", min(len(activeLabel), max(4, inner)))))
	lines = append(lines, "")

	for _, line := range m.sectionFacts(m.section) {
		lines = append(lines, strings.Repeat(" ", pad)+line)
	}
	lines = append(lines, "")

	for s := sectionHeader; s <= sectionSignature; s++ {
		if s == m.section {
			continue
		}
		if full {
			label := s.String()
			if m.sectionError(s) != nil {
				label += " ✗"
			}
			nav := m.theme.mutedStyle().Render(fmt.Sprintf("%02d", int(s)+1)) + "  " +
				m.theme.fg(m.theme.ink).Render(label)
			lines = append(lines, strings.Repeat(" ", pad)+padLine(nav, inner))
			for _, fact := range m.sectionFacts(s) {
				lines = append(lines, strings.Repeat(" ", pad+4)+fact)
			}
			lines = append(lines, "")
			continue
		}
		glyph, color := m.sectionGlyph(s)
		nav := m.theme.fg(color).Render(glyph) + " " +
			m.theme.mutedStyle().Render(fmt.Sprintf("%02d", int(s)+1)) + " " +
			m.theme.fg(m.theme.ink).Render(s.String())
		lines = append(lines, strings.Repeat(" ", pad)+padLine(nav, inner))
		lines = append(lines, "")
	}
	// The trailing blank is only useful when something follows it. Drop
	// it so the richest rail fits in exactly 24 body rows at 120x32.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// sectionGlyph is the compact rail's one-character verdict.
func (m model) sectionGlyph(s section) (string, lipgloss.TerminalColor) {
	if s == sectionSignature {
		if m.headerErr != nil || m.payloadErr != nil {
			return "!", m.theme.warning
		}
		return sigGlyph(m.sigState), m.sigColor()
	}
	if m.sectionError(s) != nil {
		return "✗", m.theme.danger
	}
	return "✓", m.theme.success
}

// sectionFacts is the small status block under a section name. It is
// intentionally short: the editor and the audit page carry the detail.
func (m model) sectionFacts(s section) []string {
	label := m.theme.mutedStyle()
	switch s {
	case sectionHeader, sectionPayload:
		if err := m.sectionError(s); err != nil {
			return []string{
				label.Render("JSON"),
				m.theme.fg(m.theme.danger).Render("invalid"),
				label.Render("fix syntax"),
			}
		}
		kind := "JSON"
		if s == sectionHeader {
			kind = "OBJECT"
		}
		facts := []string{
			label.Render(kind),
			m.theme.fg(m.theme.success).Render("valid"),
		}
		if m.parsedOK {
			keys := len(m.parsed.Header)
			if s == sectionPayload {
				keys = len(m.parsed.Payload)
			}
			facts = append(facts, label.Render(fmt.Sprintf("%d key%s", keys, plural(keys))))
		}
		if m.sectionDirty(s) {
			facts = append(facts, m.theme.fg(m.theme.warning).Render("edited"))
		} else if m.loaded {
			facts = append(facts, label.Render("original"))
		}
		return facts
	case sectionSignature:
		state := strings.ToLower(m.sigState.label())
		color := m.sigColor()
		if m.headerErr != nil || m.payloadErr != nil {
			state = "unavailable"
			color = m.theme.warning
		}
		return []string{
			label.Render("BASE64URL"),
			m.theme.fg(color).Render(state),
			label.Render(humanBytes(len(m.signature.Value()))),
		}
	}
	return nil
}

func (m model) sectionDirty(s section) bool {
	switch s {
	case sectionHeader:
		return m.headerDirty
	case sectionPayload:
		return m.payloadDirty
	default:
		return false
	}
}

func (m model) renderFooter(width int) string {
	name, color := m.modeLabel()
	hints := m.footerHints(width)
	left := m.theme.fg(color).Render("●") + " " + m.theme.bold(color).Render(name)
	if hints != "" {
		left += "   " + m.theme.mutedStyle().Render(hints)
	}

	right := ""
	if m.toast != "" {
		right = m.theme.fg(m.toastColor()).Render(m.toast)
	} else if m.mode == modeEdit && width >= 96 {
		right = m.theme.mutedStyle().Render(m.cursorLabel())
	}
	return joinLR(left, right, width)
}

func (m model) modeLabel() (string, lipgloss.TerminalColor) {
	switch m.mode {
	case modeCommand:
		return "COMMAND", m.theme.warning
	case modeSign:
		return "SIGN", m.theme.header
	case modeAudit:
		return "AUDIT", m.auditColor()
	case modeHelp:
		return "HELP", m.theme.signature
	case modeToken:
		return "TOKEN", m.theme.signature
	default:
		return "EDIT", m.theme.sectionColor(m.section)
	}
}

func (m model) footerHints(width int) string {
	vim := "ctrl+hjkl move"
	if !m.vimKeys {
		vim = "arrows move"
	}
	switch m.mode {
	case modeCommand:
		if width >= 96 {
			return "i edit · s sign · y copy · f format · a audit · v token · ? help · q quit"
		}
		return "i edit · s sign · a audit · q quit"
	case modeSign:
		return "enter sign · ctrl+r reveal · esc cancel"
	case modeAudit, modeToken:
		copyHint := " · y copy"
		if m.currentToken == "" {
			copyHint = ""
		}
		if width >= 90 {
			return "↑↓ scroll · g/G top/bottom" + copyHint + " · esc close"
		}
		return "↑↓ scroll · esc close"
	case modeHelp:
		return "↑↓ scroll · esc close"
	}
	if width >= 96 {
		return vim + " · tab section · ctrl+s sign · ctrl+y copy · ctrl+o command"
	}
	if width >= 76 {
		return vim + " · tab section · ctrl+s sign · ctrl+y copy"
	}
	if width >= 70 {
		return vim + " · tab section · ctrl+s sign"
	}
	return strings.ReplaceAll(vim, " move", "") + " · tab · ctrl+s"
}

func (m model) toastColor() lipgloss.TerminalColor {
	switch m.toastK {
	case toastSuccess:
		return m.theme.success
	case toastWarning:
		return m.theme.warning
	case toastDanger:
		return m.theme.danger
	default:
		return m.theme.signature
	}
}

func (m model) cursorLabel() string {
	switch m.section {
	case sectionHeader, sectionPayload:
		line := 1
		col := 1
		if m.section == sectionHeader {
			line = m.header.Line() + 1
			col = m.header.LineInfo().ColumnOffset + 1
		} else {
			line = m.payload.Line() + 1
			col = m.payload.LineInfo().ColumnOffset + 1
		}
		return fmt.Sprintf("L%d  C%d", line, col)
	case sectionSignature:
		return fmt.Sprintf("POS %d", m.signature.Position()+1)
	}
	return ""
}
