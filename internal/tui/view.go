package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// view.go composes the screen from the first token studio layout: a top
// status bar, a tab row, an editor card beside the token inspector, and
// a key line. On narrow terminals the inspector stacks under the
// editor. Every mode ends in finish(), which pads the frame to the
// terminal size without painting a background.

func (m model) View() string {
	if m.width < minWidth || m.height < minHeight {
		return m.finish(m.renderTooSmall())
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
	return m.finish(s)
}

func (m model) renderMain() string {
	l := m.layout()
	var body string
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
