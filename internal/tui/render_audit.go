package tui

// render_audit.go turns audit.Report into the page behind "a". The
// audit package stays presentation-free: this file owns the score, the
// severity tags, and the language.
//
// The page leads with the verdict and one dot per rule, then explains
// only the failures. Passing checks get a compact list at the bottom so
// a clean token still feels like a result, not an empty screen.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/yourpwnguy/jwt/v2/internal/audit"
)

func (m model) sortedFindings() []audit.Finding {
	out := append([]audit.Finding(nil), m.report.Findings...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Pass != out[j].Pass {
			return !out[i].Pass
		}
		return out[i].Severity.Rank() > out[j].Severity.Rank()
	})
	return out
}

func (m model) auditColor() lipgloss.TerminalColor {
	if !m.auditOK {
		return m.theme.warning
	}
	if m.report.FailedCount() > 0 {
		return m.theme.danger
	}
	return m.theme.success
}

func (m model) auditSummary() string {
	if !m.auditOK {
		return "paused"
	}
	failed := m.report.FailedCount()
	if failed == 0 {
		return fmt.Sprintf("%d/%d passed", len(m.report.Findings), len(m.report.Findings))
	}
	return fmt.Sprintf("%d issue%s", failed, plural(failed))
}

func (m model) auditHeader() string {
	label := m.theme.bold(m.auditColor()).Render("AUDIT")
	sym := "✓"
	if !m.auditOK {
		sym = "!"
	} else if m.report.FailedCount() > 0 {
		sym = "✗"
	}
	return label + "  " + m.theme.fg(m.auditColor()).Render(sym+" "+m.auditSummary())
}

func (m model) severityColor(f audit.Finding) lipgloss.TerminalColor {
	if f.Pass {
		return m.theme.success
	}
	switch f.Severity {
	case audit.SeverityCritical, audit.SeverityHigh:
		return m.theme.danger
	case audit.SeverityMedium:
		return m.theme.warning
	default:
		return m.theme.signature
	}
}

func (m model) severitySymbol(f audit.Finding) string {
	if f.Pass {
		return "✓"
	}
	if f.Severity == audit.SeverityCritical || f.Severity == audit.SeverityHigh {
		return "✗"
	}
	return "!"
}

func (m model) severityLabel(f audit.Finding) string {
	return m.theme.bold(m.severityColor(f)).Render(strings.ToUpper(string(f.Severity)))
}

// auditDots is one dot per rule. It stays because it is the fastest
// way to see that a token was actually checked; it is just colored
// text now, with no filled background.
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

// renderAuditInline fits the summary and as many findings as maxLines
// allows. The inspector card decides how much vertical room the audit
// gets, so the full report stays behind the audit command.
func (m model) renderAuditInline(width, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	lines := []string{m.auditHeader()}
	if !m.auditOK {
		lines = append(lines, m.theme.mutedStyle().Render("fix the JSON to run the checks"))
		return fitLines(lines, width, maxLines)
	}

	findings := m.sortedFindings()
	if m.report.FailedCount() == 0 {
		lines = append(lines, m.theme.fg(m.theme.success).Render("✓ all checks passed"))
		ids := findingIDs(findings)
		for _, line := range wrapPlain(ids, width) {
			lines = append(lines, m.theme.mutedStyle().Render(line))
		}
		return fitLines(lines, width, maxLines)
	}

	for _, f := range findings {
		if f.Pass {
			continue
		}
		if len(lines) >= maxLines {
			break
		}
		lines = append(lines, m.findingHeadline(f, width))
		detail := wrapPlain(f.Detail, max(4, width-2))
		for _, d := range detail {
			if len(lines) >= maxLines {
				break
			}
			lines = append(lines, m.theme.mutedStyle().Render("  "+d))
		}
	}

	passed := len(findings) - m.report.FailedCount()
	if passed > 0 && len(lines) < maxLines {
		lines = append(lines, m.theme.fg(m.theme.success).Render(fmt.Sprintf("✓ %d checks passed", passed)))
		ids := passedIDs(findings)
		for _, line := range wrapPlain(ids, width) {
			if len(lines) >= maxLines {
				break
			}
			lines = append(lines, m.theme.mutedStyle().Render(line))
		}
	}
	return fitLines(lines, width, maxLines)
}

