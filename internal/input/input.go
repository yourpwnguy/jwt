// Package input answers "where do tokens come from?" so the rest of
// the codebase never thinks about stdin, files, or Bearer prefixes.
//
// Dependency direction is strict: input may depend on internal/reader
// and internal/jwt for validation, but nobody depends on input except
// internal/cli. That keeps cleaning logic in one testable place.
//
// Design choice: Resolve takes an explicit Config with Stdin injected
// as io.Reader plus StdinIsTTY. Reading os.Stdin directly would make
// tests need real pipes; injecting makes every path unit-testable.
package input

import "io"

// Config describes every way a user can hand us tokens.
//
// Precedence is deliberate and documented so scripts are predictable:
//  1. -t flag (explicit single token, backward compat)
//  2. positional args (jwt <token>, jwt - for stdin)
//  3. -tL file (one token per line)
//  4. piped stdin, but ONLY when stdin isn't a TTY (avoids hanging
//     when a human just runs "jwt" with no input).
//
// Extract switches args/file/stdin from "this IS a token" to
// "find tokens INSIDE this text" (logs, curl dumps, headers).
type Config struct {
	// Args are positional CLI args after flags (may contain "-" for stdin).
	Args []string
	// Token is the -t flag value, if any.
	Token string
	// File is the -tL flag value, if any.
	File string
	// Extract enables scanning text for embedded JWTs.
	Extract bool
	// Stdin is usually os.Stdin; injected for tests.
	Stdin io.Reader
	// StdinIsTTY reports whether stdin is a terminal. Piped input
	// is only auto-read when this is false.
	StdinIsTTY bool
}

// Limits keep a malicious 2GB log from OOMing us.
const (
	// MaxStdinBytes caps piped input. 1MB holds ~300 tokens, plenty.
	MaxStdinBytes = 1 << 20
	// MaxTokens caps total tokens per invocation. Beyond this the
	// user should split the file — it also bounds output memory.
	MaxTokens = 10000
)
