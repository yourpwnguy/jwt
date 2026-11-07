package tui

// view.go composes the page. The main screen is a masthead, a section
// index, a borderless editor, and either a specimen panel or a short
// token strip, then the footer. Every mode returns through finish(),
// which paints the canvas and guarantees an exact terminal-sized frame.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if m.width < minWidth || m.height < minHeight {
		return m.finish(m.renderTooSmall())
	}
	switch m.mode {
	case modeSign:
		return m.finish(m.renderSign())
	case modeAudit, modeHelp, modeToken:
		return m.finish(m.renderOverlay())
	default:
		return m.finish(m.renderMain())
	}
}

func (m model) renderMain() string {
	l := m.layout()
	lines := make([]string, 0, l.height)

	if l.topPad > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, frame(m.renderMasthead(l.contentW), l.margin, l.contentW))
	lines = append(lines, frame(m.rule(l.contentW, m.theme.hairline), l.margin, l.contentW))

	if l.showTabs {
		lines = append(lines, frame(m.renderTabs(l.contentW), l.margin, l.contentW))
		lines = append(lines, frame(m.renderTabRule(l), l.margin, l.contentW))
	}
	if l.gap > 0 {
		lines = append(lines, "")
	}

	for _, line := range m.renderBody(l) {
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

// renderBody joins the panes. Wide screens get the page-number rail,
// medium screens get editor plus specimen, and narrow screens stack a
// token strip under the editor.
func (m model) renderBody(l layout) []string {
	switch l.mode {
	case layoutWide:
		rail := m.renderRail(l.railW, l.bodyH)
		editor := m.renderEditorPane(l.editorW, l.bodyH)
		panel := m.renderInspector(l.panelW, l.bodyH)
		return joinColumns(l.bodyH,
			column{width: l.railW, lines: rail},
			column{width: l.editorW, lines: editor},
			column{width: l.panelW, lines: panel},
		)
	case layoutMedium:
		editor := m.renderEditorPane(l.editorW, l.bodyH)
		panel := m.renderInspector(l.panelW, l.bodyH)
		return joinColumns(l.bodyH,
			column{width: l.editorW, lines: editor},
			column{width: l.panelW, lines: panel},
		)
	default:
		editor := m.renderEditorPane(l.editorW, l.stackEditorH)
		strip := m.renderTokenStrip(l.panelW, l.stackStripH)
		out := make([]string, 0, l.bodyH)
		out = append(out, editor...)
		out = append(out, "")
		out = append(out, strip...)
		return out
	}
}

type column struct {
	width int
	lines []string
}

// joinColumns lays out already-padded panes with the page gutter
// between them. The output is exactly the sum of the widths plus the
// gutters, which is what the body arithmetic promised.
func joinColumns(height int, cols ...column) []string {
	out := make([]string, height)
	for row := 0; row < height; row++ {
		var b strings.Builder
		for i, col := range cols {
			if i > 0 {
				b.WriteString(strings.Repeat(" ", gutter))
			}
			line := ""
			if row < len(col.lines) {
				line = col.lines[row]
			}
			b.WriteString(padLine(line, col.width))
		}
		out[row] = b.String()
	}
	return out
}

func (m model) renderTooSmall() string {
	msg := m.theme.bold(m.theme.ink).Render("JWT.") + "\n\n" +
		m.theme.bold(m.theme.payload).Render("Terminal too small.") + "\n\n" +
		m.theme.mutedStyle().Render(fmt.Sprintf("This page needs at least %dx%d.", minWidth, minHeight)) + "\n" +
		m.theme.mutedStyle().Render(fmt.Sprintf("The window is %dx%d.", m.width, m.height)) + "\n\n" +
		m.theme.mutedStyle().Render("Resize it, or press ctrl+c to quit.")
	block := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	return block
}
