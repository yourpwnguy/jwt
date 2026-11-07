package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/yourpwnguy/jwt/v2/internal/audit"
)

// render_audit.go turns audit.Report into the small inline summary and
// the full-screen report. The audit package stays presentation-free.

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

func (m model) auditColor() lipgloss.TerminalColor {
	if !m.auditOK {
		return m.theme.warning
	}
	if m.report.FailedCount() > 0 {
		return m.theme.danger
	}
	return m.theme.success
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

// renderAuditInline fits the summary and as many findings as maxLines
// allows. The caller decides how much vertical room the inspector has.
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

// auditFullContent is the scrollable report behind "a". It includes
// passed checks too, because a clean result is information.
func (m model) auditFullContent(width int) []string {
	sym := "✓"
	if !m.auditOK {
		sym = "!"
	} else if m.report.FailedCount() > 0 {
		sym = "✗"
	}
	lines := []string{
		m.theme.bold(m.auditColor()).Render("AUDIT  " + sym + " " + m.auditSummary()),
		"",
	}
	if !m.auditOK {
		lines = append(lines,
			m.theme.mutedStyle().Render("The audit is paused because the token is not valid JSON."),
			m.theme.mutedStyle().Render("Fix the header or payload editor and try again."),
		)
		for _, problem := range m.tokenProblems() {
			for _, line := range wrapPlain(problem, max(4, width-2)) {
				lines = append(lines, "  "+m.theme.fg(m.theme.danger).Render(line))
			}
		}
		return lines
	}

	for _, f := range m.sortedFindings() {
		color := m.severityColor(f)
		head := m.theme.bold(color).Render(m.severitySymbol(f)+" "+strings.ToUpper(string(f.Severity))) +
			"  " + m.theme.bold(m.theme.muted).Render(f.ID)
		lines = append(lines, ansi.Truncate(head, width, "…"))
		lines = append(lines, m.theme.mutedStyle().Render("  "+f.Title))
		for _, d := range wrapPlain(f.Detail, max(4, width-2)) {
			lines = append(lines, "  "+m.theme.mutedStyle().Render(d))
		}
		lines = append(lines, "")
	}
	lines = append(lines,
		m.theme.mutedStyle().Render("Offline structural checks. This does not verify the signature."),
	)
	return lines
}

// wrapPlain wraps unstyled text with lipgloss and returns lines.
func wrapPlain(text string, width int) []string {
	if width < 4 {
		width = 4
	}
	return strings.Split(lipgloss.NewStyle().Width(width).Render(text), "\n")
}

// fitLines truncates styled lines to width and to maxLines. Callers use
// it so a long audit finding cannot push the layout around.
func fitLines(lines []string, width, maxLines int) []string {
	if width < 4 {
		width = 4
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, width, "…")
	}
	return lines
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
