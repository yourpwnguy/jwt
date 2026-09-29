package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// table.go renders batch overviews: one row per token.
//
// Why a hand-rolled table instead of a library? Our columns are fixed
// (#, alg, sub, exp, status) and widths are small. A dependency would
// buy nothing but version churn. Fixed-width with fmt is boring and
// reliable — exactly what a table should be.

// formatTable builds the batch table. It never errors: unparseable
// fields render as "-" so one weird token doesn't kill the overview.
func formatTable(toks []jwt.Token, opts Options) string {
	p := newPalette(opts.NoColor)
	rows := make([][5]string, 0, len(toks))
	for i, tok := range toks {
		alg := tok.Alg()
		if alg == "" {
			alg = "-"
		}
		sub := "-"
		if v, ok := tok.Claim("sub"); ok {
			sub = shortValue(v, 24)
		}
		exp := "-"
		if e, ok := tok.Exp(); ok {
			exp = shortDate(e)
		}
		status, _ := tok.CheckTime(opts.Now, 0)
		statusStr := colorStatus(p, status)
		rows = append(rows, [5]string{
			fmt.Sprintf("%d", i+1), alg, sub, exp, statusStr,
		})
	}
	// Column widths from content so output stays compact.
	widths := [5]int{len("#"), len("alg"), len("sub"), len("exp"), len("status")}
	for _, r := range rows {
		for c, v := range r {
			if len(stripANSI(v)) > widths[c] {
				widths[c] = len(stripANSI(v))
			}
		}
	}
	var b strings.Builder
	header := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %-*s",
		widths[0], "#", widths[1], "alg", widths[2], "sub", widths[3], "exp", widths[4], "status")
	b.WriteString(p.gray(header) + "\n")
	b.WriteString(p.rule(46) + "\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-*s  %-*s  %-*s  %-*s  %s\n",
			widths[0], r[0], widths[1], r[1], widths[2], r[2], widths[3], r[3], r[4])
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// shortValue renders a claim for a narrow column.
func shortValue(v any, max int) string {
	var s string
	switch t := v.(type) {
	case string:
		s = t
	default:
		s = fmt.Sprintf("%v", t)
	}
	if len(s) > max {
		return s[:max-1] + "…"
	}
	if s == "" {
		return "-"
	}
	return s
}

// colorStatus maps temporal state to signal colors with symbols.
// Text keeps plain words (valid/expired) so grep still works.
func colorStatus(p palette, s jwt.TemporalStatus) string {
	switch s {
	case jwt.TemporalExpired:
		return p.red(symFail + " expired")
	case jwt.TemporalNotYetValid:
		return p.yellow(symWarn + " not-yet-valid")
	case jwt.TemporalValid:
		return p.green(symOK + " valid")
	default:
		return p.yellow(symWarn + " no-exp")
	}
}

// shortDate renders unix seconds as YYYY-MM-DD. Tables need scannable
// dates, not full RFC3339 timestamps — details live in decode output.
func shortDate(sec int64) string {
	if sec < 0 || sec > 4102444800 {
		return "-"
	}
	return time.Unix(sec, 0).UTC().Format("2006-01-02")
}

// stripANSI removes our own color wrappers for width math.
// We only emit SGR sequences via gostyle, so a minimal scanner
// for "\x1b[...m" is enough — no regex, no allocation-heavy lib.
func stripANSI(s string) string {
	if !strings.Contains(s, "\x1b[") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	inEsc := false
	for i := 0; i < len(s); i++ {
		if !inEsc && s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			inEsc = true
			i++
			continue
		}
		if inEsc {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
