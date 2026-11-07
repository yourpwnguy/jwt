package tui

// canvas.go is the page geometry layer. It pads, truncates, joins, and
// fits lines so every screen ends up exactly width by height. The card
// design paints no page background, so finish() is only a safety net:
// it pads the frame and leaves every empty cell transparent.
//
// The helpers here are deliberately ANSI-aware. lipgloss.Width and
// ansi.Truncate understand escape sequences, so styled text can be
// measured without stripping its color first.

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func padLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) > width {
		return ansi.Truncate(line, width, "…")
	}
	if gap := width - lipgloss.Width(line); gap > 0 {
		return line + strings.Repeat(" ", gap)
	}
	return line
}

// joinLR puts left and right on one line and pads the gap. If space is
// short, the left side truncates first because the right side usually
// carries status the reader needs.
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
// side. The masthead uses it so a long status never hides the tabs.
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

// fitAndPad is the last line of defence: no renderer can push the key
// line off screen, and every line is exactly width cells wide.
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
		out = append(out, padLine(line, width))
	}
	return out
}

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

// wrapPlain wraps unstyled text with lipgloss and returns lines. Callers
// color the lines after wrapping, so ANSI codes never count toward the
// width.
func wrapPlain(text string, width int) []string {
	if width < 4 {
		width = 4
	}
	return strings.Split(lipgloss.NewStyle().Width(width).Render(text), "\n")
}

// finish pads the page to the terminal size and joins the lines. It
// intentionally does not paint a background: the card design lets the
// terminal background through, including the outer rows and every gap.
func (m model) finish(s string) string {
	lines := fitAndPad(strings.Split(s, "\n"), m.width, m.height)
	return strings.Join(lines, "\n")
}

// rule draws a hairline using the muted text color. The card design
// keeps rules inside the panels rather than across the whole page.
func (m model) rule(width int) string {
	if width <= 0 {
		return ""
	}
	return m.theme.mutedStyle().Render(strings.Repeat("─", width))
}
