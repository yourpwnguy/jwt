package output

import (
	"os"

	"github.com/yourpwnguy/gostyle"
)

// style.go centralizes color decisions so formatting code never asks
// "is stdout a TTY?" directly.
//
// Why gostyle instead of raw ANSI? Single dependency, already in go.mod,
// consistent palette across cli and output. Colors are best-effort UI —
// they must never change machine output (json/compact ignore them).

var style = gostyle.New()

// SupportsColor reports whether styled human output is appropriate.
// Explicit NoColor always wins; otherwise we require a character device
// on stdout and a non-dumb TERM. Pipes and CI get plain text.
func SupportsColor(noColor bool) bool {
	if noColor {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return true
}

// palette bundles the few colors we actually use. Keeping it tiny
// avoids rainbow-vomit output — color should signal, not decorate.
// Signals: ✓ green, ✗ red, ! yellow. Everything else is plain bright
// text: gray/dim looked elegant on one dark terminal and vanished on
// every light one, so structural elements (rules, dots, labels) stay
// unstyled. Visibility beats subtlety.
type palette struct {
	use bool
}

// Symbols stay ASCII-safe and greppable: verdict text keeps
// "PASS"/"VALID" words, symbols are a prefix, never a replacement.
const (
	symOK   = "✓"
	symFail = "✗"
	symWarn = "!"
	symDot  = "●"
)

func newPalette(noColor bool) palette {
	return palette{use: SupportsColor(noColor)}
}

func (p palette) red(s string) string {
	if !p.use {
		return s
	}
	return style.Red(s)
}

func (p palette) green(s string) string {
	if !p.use {
		return s
	}
	return style.Green(s)
}

func (p palette) yellow(s string) string {
	if !p.use {
		return s
	}
	return style.Yellow(s)
}

func (p palette) dim(s string) string {
	// Deliberately plain: Dim (SGR 2) renders as near-invisible on
	// most terminals. Signal colors carry meaning; dim carried only
	// decoration, so we drop it entirely.
	return s
}

func (p palette) gray(s string) string {
	// Deliberately plain: BrBlack (90) is unreadable on light
	// backgrounds and faint on dark ones. Structural text stays
	// full-brightness so it reads everywhere.
	return s
}

func (p palette) bold(s string) string {
	if !p.use {
		return s
	}
	return style.Bold(s)
}

// rule returns a thin separator. Fixed widths (not terminal-width)
// keep output stable in pipes, tests, and narrow windows. Plain (no
// color) so it never washes out.
func (p palette) rule(n int) string {
	if n <= 0 {
		n = 52
	}
	s := ""
	for i := 0; i < n; i++ {
		s += "─"
	}
	return s
}
