// Package cli is the thin wiring between OS and domain: flags in,
// exit codes out. It owns no JWT logic: parsing lives in internal/jwt,
// input collecting in internal/input, rendering in internal/output,
// and the interactive editor in internal/tui.
//
// Exit contract (stable for scripts):
//
//	0: success (decoded; audit passed; signature+claims valid; TUI quit)
//	1: system error (bad flags, no input, unreadable file, malformed
//	    token, missing keys, network failure, no interactive terminal)
//	2: validation failure (expired with --verify-exp, audit --fail-on
//	    breached, signature invalid, claims invalid)
package cli

import (
	"context"
	"io"
)

// Exit codes, named so call sites read like English.
const (
	ExitOK         = 0
	ExitSystemErr  = 1
	ExitValidation = 2
)

// Run dispatches argv[1:] to decode/audit/verify. args excludes the
// program name. stdin/stdout/stderr are injected for testability;
// stdinIsTTY guards implicit pipe reading (avoids hanging on "jwt").
func Run(ctx context.Context, args []string, stdin io.Reader, stdinIsTTY bool, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		// No subcommand and no flags: implicit decode. It will report
		// "no tokens" with help if stdin is a TTY and nothing was piped.
		return runDecode(ctx, []string{}, stdin, stdinIsTTY, stdout, stderr)
	}
	switch args[0] {
	case "decode":
		return runDecode(ctx, args[1:], stdin, stdinIsTTY, stdout, stderr)
	case "audit":
		return runAudit(ctx, args[1:], stdin, stdinIsTTY, stdout, stderr)
	case "verify":
		return runVerify(ctx, args[1:], stdin, stdinIsTTY, stdout, stderr)
	case "tui":
		return runTUI(ctx, args[1:], stdin, stdinIsTTY, stdout, stderr)
	case "version", "-v", "--version":
		printVersion(stderr)
		return ExitOK
	case "-h", "--help", "help":
		printUsage(stderr)
		return ExitOK
	default:
		// Anything else is a decode: `jwt -t ...`, `jwt <token>`, or flags.
		// Unknown words are treated as token candidates so a pasted token
		// is never rejected as an "unknown command"; a non-token word
		// fails with a precise parse error and exit 1 from decode.
		return runDecode(ctx, args, stdin, stdinIsTTY, stdout, stderr)
	}
}
