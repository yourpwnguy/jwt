package jwt

import (
	"fmt"
	"time"
)

// time.go makes numeric dates human. Nobody can read 1716239022,
// but everybody understands "expired 2h ago".
//
// All functions take an explicit `now` so tests don't depend on the
// wall clock — passing time in is boringly reliable, using time.Now()
// deep inside logic is how flaky tests are born.

// TemporalStatus describes where a token sits in time.
type TemporalStatus int

const (
	// TemporalNoExp means no "exp" — token never expires by itself.
	TemporalNoExp TemporalStatus = iota
	// TemporalValid means now is within [nbf, exp] (with leeway applied).
	TemporalValid
	// TemporalExpired means now is past exp + leeway.
	TemporalExpired
	// TemporalNotYetValid means now is before nbf - leeway.
	TemporalNotYetValid
)

// CheckTime evaluates exp/nbf against now with leeway.
//
// Leeway exists because clocks disagree — 30s is sane, 0 is strict,
// >5m deserves a warning (you're hiding real skew at that point).
// Missing exp is NOT an error here; it's reported as TemporalNoExp
// and left to the audit package to flag as policy.
func (t Token) CheckTime(now time.Time, leeway time.Duration) (TemporalStatus, string) {
	nowUnix := now.Unix()
	if nbf, ok := t.Nbf(); ok && nowUnix < nbf-int64(leeway.Seconds()) {
		return TemporalNotYetValid, fmt.Sprintf("not yet valid, starts %s (%s)", FormatUnix(nbf), Humanize(nbf-nowUnix))
	}
	if exp, ok := t.Exp(); ok && nowUnix > exp+int64(leeway.Seconds()) {
		return TemporalExpired, fmt.Sprintf("expired %s ago (%s)", Humanize(nowUnix-exp), FormatUnix(exp))
	}
	if exp, ok := t.Exp(); ok {
		return TemporalValid, fmt.Sprintf("expires in %s (%s)", Humanize(exp-nowUnix), FormatUnix(exp))
	}
	return TemporalNoExp, "no exp claim (never expires)"
}

// FormatUnix renders seconds-since-epoch as RFC3339 UTC.
// Returns "invalid time" rather than zero-time weirdness on garbage.
func FormatUnix(sec int64) string {
	if sec < 0 || sec > 4102444800 { // year 2100 sanity bound
		return "invalid time"
	}
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}

// Humanize turns seconds into "14m", "2h", "3d" — short, no plurals drama.
// Used for "expires in X" / "expired X ago" summaries.
func Humanize(sec int64) string {
	if sec < 0 {
		sec = -sec
	}
	switch {
	case sec < 60:
		return fmt.Sprintf("%ds", sec)
	case sec < 3600:
		return fmt.Sprintf("%dm", sec/60)
	case sec < 86400:
		return fmt.Sprintf("%dh", sec/3600)
	default:
		return fmt.Sprintf("%dd", sec/86400)
	}
}
