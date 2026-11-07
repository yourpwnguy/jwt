// Package tui implements the interactive token editor behind `jwt tui`.
//
// The package owns layout and keyboard flow only. Parsing and encoding
// live in internal/jwt, policy checks in internal/audit, and signing
// math in internal/sign. Keeping those boundaries means the editor can
// be tested without a terminal and the domain packages stay usable from
// scripts.
//
// The central rule is that the editor never rewrites a token silently:
// untouched panes keep their original base64 segments, and edited panes
// are encoded from the exact text on screen. The signature badge reports
// untouched, signed, stale, custom, unsigned, or invalid, and never
// claims "valid" unless the user just signed the token.
package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// ThemeMode selects the palette. Auto follows the terminal background;
// dark and light force a side for terminals that report it wrong.
type ThemeMode string

const (
	ThemeAuto  ThemeMode = "auto"
	ThemeDark  ThemeMode = "dark"
	ThemeLight ThemeMode = "light"
)

// Options configures one editor session.
type Options struct {
	// Token is a normalized token to edit. Empty starts a blank template.
	Token string
	// NoColor disables styling. NO_COLOR in the environment does the same.
	NoColor bool
	// Theme is auto, dark, or light.
	Theme ThemeMode
	// InputIsTTY is false when stdin is a pipe. In that case the TUI
	// opens the controlling terminal for keys instead of reading EOF.
	InputIsTTY bool
	// Input and Output default to os.Stdin and os.Stdout.
	Input  io.Reader
	Output io.Writer
	// Copy is injected for tests. The default sends OSC 52 to Output.
	Copy func(string) error
	// Now is injected for deterministic countdowns and audits.
	Now func() time.Time
}

// Run starts the editor and blocks until the user quits.
//
// A canceled context exits through Bubble Tea's ErrProgramKilled, which
// is treated as a normal quit. Real terminal setup failures are passed
// through with context so the CLI can print them.
func Run(ctx context.Context, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	switch opts.Theme {
	case "", ThemeAuto, ThemeDark, ThemeLight:
	default:
		return fmt.Errorf("tui: invalid theme %q (want auto, dark, or light)", opts.Theme)
	}

	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	if !isTerminal(out) {
		return fmt.Errorf("tui: output is not a terminal")
	}
	if opts.NoColor || os.Getenv("NO_COLOR") != "" {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	if opts.Copy == nil {
		opts.Copy = func(s string) error { return copyOSC52(out, s) }
	}

	m, err := newModel(ctx, opts)
	if err != nil {
		return err
	}

	programOpts := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithAltScreen(),
		tea.WithOutput(out),
	}
	if opts.InputIsTTY {
		if opts.Input != nil {
			programOpts = append(programOpts, tea.WithInput(opts.Input))
		}
	} else {
		// stdin was a pipe, so its reader is already at EOF. Open the
		// controlling terminal for keys instead.
		programOpts = append(programOpts, tea.WithInputTTY())
	}

	_, err = tea.NewProgram(m, programOpts...).Run()
	if errors.Is(err, tea.ErrProgramKilled) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
