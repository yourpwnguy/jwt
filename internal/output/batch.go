package output

import (
	"fmt"
	"strings"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// batch.go renders many tokens with stable numbering.
// Table mode returns one table; other modes join single outputs with
// "Token N:" headers unless Quiet. Order matches input order so scripts
// can correlate line N in with Token N out.

// FormatBatch renders many tokens.
func FormatBatch(toks []jwt.Token, opts Options) (string, error) {
	opts.Normalize()
	if len(toks) == 0 {
		return "", nil
	}
	// Claim batch prints one value per line, no headers — what
	// `jwt -tL list --claim sub` scripts expect.
	if opts.Claim != "" {
		var b strings.Builder
		for i, tok := range toks {
			v, err := ClaimValue(tok, opts.Claim)
			if err != nil {
				return "", fmt.Errorf("token %d: %w", i+1, err)
			}
			b.WriteString(v)
			b.WriteByte('\n')
		}
		return strings.TrimSuffix(b.String(), "\n"), nil
	}
	if opts.Format == FormatTable {
		return formatTable(toks, opts), nil
	}
	var b strings.Builder
	showIndex := len(toks) > 1 && !opts.Quiet
	p := newPalette(opts.NoColor)
	for i, tok := range toks {
		one, err := FormatOne(tok, opts)
		if err != nil {
			return "", fmt.Errorf("token %d: %w", i+1, err)
		}
		if showIndex {
			fmt.Fprintf(&b, "%s %s\n", p.gray(symDot), p.bold(fmt.Sprintf("Token %d", i+1)))
		}
		b.WriteString(one)
		if i < len(toks)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String(), nil
}
