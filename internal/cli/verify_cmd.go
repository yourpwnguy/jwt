package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/yourpwnguy/jwt/v2/internal/input"
	"github.com/yourpwnguy/jwt/v2/internal/jwt"
	"github.com/yourpwnguy/jwt/v2/internal/output"
	"github.com/yourpwnguy/jwt/v2/internal/verify"
)

// verify_cmd.go implements `jwt verify`: math first, usability second.
// Each token prints its decoded body plus Signature/Claims verdicts.
// Batch continues past individual failures; the exit code remembers.

// verifyFlags mirrors `jwt verify -h`.
type verifyFlags struct {
	token     string
	file      string
	extract   bool
	format    string
	quiet     bool
	noColor   bool
	secret    string
	pubkey    string
	jwks      string
	aud       string
	iss       string
	leewayStr string
}

func runVerify(ctx context.Context, args []string, stdin io.Reader, stdinIsTTY bool, stdout, stderr io.Writer) int {
	var o verifyFlags
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.token, "t", "", "single JWT token")
	fs.StringVar(&o.file, "tL", "", "file with one token per line")
	fs.BoolVar(&o.extract, "extract", false, "scan text for embedded JWTs")
	fs.StringVar(&o.format, "o", "pretty", "output: pretty|json|compact|table")
	fs.StringVar(&o.format, "format", "pretty", "output: pretty|json|compact|table")
	fs.BoolVar(&o.quiet, "q", false, "no Token N: headers")
	fs.BoolVar(&o.quiet, "quiet", false, "no Token N: headers")
	fs.BoolVar(&o.noColor, "no-color", false, "plain output")
	fs.StringVar(&o.secret, "secret", "", "HMAC secret (prefix @ for file)")
	fs.StringVar(&o.pubkey, "pubkey", "", "PEM/DER/cert/JWK public key file")
	fs.StringVar(&o.jwks, "jwks", "", "JWKS file or https URL")
	fs.StringVar(&o.aud, "aud", "", "require audience")
	fs.StringVar(&o.iss, "iss", "", "require issuer")
	fs.StringVar(&o.leewayStr, "leeway", "", "clock skew tolerance (e.g. 30s)")
	flagArgs, positional := partitionArgs(fs, args)
	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitSystemErr
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
		return ExitSystemErr
	}

	now := time.Now()
	vOpts := verify.Options{
		Secret: o.secret, PubKeyPath: o.pubkey, JWKS: o.jwks,
		Aud: o.aud, Iss: o.iss, Leeway: leeway, Now: now,
	}

	// Per-token loop: decode errors are system errors (bad input),
	// invalid signatures/claims are validation failures. Both print
	// per-token context but don't abort siblings.
	hadSystemErr := false
	hadValidationFail := false
	showIndex := len(rawTokens) > 1 && !o.quiet
	for i, raw := range rawTokens {
		if err := ctx.Err(); err != nil {
			errPrint(stderr, "cancelled")
			return ExitSystemErr
		}
		tok, err := jwt.Parse(raw)
		if err != nil {
			errPrint(stderr, "Token %d: %v", i+1, err)
			hadSystemErr = true
			continue
		}
		res, err := verify.Verify(ctx, tok, vOpts)
		if err != nil {
			// Map sentinel errors to the right exit class: none and
			// bad-sig-shape are validation, missing keys/files/net
			// are system errors. Default unknown errors to system.
			if errors.Is(err, verify.ErrNone) {
				hadValidationFail = true
			} else if errors.Is(err, verify.ErrNoKey) || errors.Is(err, verify.ErrUnsupportedAlg) {
				errPrint(stderr, "Token %d: %v", i+1, err)
				hadSystemErr = true
				printVerifyResult(stdout, stderr, i, tok, res, o, showIndex)
				continue
			} else {
				errPrint(stderr, "Token %d: %v", i+1, err)
				hadSystemErr = true
				printVerifyResult(stdout, stderr, i, tok, res, o, showIndex)
				continue
			}
		}
		if !res.SignatureValid || !res.ClaimsValid {
			// No-key mode reports claims only; missing sig isn't a
			// "forgery" there. Only flag validation when keys existed
			// or claims themselves failed.
			if vOpts.HasKey() || !res.ClaimsValid {
				// Distinguish: HasKey + sig false => forgery (validation).
				// No keys + claims false => expired/aud (validation).
				hadValidationFail = true
			}
		}
		printVerifyResult(stdout, stderr, i, tok, res, o, showIndex)
	}

	switch {
	case hadValidationFail:
		return ExitValidation
	case hadSystemErr:
		return ExitSystemErr
	default:
		// Single-token valid path also needs the HasKey nuance: if the
		// user forgot keys entirely, Verify errored above and we already
		// set hadSystemErr. Reaching here means all good.
		return ExitOK
	}
}

// printVerifyResult formats one token's verdict. Format errors are
// reported inline (they shouldn't happen) without aborting the batch.
func printVerifyResult(stdout, stderr io.Writer, idx int, tok jwt.Token, res verify.Result, o verifyFlags, showIndex bool) {
	formatted, err := output.FormatVerify(tok, res, output.Format(o.format), o.noColor)
	if err != nil {
		errPrint(stderr, "Token %d: %v", idx+1, err)
		return
	}
	if showIndex {
		fmt.Fprintf(stdout, "● Token %d\n", idx+1)
	}
	fmt.Fprintln(stdout, formatted)
	if showIndex {
		fmt.Fprintln(stdout, "")
	}
}