func (m model) findingHeadline(f audit.Finding, width int) string {
	color := m.severityColor(f)
	sym := m.severitySymbol(f)
	sev := fmt.Sprintf("%-8s", strings.ToUpper(string(f.Severity)))
	head := m.theme.bold(color).Render(sym + " " + sev)
	line := head + " " + m.theme.fg(m.theme.muted).Render(f.ID)
	return ansi.Truncate(line, width, "…")
}

func findingIDs(findings []audit.Finding) string {
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		ids = append(ids, f.ID)
	}
	return strings.Join(ids, " · ")
}

func passedIDs(findings []audit.Finding) string {
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		if f.Pass {
			ids = append(ids, f.ID)
		}
	}
	return strings.Join(ids, " · ")
}

// auditFullContent is the scrollable report. The caller gives it the
// full content width, and the viewport handles the rest.
func (m model) auditFullContent(width int) []string {
	if !m.auditOK {
		return m.pausedAuditContent(width)
	}

	failed := m.report.FailedCount()
	summary := m.theme.bold(m.theme.success).Render(fmt.Sprintf("✓ %d/%d PASSED",
		len(m.report.Findings), len(m.report.Findings)))
	if failed > 0 {
		summary = m.theme.bold(m.theme.danger).Render(
			fmt.Sprintf("✗ %d ISSUE%s", failed, strings.ToUpper(plural(failed))))
	}
	lines := []string{
		joinLR(summary, m.theme.mutedStyle().Render(fmt.Sprintf("%d CHECKS", len(m.report.Findings))), width),
		m.auditDots(width),
		"",
	}

	findings := m.sortedFindings()
	if failed == 0 {
		lines = append(lines,
			m.theme.bold(m.theme.success).Render("✓ CLEAN"),
			m.theme.mutedStyle().Render("No structural issues found by the offline rules."),
			"",
		)
	}

	for _, f := range findings {
		if f.Pass {
			continue
		}
		head := m.severityLabel(f) + "  " + m.theme.bold(m.theme.header).Render(f.ID)
		lines = append(lines, head)
		lines = append(lines, m.theme.mutedStyle().Render("  "+f.Title))
		for _, detail := range wrapPlain(f.Detail, max(4, width-2)) {
			lines = append(lines, m.theme.mutedStyle().Render("  "+detail))
		}
		lines = append(lines, "")
	}

	lines = append(lines, m.theme.bold(m.theme.header).Render("PASSED"))
	lines = append(lines, m.rule(width))
	for _, f := range findings {
		if !f.Pass {
			continue
		}
		line := m.theme.fg(m.theme.success).Render("✓ ") + m.theme.fg(m.theme.header).Render(f.ID) +
			m.theme.mutedStyle().Render("  "+f.Title)
		for _, wrapped := range wrapPlain(line, width) {
			lines = append(lines, wrapped)
		}
	}
	lines = append(lines, "")
	lines = append(lines, m.theme.mutedStyle().Render("Offline structural checks. This does not verify the signature."))
	return lines
}

func (m model) pausedAuditContent(width int) []string {
	lines := []string{
		m.theme.bold(m.theme.warning).Render("AUDIT PAUSED"),
		m.theme.mutedStyle().Render("The token is not valid JSON, so the rules cannot run."),
		"",
	}
	for _, problem := range m.tokenProblems() {
		for _, line := range wrapPlain(problem, max(4, width-2)) {
			lines = append(lines, m.theme.fg(m.theme.danger).Render(line))
		}
	}
	lines = append(lines, "",
		m.theme.mutedStyle().Render("Fix the header or payload editor and the audit resumes automatically."),
	)
	return lines
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
