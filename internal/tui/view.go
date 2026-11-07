package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// view.go composes the screen. The body is a two-column editor plus
// inspector on wide terminals, and a stacked editor plus token strip on
// narrow ones. Every mode ends in fitView, so a long line can never push
// the footer off screen.

func (m model) View() string {
	if m.width < minWidth || m.height < minHeight {
		return m.fitView(m.renderTooSmall())
	}
	var s string
	switch m.mode {
	case modeSign:
		s = m.renderSign()
	case modeAudit, modeHelp, modeToken:
		s = m.renderOverlay()
	default:
		s = m.renderMain()
	}
	return m.fitView(s)
}

func (m model) renderMain() string {
	l := m.layout()
	body := ""
	if l.wide {
		editor := m.renderEditorCard(l.leftWidth, l.bodyHeight)
		inspector := m.renderInspectorCard(l.rightWidth, l.bodyHeight)
		body = lipgloss.JoinHorizontal(lipgloss.Top, editor, strings.Repeat(" ", gutter), inspector)
	} else {
		editor := m.renderEditorCard(l.width, l.stackEditorH)
		inspector := m.renderInspectorCard(l.width, l.stackTokenH)
		body = lipgloss.JoinVertical(lipgloss.Left, editor, "", inspector)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderTopBar(),
		"",
		m.renderTabs(),
		body,
		"",
		m.renderFooter(),
	)
}

func (m model) renderTooSmall() string {
	msg := m.theme.bold(m.theme.header).Render("JWT token studio") + "\n\n" +
		m.theme.fg(m.theme.warning).Render("Terminal too small.") + "\n" +
		m.theme.mutedStyle().Render(fmt.Sprintf("Current %dx%d, need at least %dx%d.", m.width, m.height, minWidth, minHeight)) + "\n\n" +
		m.theme.mutedStyle().Render("Resize the window, or press ctrl+c to quit.")
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
}

// joinLR puts left and right on one line and pads the gap. If space is
// short, the left side truncates first because the right side usually
// carries status the user needs.
func joinLR(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	rw := lipgloss.Width(right)
	if rw > width {
		right = ansi.Truncate(right, width, "…")
		rw = lipgloss.Width(right)
	}
	maxLeft := width - rw - 1
	if maxLeft < 0 {
		maxLeft = 0
	}
	if lipgloss.Width(left) > maxLeft {
		left = ansi.Truncate(left, maxLeft, "")
	}
	gap := width - lipgloss.Width(left) - rw
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + right
}

// joinLRKeepLeft keeps the left side intact and truncates the right
// side. The tab bar uses it so a long parse error never hides a tab.
func joinLRKeepLeft(left, right string, width int) string {
	if width <= 0 {
		return ""
	}
	lw := lipgloss.Width(left)
	if lw > width {
		return ansi.Truncate(left, width, "…")
	}
	maxRight := width - lw - 1
	if maxRight < 0 {
		maxRight = 0
	}
	if lipgloss.Width(right) > maxRight {
		right = ansi.Truncate(right, maxRight, "…")
	}
	gap := width - lw - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + right
}

func fitAndPad(lines []string, width, height int) []string {
	if height < 0 {
		height = 0
	}
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		if width <= 0 {
			line = ""
		} else if lipgloss.Width(line) > width {
			line = ansi.Truncate(line, width, "…")
		}
		out = append(out, line)
	}
	return out
}

func (m model) fitView(s string) string {
	lines := strings.Split(s, "\n")
	return strings.Join(fitAndPad(lines, m.width, m.height), "\n")
}
