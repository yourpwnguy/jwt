package tui

// view_inspector.go is the specimen column: the exact encoded token,
// the facts you usually need, and the audit in miniature. On narrow
// terminals the same content collapses into a short strip under the
// editor so the token never disappears.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

func (m model) renderInspector(totalW, totalH int) []string {
	if totalW < 6 || totalH < 1 {
		return nil
	}
	textW := totalW - 5
	content := m.inspectorContent(textW, totalH)
	content = fitAndPad(content, textW, totalH)

	seq := bgSequence(m.theme.surface)
	rule := m.theme.fg(m.theme.hairline).Render("│")
	lines := make([]string, totalH)
	for i, line := range content {
		lines[i] = paintBG(padLine(rule+"  "+line+"  ", totalW), seq)
	}
	return lines
}

func (m model) inspectorContent(width, height int) []string {
	if height <= 0 {
		return nil
	}
	label := "ENCODED TOKEN"
	if m.currentToken == "" {
		label = "LAST VALID TOKEN"
	}
	header := joinLR(
		m.theme.bold(m.theme.signature).Render(label),
		m.theme.mutedStyle().Render(humanBytes(len(m.previewToken()))),
		width,
	)

	// Tiny panes get the token and nothing else. A label is less useful
	// than the first characters of the thing you came to read.
	if height <= 3 {
		return fitLines(m.tokenDisplayLines(width, height), width, height)
	}

	if height < 10 {
		lines := []string{header}
		lines = append(lines, m.tokenDisplayLines(width, height-1)...)
		return fitLines(lines, width, height)
	}

	facts := m.inspectorFacts(width)
	const fixed = 8 // header, blank, blank, rule, and four facts
	tokenBudget := height - fixed
	if tokenBudget < 1 {
		tokenBudget = 1
	}
	lines := []string{header, ""}
	lines = append(lines, m.tokenDisplayLines(width, tokenBudget)...)
	lines = append(lines, "", m.rule(width, m.theme.hairline))
	lines = append(lines, facts...)

	// Keep the audit attached even when the page is tight. A rule is
	// enough separation; the blank line is a luxury only taller panes
	// can afford.
	if remaining := height - len(lines); remaining >= 7 {
		lines = append(lines, "", m.rule(width, m.theme.hairline))
		lines = append(lines, m.auditPanel(width, remaining-2)...)
	} else if remaining >= 3 {
		lines = append(lines, m.rule(width, m.theme.hairline))
		lines = append(lines, m.auditPanel(width, remaining-1)...)
	}
	return fitLines(lines, width, height)
}

