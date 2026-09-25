package output

import (
	"encoding/json"

	"github.com/yourpwnguy/jwt/internal/jwt"
)

// format.go turns one token into stdout text. It owns JSON marshaling
// for decode output — jwt.Token stays a struct, this package decides
// indentation and human extras.
//
// Performance note: each token marshals once. Table mode avoids
// pretty-printing bodies it won't show (see table.go).

// view is the stable decode shape. Keeping header/payload/signature
// keys identical to jwt.io means existing jq filters keep working.
type view struct {
	Header    map[string]any `json:"header"`
	Payload   map[string]any `json:"payload"`
	Signature string         `json:"signature"`
}

// FormatOne renders a single token per opts (no "Token N:" header).
func FormatOne(tok jwt.Token, opts Options) (string, error) {
	opts.Normalize()
	// Single-claim mode wins — it's the scripting path.
	if opts.Claim != "" {
		return ClaimValue(tok, opts.Claim)
	}
	if opts.HeaderOnly {
		return marshalBody(tok.Header, opts.Format)
	}
	if opts.PayloadOnly {
		return marshalBody(tok.Payload, opts.Format)
	}
	if opts.Format == FormatTable {
		return formatTable([]jwt.Token{tok}, opts), nil
	}
	data, err := marshalView(view{tok.Header, tok.Payload, tok.Signature}, opts.Format)
	if err != nil {
		return "", err
	}
	// Pretty leads with a key-facts row (alg · sub · status) so the eye
	// lands on signal first, JSON second. Quiet drops the row and prints
	// just the data, for users who want headers nowhere. json/compact
	// stay pure for `| jq` either way.
	if opts.Format == FormatPretty && !opts.Quiet {
		return factsRow(tok, opts) + "\n" + data, nil
	}
	return data, nil
}

// marshalView encodes the full token shape per format.
func marshalView(v view, f Format) (string, error) {
	if f == FormatCompact {
		data, err := json.Marshal(v)
		return string(data), err
	}
	// Pretty and json both indent; pretty may add extras upstream.
	data, err := json.MarshalIndent(v, "", "  ")
	return string(data), err
}

// marshalBody encodes a header-only or payload-only fragment.
func marshalBody(body map[string]any, f Format) (string, error) {
	if f == FormatCompact {
		data, err := json.Marshal(body)
		return string(data), err
	}
	data, err := json.MarshalIndent(body, "", "  ")
	return string(data), err
}

// factsRow is the minimalist status header: dim structure, bright data,
// one signal at the end. Example:
// ● HS256 · sub 1234567890 · ✓ valid expires in 14m (2024-05-20T10:23:42Z)
func factsRow(tok jwt.Token, opts Options) string {
	p := newPalette(opts.NoColor)
	alg := tok.Alg()
	if alg == "" {
		alg = "-"
	}
	sub := "-"
	if v, ok := tok.Claim("sub"); ok {
		sub = shortValue(v, 24)
	}
	dot := p.gray(symDot + " ")
	mid := p.gray(" · ")
	subPart := "sub " + p.bold(sub)
	status, msg := tok.CheckTime(opts.Now, 0)
	var sig string
	switch status {
	case jwt.TemporalExpired:
		// msg already starts with "expired …", don't repeat the verb.
		sig = p.red(symFail + " " + msg)
	case jwt.TemporalNotYetValid:
		// msg starts with "not yet valid …".
		sig = p.yellow(symWarn + " " + msg)
	case jwt.TemporalValid:
		// msg starts with "expires in …" — "valid" names the state.
		sig = p.green(symOK+" valid") + " " + p.dim(msg)
	default:
		// msg is "no exp claim (never expires)".
		sig = p.yellow(symWarn + " " + msg)
	}
	return dot + p.bold(alg) + mid + subPart + mid + sig
}
