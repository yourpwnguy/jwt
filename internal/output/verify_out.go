package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
	"github.com/yourpwnguy/jwt/v2/internal/verify"
)

// verify_out.go renders trust decisions. Human mode shows the token
// plus two verdict lines; JSON mode merges verdicts into the token
// object so `jq .signatureValid` works in scripts.

// verifyJSON is the machine shape: token fields plus verdicts.
// Keeping header/payload/signature keys identical to decode output
// means one jq filter works for both commands.
type verifyJSON struct {
	Header         map[string]any `json:"header"`
	Payload        map[string]any `json:"payload"`
	Signature      string         `json:"signature"`
	SignatureValid bool           `json:"signatureValid"`
	ClaimsValid    bool           `json:"claimsValid"`
	ClaimsReason   string         `json:"claimsReason,omitempty"`
}

// FormatVerify renders one verification.
// pretty = verdicts + JSON context. table = verdicts only (no JSON dump,
// useful for batch triage). json/compact = machine shapes for scripts.
func FormatVerify(tok jwt.Token, res verify.Result, format Format, noColor bool) (string, error) {
	switch format {
	case FormatJSON, FormatCompact:
		v := verifyJSON{
			Header: tok.Header, Payload: tok.Payload, Signature: tok.Signature,
			SignatureValid: res.SignatureValid, ClaimsValid: res.ClaimsValid, ClaimsReason: res.ClaimsReason,
		}
		if format == FormatCompact {
			data, err := json.Marshal(v)
			return string(data), err
		}
		data, err := json.MarshalIndent(v, "", "  ")
		return string(data), err
	case FormatTable:
		return formatVerifyTable(tok, res, noColor), nil
	default:
		return formatVerifyHuman(tok, res, noColor), nil
	}
}

// formatVerifyHuman leads with verdicts (the point), then a rule, then
// dimmed JSON (context). Text keeps "Signature: VALID" / "Claims: VALID"
// phrasing so grep and old eyes keep working — symbols are a prefix.
func formatVerifyHuman(tok jwt.Token, res verify.Result, noColor bool) string {
	p := newPalette(noColor)
	var b strings.Builder
	if res.SignatureValid {
		fmt.Fprintf(&b, "%s %s   %s\n", p.green(symOK), p.green("Signature: VALID"), p.gray("· "+res.Alg))
	} else {
		fmt.Fprintf(&b, "%s %s %s\n", p.red(symFail), p.red("Signature: INVALID"), p.gray("· "+res.Alg))
	}
	if res.ClaimsValid {
		line := p.green(symOK) + " " + p.green("Claims: VALID")
		if res.ClaimsReason != "" {
			line += "      " + p.gray("· "+res.ClaimsReason)
		}
		b.WriteString(line + "\n")
	} else {
		reason := res.ClaimsReason
		if reason == "" {
			reason = "invalid"
		}
		fmt.Fprintf(&b, "%s %s    %s\n", p.red(symFail), p.red("Claims: INVALID"), p.yellow(reason))
	}
	b.WriteString(p.rule(40) + "\n")
	data, _ := json.MarshalIndent(view{tok.Header, tok.Payload, tok.Signature}, "", "  ")
	b.WriteString(p.dim(string(data)))
	return b.String()
}

// formatVerifyTable renders verdicts without the JSON dump: two lines,
// greppable, batch-friendly. Pretty shows context; table shows signal.
func formatVerifyTable(tok jwt.Token, res verify.Result, noColor bool) string {
	p := newPalette(noColor)
	var b strings.Builder
	if res.SignatureValid {
		fmt.Fprintf(&b, "%s %s   %s\n", p.green(symOK), p.green("Signature: VALID"), p.gray("· "+res.Alg))
	} else {
		fmt.Fprintf(&b, "%s %s %s\n", p.red(symFail), p.red("Signature: INVALID"), p.gray("· "+res.Alg))
	}
	if res.ClaimsValid {
		line := p.green(symOK) + " " + p.green("Claims: VALID")
		if res.ClaimsReason != "" {
			line += "      " + p.gray("· "+res.ClaimsReason)
		}
		b.WriteString(line)
	} else {
		reason := res.ClaimsReason
		if reason == "" {
			reason = "invalid"
		}
		fmt.Fprintf(&b, "%s %s    %s", p.red(symFail), p.red("Claims: INVALID"), p.yellow(reason))
	}
	return b.String()
}
