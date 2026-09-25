package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/yourpwnguy/jwt/internal/input"
	"github.com/yourpwnguy/jwt/internal/jwt"
	"github.com/yourpwnguy/jwt/internal/output"
)

// decode.go implements `jwt decode` (and the bare `jwt` compat path).
// Flow is linear and obvious: flags -> resolve input -> parse each ->
// format once -> print. Errors per token don't abort the batch, but the
// exit code remembers if anything failed.

// decodeOptions mirrors flags; kept separate from input.Config and
// output.Options so each package's API stays narrow.
type decodeOptions struct {
	token       string
	file        string
	extract     bool
	format      string
	headerOnly  bool
	payloadOnly bool
	claim       string
	quiet       bool
	noColor     bool
	verifyExp   bool
	leewayStr   string
}

func runDecode(ctx context.Context, args []string, stdin io.Reader, stdinIsTTY bool, stdout, stderr io.Writer) int {
	var o decodeOptions
	fs := flag.NewFlagSet("decode", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.token, "t", "", "single JWT token")
	fs.StringVar(&o.file, "tL", "", "file with one token per line")
	fs.BoolVar(&o.extract, "extract", false, "scan text for embedded JWTs")
	fs.StringVar(&o.format, "o", "pretty", "output: pretty|json|compact|table")
	fs.StringVar(&o.format, "format", "pretty", "output: pretty|json|compact|table")
	fs.BoolVar(&o.headerOnly, "H", false, "header only")
	fs.BoolVar(&o.payloadOnly, "P", false, "payload only")
	fs.StringVar(&o.claim, "claim", "", "single claim dot-path (e.g. sub)")
	fs.BoolVar(&o.quiet, "q", false, "just data, no Token headers or status line")
	fs.BoolVar(&o.quiet, "quiet", false, "just data, no Token headers or status line")
	fs.BoolVar(&o.noColor, "no-color", false, "plain output")
	fs.BoolVar(&o.verifyExp, "verify-exp", false, "exit 2 when expired/not-yet-valid")
	fs.StringVar(&o.leewayStr, "leeway", "", "clock skew tolerance (e.g. 30s)")
	// Compat: -v prints version anywhere.
	showVersion := fs.Bool("v", false, "print version")
	flagArgs, positional := partitionArgs(fs, args)
	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitSystemErr
	}
	if *showVersion {
		printVersion(stderr)
		return ExitOK
	}

	leeway, err := parseLeeway(o.leewayStr)
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}

	if _, err := parseFormat(o.format); err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}

	rawTokens, err := input.Resolve(ctx, input.Config{
		Args: positional, Token: o.token, File: o.file,
		Extract: o.extract, Stdin: stdin, StdinIsTTY: stdinIsTTY,
	})
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}
	if len(rawTokens) == 0 {
		errPrint(stderr, "no tokens: provide a token, -t, -tL, or piped input")
		printUsage(stderr)
		return ExitSystemErr
	}

	// Parse all before printing so a malformed token mid-batch still
	// yields per-token errors but stable numbering.
	toks := make([]jwt.Token, 0, len(rawTokens))
	hadDecodeErr := false
	now := time.Now()
	expired := false
	for i, raw := range rawTokens {
		tok, err := jwt.Parse(raw)
		if err != nil {
			errPrint(stderr, "Token %d: %v", i+1, err)
			hadDecodeErr = true
			continue
		}
		if o.verifyExp {
			if st, _ := tok.CheckTime(now, leeway); st == jwt.TemporalExpired || st == jwt.TemporalNotYetValid {
				expired = true
			}
		}
		toks = append(toks, tok)
	}
	if len(toks) == 0 {
		return ExitSystemErr
	}

	outOpts := output.Options{
		Format: output.Format(o.format), HeaderOnly: o.headerOnly,
		PayloadOnly: o.payloadOnly, Claim: o.claim,
		Quiet: o.quiet, NoColor: o.noColor, Now: now,
	}
	formatted, err := output.FormatBatch(toks, outOpts)
	if err != nil {
		errPrint(stderr, "%v", err)
		return ExitSystemErr
	}
	fmt.Fprintln(stdout, formatted)

	switch {
	case expired:
		return ExitValidation
	case hadDecodeErr:
		return ExitSystemErr
	default:
		return ExitOK
	}
}
