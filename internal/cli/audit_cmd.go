package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/yourpwnguy/jwt/internal/audit"
	"github.com/yourpwnguy/jwt/internal/input"
	"github.com/yourpwnguy/jwt/internal/jwt"
	"github.com/yourpwnguy/jwt/internal/output"
)

// audit_cmd.go implements `jwt audit`: static checks over every token.
// No secrets, no network — safe for CI on untrusted dumps. Output is
// either a human triage table or JSON for `jq`/gates.

// parseFailOn maps --fail-on to a severity threshold. Unknown values
// error instead of silently defaulting — a misspelled gate that always
// passes would be worse than no gate.
func parseFailOn(s string) (audit.Severity, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "low":
		return audit.SeverityLow, nil
	case "med", "medium":
		return audit.SeverityMedium, nil
	case "high":
		return audit.SeverityHigh, nil
	case "crit", "critical":
		return audit.SeverityCritical, nil
	default:
		return "", fmt.Errorf("invalid --fail-on %q (want low|medium|high|critical)", s)
	}
}

func runAudit(ctx context.Context, args []string, stdin io.Reader, stdinIsTTY bool, stdout, stderr io.Writer) int {
	var token, file, format, failOn string
	var extract, quiet, noColor bool
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&token, "t", "", "single JWT token")
	fs.StringVar(&file, "tL", "", "file with one token per line")
	fs.BoolVar(&extract, "extract", false, "scan text for embedded JWTs")
	fs.StringVar(&format, "o", "pretty", "output: pretty|json|compact|table")
	fs.StringVar(&format, "format", "pretty", "output: pretty|json|compact|table")
	fs.StringVar(&failOn, "fail-on", "high", "fail when severity >= low|medium|high|critical")
	fs.BoolVar(&quiet, "q", false, "no Token N: headers")
	fs.BoolVar(&quiet, "quiet", false, "no Token N: headers")
	fs.BoolVar(&noColor, "no-color", false, "plain output")
	flagArgs, positional := partitionArgs(fs, args)
	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitSystemErr
	}

	threshold, err := parseFailOn(failOn)
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}

	if _, err := parseFormat(format); err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}

	rawTokens, err := input.Resolve(ctx, input.Config{
		Args: positional, Token: token, File: file,
		Extract: extract, Stdin: stdin, StdinIsTTY: stdinIsTTY,
	})
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}
	if len(rawTokens) == 0 {
		errPrint(stderr, "no tokens: provide a token, -t, -tL, or piped input")
		return ExitSystemErr
	}

	now := time.Now()
	toks := make([]jwt.Token, 0, len(rawTokens))
	hadDecodeErr := false
	for i, raw := range rawTokens {
		tok, err := jwt.Parse(raw)
		if err != nil {
			errPrint(stderr, "Token %d: %v", i+1, err)
			hadDecodeErr = true
			continue
		}
		toks = append(toks, tok)
	}
	if len(toks) == 0 {
		return ExitSystemErr
	}

	reps := make([]audit.Report, 0, len(toks))
	breached := false
	for _, tok := range toks {
		if err := ctx.Err(); err != nil {
			errPrint(stderr, "cancelled")
			return ExitSystemErr
		}
		rep := audit.Run(ctx, tok, now)
		reps = append(reps, rep)
		if rep.FailedAtOrAbove(threshold) {
			breached = true
		}
	}

	formatted, err := output.FormatAuditBatch(reps, output.Options{
		Format: output.Format(format), Quiet: quiet, NoColor: noColor, Now: now,
		FailOn: strings.ToLower(strings.TrimSpace(failOn)),
	})
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}
	fmt.Fprintln(stdout, formatted)

	switch {
	case breached:
		return ExitValidation
	case hadDecodeErr:
		return ExitSystemErr
	default:
		return ExitOK
	}
}
