package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// theme.go owns the palette and the small style vocabulary used by the
// editor. The look is an editorial page, not a dashboard: a painted
// paper or ink canvas, one accent per token section, hairline rules, and
// a lot of empty space.
//
// The three section colors still come from the README logo: amber for
// the header, coral for the payload, cyan for the signature. They are
// used as ink, not as filled blocks, so the screen stays quiet. Fills
// are reserved for rare moments: a severity tag and the signature stamp.
//
// Text colors are adaptive. Light terminals get darker accents, dark
// terminals get the bright logo colors. Nothing uses faint or dim SGR,
// because that disappears on some light themes.
type theme struct {
	mode ThemeMode

	canvas  lipgloss.TerminalColor
	surface lipgloss.TerminalColor
	edge    lipgloss.TerminalColor

	ink      lipgloss.TerminalColor
	muted    lipgloss.TerminalColor
	hairline lipgloss.TerminalColor

	header    lipgloss.TerminalColor
	payload   lipgloss.TerminalColor
	signature lipgloss.TerminalColor
	success   lipgloss.TerminalColor
	warning   lipgloss.TerminalColor
	danger    lipgloss.TerminalColor
}

// Fill colors are the logo colors at full strength. They always carry
// dark ink, so contrast does not depend on the terminal background.
var (
	fillHeader    = lipgloss.Color("#FFC107")
	fillSignature = lipgloss.Color("#3DD0F2")
	fillSuccess   = lipgloss.Color("#7EE787")
	fillWarning   = lipgloss.Color("#FFA657")
	fillDanger    = lipgloss.Color("#FF6B6B")
	inkOnFill     = lipgloss.Color("#141414")
)

func newTheme(mode ThemeMode) theme {
	if mode == "" {
		mode = ThemeAuto
	}
	return theme{
		mode:    mode,
		canvas:  adaptive("#F6F1E7", "#0D1014", mode),
		surface: adaptive("#EFE8DA", "#141A21", mode),
		edge:    adaptive("#E1D7C5", "#1E2630", mode),

		ink:      adaptive("#1B1916", "#EDE7DA", mode),
		muted:    adaptive("#6B6357", "#A79E8E", mode),
		hairline: adaptive("#CEC3B0", "#2C343E", mode),

		header:    adaptive("#946000", "#FFC107", mode),
		payload:   adaptive("#B04932", "#F48C6C", mode),
		signature: adaptive("#007084", "#3DD0F2", mode),
		success:   adaptive("#1A7F37", "#7EE787", mode),
		warning:   adaptive("#985100", "#FFA657", mode),
		danger:    adaptive("#B3261E", "#FF6B6B", mode),
	}
}

// adaptive picks a fixed color when the user forced a theme, and an
// AdaptiveColor when they left it on auto. Lip Gloss resolves the auto
// value from the terminal background.
func adaptive(light, dark string, mode ThemeMode) lipgloss.TerminalColor {
	switch mode {
	case ThemeDark:
		return lipgloss.Color(dark)
	case ThemeLight:
		return lipgloss.Color(light)
	default:
		return lipgloss.AdaptiveColor{Light: light, Dark: dark}
	}
}

func (t theme) sectionColor(s section) lipgloss.TerminalColor {
	switch s {
	case sectionHeader:
		return t.header
	case sectionPayload:
		return t.payload
	case sectionSignature:
		return t.signature
	default:
		return t.muted
	}
}

// cursorLine is a barely-there wash under the active line. It is a
// background tint, never a text color, so JSON stays at full contrast.
func (t theme) cursorLine(s section) lipgloss.TerminalColor {
	switch s {
	case sectionHeader:
		return adaptive("#FBF0CE", "#272009", t.mode)
	case sectionPayload:
		return adaptive("#FBE3DB", "#271714", t.mode)
	default:
		return adaptive("#DDF5FA", "#07232B", t.mode)
	}
}

func (t theme) fg(color lipgloss.TerminalColor) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(color)
}

func (t theme) bold(color lipgloss.TerminalColor) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(color).Bold(true)
}

func (t theme) mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.muted)
}

// chip is the only filled label in the main layout. Severity tags and
// the signed stamp use it; everything else is ink on paper.
func (t theme) chip(label string, fill lipgloss.TerminalColor) string {
	return lipgloss.NewStyle().
		Background(fill).
		Foreground(inkOnFill).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

// keycap is a quiet key label for the help page and the sign form. The
// background is a paper shade, not a bright accent, so a page full of
// keys still reads as text.
func (t theme) keycap(key string) string {
	return lipgloss.NewStyle().
		Background(t.edge).
		Foreground(t.ink).
		Bold(true).
		Padding(0, 1).
		Render(key)
}

// tracked adds a little air between letters. It is used once, in the
// masthead, where a mono font needs help to feel like a masthead.
func tracked(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r != ' ' {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}
