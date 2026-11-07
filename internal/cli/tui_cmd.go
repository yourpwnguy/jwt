package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/yourpwnguy/jwt/v2/internal/input"
	"github.com/yourpwnguy/jwt/v2/internal/tui"
)

// tui_cmd.go wires the interactive editor into the CLI. It owns flags
// and input resolution; the editor owns every keystroke after launch.
// stdout must be a real terminal because Bubble Tea takes over the
// alternate screen.

func runTUI(ctx context.Context, args []string, stdin io.Reader, stdinIsTTY bool, stdout, stderr io.Writer) int {
	var opts struct {
		token     string
		file      string
		theme     string
		noColor   bool
		noVimKeys bool
	}
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { printTUIUsage(stderr) }
	fs.StringVar(&opts.token, "t", "", "token to edit")
	fs.StringVar(&opts.file, "f", "", "file with one token")
	fs.StringVar(&opts.theme, "theme", "", "auto, dark, or light")
	fs.BoolVar(&opts.noColor, "no-color", false, "disable color")
	fs.BoolVar(&opts.noVimKeys, "no-vim-keys", false, "disable ctrl+h/j/k/l motions")

	flagArgs, positional := partitionArgs(fs, args)
	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitSystemErr
	}
	if opts.theme != "" {
		switch tui.ThemeMode(opts.theme) {
		case tui.ThemeAuto, tui.ThemeDark, tui.ThemeLight:
		default:
			errPrint(stderr, "invalid --theme %q (want auto, dark, or light)", opts.theme)
			return ExitSystemErr
		}
	}

	// The TTY guard comes after flag parsing so -h and bad flags still
	// work when stdout is redirected. NO_COLOR only disables styling, so
	// it must not be part of this check.
	if !isTerminalFile(stdout) {
		errPrint(stderr, "tui needs an interactive terminal (stdout is not a TTY)")
		return ExitSystemErr
	}
	if os.Getenv("TERM") == "dumb" {
		errPrint(stderr, "tui needs a real terminal (TERM=dumb)")
		return ExitSystemErr
	}

	tokens, err := input.Resolve(ctx, input.Config{
		Args: positional, Token: opts.token, File: opts.file,
		Stdin: stdin, StdinIsTTY: stdinIsTTY,
	})
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}
	if len(tokens) > 1 {
		errPrint(stderr, "tui edits one token, got %d (pass one token or a file with one line)", len(tokens))
		return ExitSystemErr
	}

	token := ""
	if len(tokens) == 1 {
		token = tokens[0]
	}
	err = tui.Run(ctx, tui.Options{
		Token:      token,
		NoColor:    opts.noColor,
		Theme:      tui.ThemeMode(opts.theme),
		NoVimKeys:  opts.noVimKeys,
		InputIsTTY: stdinIsTTY,
		Input:      stdin,
		Output:     stdout,
	})
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}
	return ExitOK
}

// isTerminalFile reports whether w is a character device. It is
// deliberately separate from isTerminalWriter because NO_COLOR must
// disable color, not the interactive editor.
func isTerminalFile(w io.Writer) bool {
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

func printTUIUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: jwt tui [options] [token]

Live token editor. Edit the header and payload JSON, watch the encoded
token and the security audit update as you type.

Examples:
  jwt tui                         start with a blank token
  jwt tui <token>                 edit a token
  jwt tui -t <token>              same, flag form
  echo $TOKEN | jwt tui           read the token from a pipe
  jwt tui -f token.txt            read one token from a file

Options:
  -t <token>        token to edit
  -f <file>         file with one token
  --theme           auto, dark, or light (default auto)
  --no-color        plain text UI
  --no-vim-keys     disable ctrl+h/j/k/l cursor motions
  -h                show this help

Keys:
  arrows            move the cursor
  ctrl+h / ctrl+l   move left / right
  ctrl+j / ctrl+k   move down / up
  tab / shift+tab   next or previous section
  ctrl+o            command mode (esc works too)
                    f format, c compact, a audit, v token, q quit
  ctrl+s            sign the token
  ctrl+y            copy the encoded token
  ctrl+c            quit

ctrl+i is the Tab key on a terminal, so it keeps switching sections.
`)
}