func (m model) inspectorFacts(width int) []string {
	alg := m.alg
	if alg == "" {
		alg = "missing"
	}
	state := strings.ToLower(m.sigState.label())
	sigColor := m.sigColor()
	if m.headerErr != nil || m.payloadErr != nil {
		state = "unavailable"
		sigColor = m.theme.warning
	} else if m.sigErr != nil {
		state = "invalid encoding"
	}
	rows := [][2]string{
		{"ALG", alg},
		{"SIGNATURE", state},
		{"EXPIRES", m.compactExp()},
		{"SIZE", humanBytes(len(m.previewToken()))},
	}
	colors := []lipgloss.TerminalColor{
		m.theme.header, sigColor, m.expColor(), m.theme.muted,
	}
	out := make([]string, 0, len(rows))
	for i, row := range rows {
		label := m.theme.mutedStyle().Render(fmt.Sprintf("%-10s", row[0]))
		value := ansi.Truncate(row[1], max(4, width-10), "…")
		out = append(out, label+m.theme.fg(colors[i]).Render(value))
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

// compactExp keeps the panel scannable. CheckTime produces a full
// sentence with a date in parentheses; the panel has room for the
// verdict, not the footnote.
func (m model) compactExp() string {
	if m.expMessage == "" {
		return "not available"
	}
	if i := strings.Index(m.expMessage, " ("); i > 0 {
		msg := m.expMessage[:i]
		if m.expStatus == jwt.TemporalValid {
			msg = strings.TrimPrefix(msg, "expires ")
		}
		return msg
	}
	switch m.expStatus {
	case jwt.TemporalNoExp:
		return "none"
	case jwt.TemporalExpired:
		return "expired"
	case jwt.TemporalNotYetValid:
		return "not yet valid"
	default:
		return m.expMessage
	}
}

// auditPanel is the miniature report under the facts. The dots are one
// per rule, so a clean token has a visible row of green, and a risky
// one points at the exact rule that failed.
func (m model) auditPanel(width, height int) []string {
	if height <= 0 {
		return nil
	}
	if !m.auditOK {
		lines := []string{
			m.theme.bold(m.theme.warning).Render("AUDIT  ! PAUSED"),
			m.theme.mutedStyle().Render("fix the JSON to run the checks"),
		}
		for _, problem := range m.tokenProblems() {
			if len(lines) >= height {
				break
			}
			for _, line := range wrapPlain(problem, max(4, width-2)) {
				if len(lines) >= height {
					break
				}
				lines = append(lines, m.theme.fg(m.theme.danger).Render(line))
			}
		}
		return fitLines(lines, width, height)
	}

	failed := m.report.FailedCount()
	summary := fmt.Sprintf("✓ %d/%d PASSED", len(m.report.Findings), len(m.report.Findings))
	if failed > 0 {
		summary = fmt.Sprintf("✗ %d ISSUE%s", failed, strings.ToUpper(plural(failed)))
	}
	lines := []string{
		joinLR(
			m.theme.bold(m.auditColor()).Render("AUDIT"),
			m.theme.fg(m.auditColor()).Render(summary),
			width,
		),
		m.auditDots(width),
	}
	if failed > 0 {
		first := m.sortedFindings()[0]
		head := m.theme.bold(m.severityColor(first)).Render(
			m.severitySymbol(first)+" "+strings.ToUpper(string(first.Severity))) +
			"  " + m.theme.fg(m.theme.danger).Render(first.ID)
		lines = append(lines, ansi.Truncate(head, width, "…"))
		for _, line := range wrapPlain(first.Detail, max(4, width-2)) {
			if len(lines) >= height {
				break
			}
			lines = append(lines, m.theme.mutedStyle().Render(line))
		}
	} else {
		lines = append(lines, m.theme.fg(m.theme.success).Render("✓ all checks passed"))
	}
	return fitLines(lines, width, height)
}

func (m model) auditDots(width int) string {
	findings := m.report.Findings
	if len(findings) == 0 {
		return ""
	}
	sep := ""
	if width >= 2*len(findings)-1 {
		sep = " "
	}
	parts := make([]string, 0, len(findings))
	for _, f := range findings {
		parts = append(parts, m.theme.fg(m.severityColor(f)).Render("●"))
	}
	return strings.Join(parts, sep)
}

func (m model) renderTokenStrip(totalW, totalH int) []string {
	if totalW < 6 || totalH < 1 {
		return nil
	}
	const pad = 2
	textW := totalW - 2*pad
	label := "TOKEN"
	if m.currentToken == "" {
		label = "LAST VALID TOKEN"
	}
	header := joinLR(
		m.theme.bold(m.theme.signature).Render(label),
		m.theme.mutedStyle().Render(humanBytes(len(m.previewToken()))),
		textW,
	)
	budget := totalH - 2
	if budget < 1 {
		budget = 1
	}
	lines := []string{header}
	lines = append(lines, m.tokenDisplayLines(textW, budget)...)
	lines = append(lines, m.stripMeta(textW))
	lines = fitAndPad(lines, textW, totalH)

	seq := bgSequence(m.theme.surface)
	for i, line := range lines {
		lines[i] = paintBG(padLine(strings.Repeat(" ", pad)+line+strings.Repeat(" ", pad), totalW), seq)
	}
	return lines
}

func (m model) stripMeta(width int) string {
	alg := m.alg
	if alg == "" {
		alg = "NO ALG"
	}
	auditText := fmt.Sprintf("%d/%d", len(m.report.Findings), len(m.report.Findings))
	auditColor := m.theme.success
	if !m.auditOK {
		auditText = "PAUSED"
		auditColor = m.theme.warning
	} else if m.report.FailedCount() > 0 {
		auditText = fmt.Sprintf("%d ISSUE%s", m.report.FailedCount(), strings.ToUpper(plural(m.report.FailedCount())))
		auditColor = m.theme.danger
	}
	left := m.theme.fg(m.theme.header).Render(alg) + "  ·  " +
		m.theme.fg(m.sigColor()).Render(m.sigLabel()) + "  ·  " +
		m.theme.fg(auditColor).Render(auditText)
	return joinLR(left, "", width)
}
