package tui

// canvas.go is the page geometry layer. It pads, truncates, joins, and
// paints lines so every screen ends up exactly width by height. The
// palette is applied at the very end: one background wash behind the
// whole page, with panel backgrounds layered on top.
//
// The helpers here are deliberately ANSI-aware. lipgloss.Width and
// ansi.Truncate understand escape sequences, so styled text can be
// measured without stripping its color first.

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
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
// side. The tab row uses it so a long caption never hides a tab.
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

// fitAndPad is the last line of defence: no renderer can push the
// footer off screen, and every line is exactly width cells wide.
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

// bgSequence returns just the SGR prefix for a background color. Lip
// Gloss does not expose the escape sequence, and wrapping a line in a
// background style is not enough: an inner reset clears the background
// for every cell after it. We need the raw sequence so paintBG can put
// it back after each reset.
func bgSequence(c lipgloss.TerminalColor) string {
	if lipgloss.ColorProfile() == termenv.Ascii {
		return ""
	}
	probe := lipgloss.NewStyle().Background(c).Render(" ")
	if i := strings.LastIndexByte(probe, ' '); i >= 0 {
		return probe[:i]
	}
	return ""
}

// paintBG forces one base background across a finished line. Inner
// styles that set their own background are preserved because their
// sequence follows the base sequence. The final reset keeps the next
// line independent.
func paintBG(line, seq string) string {
	if seq == "" {
		return line
	}
	line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+seq)
	line = strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+seq)
	line = strings.ReplaceAll(line, "\x1b[49m", "\x1b[49m"+seq)
	return seq + line + "\x1b[0m"
}

// finish pads the page to the terminal size, paints the canvas, and
// joins the lines. Painting last means every gap, including the outer
// margins, gets the same paper or ink color.
func (m model) finish(s string) string {
	lines := fitAndPad(strings.Split(s, "\n"), m.width, m.height)
	seq := bgSequence(m.theme.canvas)
	for i, line := range lines {
		lines[i] = paintBG(line, seq)
	}
	return strings.Join(lines, "\n")
}

func (m model) rule(width int, color lipgloss.TerminalColor) string {
	if width <= 0 {
		return ""
	}
	return m.theme.fg(color).Render(strings.Repeat("─", width))
}

// frame puts the horizontal margins around a finished content line.
func frame(line string, margin, contentW int) string {
	return strings.Repeat(" ", margin) + padLine(line, contentW) + strings.Repeat(" ", margin)
}

// displayNumber draws a big page number in a 5x5 block font. It is the
// one piece of display type in the interface: the active section in the
// wide rail. The shapes are deliberately simple, not a full typeface.
func displayNumber(n int) []string {
	digits := map[int][5]string{
		0: {"█████", "██ ██", "██ ██", "██ ██", "█████"},
		1: {"   ██", "  ███", "   ██", "   ██", " █████"},
		2: {"█████", "   ██", "█████", "██   ", "█████"},
		3: {"█████", "   ██", "█████", "   ██", "█████"},
	}
	first := n / 10
	second := n % 10
	a, okA := digits[first]
	b, okB := digits[second]
	if !okA || !okB {
		return []string{"", "", "", "", ""}
	}
	out := make([]string, 5)
	for i := 0; i < 5; i++ {
		out[i] = a[i] + " " + b[i]
	}
	return out
}

// renderBlock renders each line of a multi-line block in one style and
// keeps the block width-clean.
func renderBlock(lines []string, style lipgloss.Style) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = style.Render(line)
	}
	return out
}
