// Command jwt decodes, audits, and verifies JWTs offline.
//
// Thin entry point by design: all logic lives in internal/cli so main
// only wires OS resources (args, stdio, signals) and maps the exit code.
// Keeping main tiny makes the whole app testable without subprocesses.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/yourpwnguy/jwt/internal/cli"
)

// main exits with cli.Run's code. os.Exit skips defers by design —
// Run flushes its own writers, so nothing is lost.
func main() {
	os.Exit(run())
}

// run builds a cancellable context (Ctrl-C stops batch work promptly),
// detects piped stdin, and delegates to the CLI router.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	stdinIsTTY := isTerminal(os.Stdin)
	return cli.Run(ctx, os.Args[1:], os.Stdin, stdinIsTTY, os.Stdout, os.Stderr)
}

// isTerminal reports whether f is a character device (a human terminal)
// rather than a pipe or redirect. Piped input is auto-read; TTY input
// is not (avoids hanging waiting for keystrokes on bare "jwt").
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
