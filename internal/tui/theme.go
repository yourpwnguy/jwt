package tui

import "github.com/charmbracelet/lipgloss"

// theme.go holds the logo palette and the few styles the editor uses.
//
// The brand colors come straight from the README logo: amber, coral,
// cyan. In the editor they map to header, payload, signature, so color
// tells you which part of the token you are looking at instead of just
// decorating the screen.
//
// Text colors are adaptive. Bright logo colors are used for filled
// badges because dark ink reads well on all of them, and the text
// versions get darker on light terminals so they stay readable.
type theme struct {
	mode ThemeMode

	header    lipgloss.TerminalColor
	payload   lipgloss.TerminalColor
	signature lipgloss.TerminalColor
	success   lipgloss.TerminalColor
	warning   lipgloss.TerminalColor
	danger    lipgloss.TerminalColor
	muted     lipgloss.TerminalColor
}

// Fill colors are the actual logo colors and do not change with the
// terminal theme. They are always used with dark ink, so contrast is
// predictable on both light and dark backgrounds.
var (
	fillHeader    = lipgloss.Color("#FFC107")
	fillPayload   = lipgloss.Color("#F48C6C")
	fillSignature = lipgloss.Color("#3DD0F2")
	fillSuccess   = lipgloss.Color("#7EE787")
	fillWarning   = lipgloss.Color("#FFA657")
	fillDanger    = lipgloss.Color("#FF6B6B")
	pillInk       = lipgloss.Color("#141414")
)

func newTheme(mode ThemeMode) theme {
	if mode == "" {
		mode = ThemeAuto
	}
	return theme{
		mode:      mode,
		header:    adaptive("#9A6600", "#FFC107", mode),
		payload:   adaptive("#C2553D", "#F48C6C", mode),
		signature: adaptive("#007A8C", "#3DD0F2", mode),
		success:   adaptive("#1A7F37", "#7EE787", mode),
		warning:   adaptive("#B35C00", "#FFA657", mode),
		danger:    adaptive("#C62828", "#FF6B6B", mode),
		muted:     adaptive("#6B6257", "#A79E8F", mode),
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

func (t theme) sectionFill(s section) lipgloss.TerminalColor {
	switch s {
	case sectionHeader:
		return fillHeader
	case sectionPayload:
		return fillPayload
	case sectionSignature:
		return fillSignature
	default:
		return fillSignature
	}
}

// pill is the one filled shape in the UI. It is used for the active
// tab, the status badges, and the mode label in the footer.
func (t theme) pill(label string, fill lipgloss.TerminalColor) string {
	return lipgloss.NewStyle().
		Background(fill).
		Foreground(pillInk).
		Bold(true).
		Padding(0, 1).
		Render(label)
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

// cursorLine is a very slight tint, not a block of color. The textarea
// keeps the terminal's normal foreground, so JSON stays readable.
func (t theme) cursorLine(s section) lipgloss.TerminalColor {
	switch s {
	case sectionHeader:
		return adaptive("#FFF4D6", "#2A2208", t.mode)
	case sectionPayload:
		return adaptive("#FFE9E3", "#2A1714", t.mode)
	default:
		return adaptive("#DFF8FD", "#08222A", t.mode)
	}
}
