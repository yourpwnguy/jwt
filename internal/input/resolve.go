package input

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourpwnguy/jwt/internal/reader"
)

// resolve.go orchestrates Config into an ordered token list.
//
// Order is stable: -t first, then args, then file, then piped stdin.
// Stable order matters because output numbers results ("Token 1:") —
// shuffling would break scripts correlating input line N with output N.

// Resolve collects tokens from all sources, respecting ctx between
// sources so Ctrl-C during a big file stops promptly.
func Resolve(ctx context.Context, cfg Config) ([]string, error) {
	var out []string
	appendTokens := func(more []string) error {
		if len(out)+len(more) > MaxTokens {
			return fmt.Errorf("too many tokens: capped at %d (split the input)", MaxTokens)
		}
		out = append(out, more...)
		return nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cfg.Token != "" {
		if cfg.Extract {
			if err := appendTokens(ExtractTokens(cfg.Token)); err != nil {
				return nil, err
			}
		} else if norm, ok := Normalize(cfg.Token); ok {
			if err := appendTokens([]string{norm}); err != nil {
				return nil, err
			}
		}
	}
	for _, arg := range cfg.Args {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if arg == "-" {
			toks, err := readStdin(cfg, false)
			if err != nil {
				return nil, err
			}
			if err := appendTokens(toks); err != nil {
				return nil, err
			}
			continue
		}
		if cfg.Extract {
			if err := appendTokens(ExtractTokens(arg)); err != nil {
				return nil, err
			}
			continue
		}
		if norm, ok := Normalize(arg); ok {
			if err := appendTokens([]string{norm}); err != nil {
				return nil, err
			}
		}
	}
	if cfg.File != "" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lines, err := reader.ReadLines(cfg.File)
		if err != nil {
			return nil, err
		}
		if cfg.Extract {
			if err := appendTokens(ExtractTokens(strings.Join(lines, "\n"))); err != nil {
				return nil, err
			}
		} else {
			cleaned := make([]string, 0, len(lines))
			for _, line := range lines {
				if norm, ok := Normalize(line); ok {
					cleaned = append(cleaned, norm)
				}
			}
			if err := appendTokens(cleaned); err != nil {
				return nil, err
			}
		}
	}
	// Implicit pipe: only when TTY=false and nothing else gave tokens.
	// This avoids hanging on bare "jwt" while making pipes just work.
	if len(out) == 0 && cfg.Stdin != nil && !cfg.StdinIsTTY {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		toks, err := readStdin(cfg, true)
		if err != nil {
			return nil, err
		}
		if err := appendTokens(toks); err != nil {
			return nil, err
		}
	}
	return out, nil
}
