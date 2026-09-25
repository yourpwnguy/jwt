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
	"syscall"

	"github.com/yourpwnguy/jwt/internal/cli"
)

// main exits with cli.Run's code. os.Exit skips defers by design —
// Run flushes its own writers, so nothing is lost.
func main() {
	os.Exit(run())
}

// run builds a cancellable context (Ctrl-C stops batch work promptly),
// detects piped stdin, and delegates to the CLI router.
//
// Signal handling is two-stage. signal.NotifyContext swallows SIGINT,
// which disables the default kill behavior, so a process stuck in
// uninterruptible I/O (e.g. a blocked stdin read) would otherwise ignore
// Ctrl-C forever. The first signal cancels gracefully; a second one
// force-exits with 130, the standard shell code for SIGINT. Ctrl-C
// always works, no matter where the program is stuck.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go forceExitOnSecondSignal(ctx)

	stdinIsTTY := isTerminal(os.Stdin)
	return cli.Run(ctx, os.Args[1:], os.Stdin, stdinIsTTY, os.Stdout, os.Stderr)
}

// forceExitOnSecondSignal waits for one more interrupt after ctx is done
// and exits. The channel is buffered so a rapid double Ctrl-C is never
// lost between the cancel and this goroutine starting to listen.
func forceExitOnSecondSignal(ctx context.Context) {
	<-ctx.Done()
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	<-sigs
	os.Exit(130)
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
