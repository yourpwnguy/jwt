package verify

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// check.go runs the Verify orchestration: signature first, claims second.
//
// Why signature before claims? A forged token with a valid exp is still
// forged. Checking math first avoids "claims valid!" confusing output
// on tokens nobody signed.

// Verify checks signature (if keys given) and temporal/audience claims.
// It never fetches keys without ctx — pass a timeout from the CLI.
func Verify(ctx context.Context, tok jwt.Token, opts Options) (Result, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	alg := tok.Alg()
	if alg == "" {
		return Result{Alg: alg}, fmt.Errorf("missing alg header: cannot verify")
	}
	// none must never verify — not even with empty secret. This is the
	// single most important line in the package.
	if strings.EqualFold(alg, "none") {
		reason := claimCheck(tok, opts, now)
		return Result{Alg: alg, ClaimsValid: reason == "", ClaimsReason: reason},
			fmt.Errorf("%w (unsigned)", ErrNone)
	}
	res := Result{Alg: alg}
	if !opts.HasKey() {
		// No keys: can't judge math, but still report claims so
		// "jwt verify --aud api" without keys is a useful expiry check.
		reason := claimCheck(tok, opts, now)
		res.ClaimsValid = reason == ""
		res.ClaimsReason = reason
		return res, fmt.Errorf("%w: provide --secret, --pubkey, or --jwks", ErrNoKey)
	}
	sigValid, err := verifySignature(ctx, tok, opts)
	res.SignatureValid = sigValid
	if err != nil {
		reason := claimCheck(tok, opts, now)
		res.ClaimsValid = reason == ""
		res.ClaimsReason = reason
		return res, err
	}
	reason := claimCheck(tok, opts, now)
	res.ClaimsValid = reason == ""
	res.ClaimsReason = reason
	return res, nil
}

// claimCheck returns "" when usable, else a human reason.
// It checks nbf/exp with leeway, then aud/iss when requested.
func claimCheck(tok jwt.Token, opts Options, now time.Time) string {
	status, msg := tok.CheckTime(now, opts.Leeway)
	switch status {
	case jwt.TemporalExpired, jwt.TemporalNotYetValid:
		return msg
	}
	if opts.Aud != "" && !audMatches(tok, opts.Aud) {
		return fmt.Sprintf("audience mismatch: want %q", opts.Aud)
	}
	if opts.Iss != "" {
		if iss, _ := tok.Payload["iss"].(string); iss != opts.Iss {
			return fmt.Sprintf("issuer mismatch: want %q got %q", opts.Iss, iss)
		}
	}
	return ""
}

// audMatches handles both "aud":"api" and "aud":["a","b"] forms,
// because producers can't agree and strictness here would false-fail.
func audMatches(tok jwt.Token, want string) bool {
	v, ok := tok.Payload["aud"]
	if !ok {
		return false
	}
	switch a := v.(type) {
	case string:
		return a == want
	case []any:
		for _, item := range a {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}
