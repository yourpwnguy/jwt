package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// token.go wraps the encoded token without breaking ANSI styles. The
// three segments keep their section colors across line breaks, which is
// the main visual cue that a JWT really is three parts.

type tokenRun struct {
	text  string
	color lipgloss.TerminalColor
}

// tokenSegments returns the three visible pieces of the token in the
// inspector. It splits the assembled string instead of re-encoding, so
// the display cannot drift from the copied value.
func (m model) tokenSegments() (header, payload, signature string) {
	token := m.previewToken()
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return token, "", ""
	}
	return parts[0], parts[1], parts[2]
}

func (m model) previewToken() string {
	if m.currentToken != "" {
		return m.currentToken
	}
	return m.lastValidToken
}

func (m model) tokenRuns() []tokenRun {
	header, payload, sig := m.tokenSegments()
	return []tokenRun{
		{text: header, color: m.theme.header},
		{text: ".", color: m.theme.muted},
		{text: payload, color: m.theme.payload},
		{text: ".", color: m.theme.muted},
		{text: sig, color: m.theme.signature},
	}
}

// wrappedToken renders the whole token into lines of at most width. It
// is used by both the inspector and the full-token overlay.
func (m model) wrappedToken(width int) []string {
	return wrapRuns(m.tokenRuns(), width)
}

// tokenDisplayLines limits the inspector token to the space available.
// The full token is always one keypress away.
func (m model) tokenDisplayLines(width, maxLines int) []string {
	if m.previewToken() == "" {
		return []string{m.theme.mutedStyle().Render("waiting for valid JSON")}
	}
	lines := m.wrappedToken(width)
	if maxLines > 0 && len(lines) > maxLines {
		if maxLines == 1 {
			return lines[:1]
		}
		keep := maxLines - 1
		hidden := len(lines) - keep
		lines = append(lines[:keep], m.theme.mutedStyle().Render(
			fmt.Sprintf("… %d more lines, press v for the full token", hidden)))
	}
	return lines
}

// wrapRuns breaks styled runs into fixed-width lines. It measures with
// runewidth because base64 is ASCII, but a hand-edited signature can
// contain anything.
func wrapRuns(runs []tokenRun, width int) []string {
	if width < 4 {
		width = 4
	}
	var lines []string
	var b strings.Builder
	used := 0

	flush := func() {
		lines = append(lines, b.String())
		b.Reset()
		used = 0
	}

	for _, run := range runs {
		rest := run.text
		if rest == "" {
			continue
		}
		runWidth := runewidth.StringWidth(rest)

		// Keep short runs whole. Without this, a three-character
		// payload can split as "e" at the end of one line and "30" on
		// the next, which makes the token harder to scan.
		if used > 0 && runWidth <= width && runWidth > width-used {
			flush()
		}

		style := lipgloss.NewStyle().Foreground(run.color)
		for rest != "" {
			if used >= width {
				flush()
			}
			remain := width - used
			n := fitPrefix(rest, remain)
			if n == 0 {
				// A single rune wider than the line (extremely rare).
				// Put it on its own line rather than looping forever.
				if b.Len() > 0 {
					flush()
				}
				_, size := utf8.DecodeRuneInString(rest)
				if size == 0 {
					break
				}
				n = size
			}
			b.WriteString(style.Render(rest[:n]))
			used += runewidth.StringWidth(rest[:n])
			rest = rest[n:]
			if used >= width {
				flush()
			}
		}
	}
	if b.Len() > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}

// fitPrefix returns the byte length of the longest prefix that fits in
// width columns. It never splits a UTF-8 rune.
func fitPrefix(s string, width int) int {
	used := 0
	for i, r := range s {
		w := runewidth.RuneWidth(r)
		if used+w > width {
			return i
		}
		used += w
	}
	return len(s)
}
