package input

import (
	"fmt"
	"io"
	"strings"
)

// stdin.go drains piped input with a size cap. Splitting from resolve.go
// is deliberate: stdin has its own limits and line-splitting policy,
// file/args have theirs. Mixing them would blur two failure modes.

// readStdin reads up to MaxStdinBytes. Empty implicit pipes yield no
// tokens (not an error); callers report "no tokens" with better context.
//
// A terminal is refused outright: blocking on keystrokes for a token no
// human will ever type is always a mistake, and the blocked read below
// cannot be interrupted. The implicit-pipe path never reaches here on a
// TTY (Resolve guards it), so this only triggers on an explicit "-".
func readStdin(cfg Config, allowEmpty bool) ([]string, error) {
	if cfg.Stdin == nil {
		return nil, nil
	}
	if cfg.StdinIsTTY {
		return nil, fmt.Errorf("cannot read stdin: stdin is a terminal, pipe input instead (e.g. echo $TOKEN | jwt -)")
	}
	data, err := io.ReadAll(io.LimitReader(cfg.Stdin, MaxStdinBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}
	if len(data) > MaxStdinBytes {
		return nil, fmt.Errorf("stdin too large: capped at %d bytes", MaxStdinBytes)
	}
	text := strings.TrimSpace(string(data))
	if text == "" && allowEmpty {
		return nil, nil
	}
	if cfg.Extract {
		return ExtractTokens(text), nil
	}
	// One token or many newline-separated — split handles both.
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if norm, ok := Normalize(line); ok {
			out = append(out, norm)
			if len(out) >= MaxTokens {
				break
			}
		}
	}
	return out, nil
}
