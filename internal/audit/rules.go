package audit

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// rules.go holds the actual checks. Each rule is a tiny pure function —
// easy to test in isolation, easy to add without touching the runner.
//
// Philosophy: flag structure, not intent. We can't know if a 30-day
// token is "wrong", but we CAN say "this lives 30 days, that's a big
// window if stolen". Details educate; severities prioritize.

// rule bundles metadata with its check. Checks take now for time-aware
// rules (iat skew) but most ignore it.
type rule struct {
	id       string
	title    string
	severity Severity
	check    func(tok jwt.Token, now time.Time) (pass bool, detail string)
}

// LongLivedThreshold is the lifetime above which we warn. 24h is a
// common session ceiling; anything longer deserves a conscious choice.
const LongLivedThreshold = 24 * time.Hour

// rules runs in this order: most critical first so human tables read
// like a triage list, not alphabet soup.
var rules = []rule{
	{id: "none-alg", title: "Unsigned (alg=none) token", severity: SeverityCritical, check: checkNoneAlg},
	{id: "weak-alg", title: "Weak or unknown algorithm", severity: SeverityMedium, check: checkWeakAlg},
	{id: "missing-exp", title: "Missing expiration", severity: SeverityHigh, check: checkMissingExp},
	{id: "long-lived", title: "Excessive lifetime", severity: SeverityMedium, check: checkLongLived},
	{id: "jku-x5u", title: "External key URL (jku/x5u)", severity: SeverityHigh, check: checkKeyURL},
	{id: "kid", title: "Key ID (kid) present", severity: SeverityMedium, check: checkKid},
	{id: "sensitive-claim", title: "Sensitive data in payload", severity: SeverityMedium, check: checkSensitive},
	{id: "time-skew", title: "Suspicious timestamps", severity: SeverityLow, check: checkSkew},
}

func checkNoneAlg(tok jwt.Token, _ time.Time) (bool, string) {
	alg := tok.Alg()
	if alg == "" {
		return false, "missing alg header, validators may default insecurely"
	}
	// EqualFold catches "None"/"NONE" variants that break naive validators.
	if strings.EqualFold(alg, "none") {
		return false, fmt.Sprintf("alg=%q accepts unsigned tokens", alg)
	}
	return true, fmt.Sprintf("alg=%q requires a signature", alg)
}

func checkWeakAlg(tok jwt.Token, _ time.Time) (bool, string) {
	alg := strings.ToUpper(tok.Alg())
	switch alg {
	case "HS256", "HS384", "HS512",
		"RS256", "RS384", "RS512",
		"ES256", "ES384", "ES512",
		"PS256", "PS384", "PS512",
		"EDDSA", "ED25519":
		if strings.HasPrefix(alg, "HS") {
			return true, fmt.Sprintf("alg=%q is HMAC, ensure the secret is long and random (offline check cannot verify)", tok.Alg())
		}
		return true, fmt.Sprintf("alg=%q is standard", tok.Alg())
	case "NONE", "":
		return true, "covered by none-alg rule"
	default:
		return false, fmt.Sprintf("alg=%q is unknown, validators may mishandle it", tok.Alg())
	}
}

func checkMissingExp(tok jwt.Token, _ time.Time) (bool, string) {
	if _, ok := tok.Exp(); !ok {
		return false, "no exp claim, stolen tokens never expire"
	}
	return true, "exp claim present"
}

func checkLongLived(tok jwt.Token, _ time.Time) (bool, string) {
	exp, okExp := tok.Exp()
	iat, okIat := tok.Iat()
	if !okExp {
		return true, "no exp — covered by missing-exp rule"
	}
	if !okIat {
		return true, "no iat — cannot assess lifetime"
	}
	lifetime := exp - iat
	if lifetime < 0 {
		return false, "exp is before iat, malformed timestamps"
	}
	if time.Duration(lifetime)*time.Second > LongLivedThreshold {
		return false, fmt.Sprintf("lifetime %s exceeds %s, narrow the window", jwt.Humanize(lifetime), LongLivedThreshold)
	}
	return true, fmt.Sprintf("lifetime %s within %s", jwt.Humanize(lifetime), LongLivedThreshold)
}

func checkKeyURL(tok jwt.Token, _ time.Time) (bool, string) {
	for _, k := range []string{"jku", "x5u", "x5c", "jwk"} {
		if _, ok := tok.Header[k]; ok {
			return false, fmt.Sprintf("header %q points to external key material, SSRF and spoof surface, pin keys instead", k)
		}
	}
	return true, "no external key references"
}

func checkKid(tok jwt.Token, _ time.Time) (bool, string) {
	if kid := tok.Kid(); kid != "" {
		return false, fmt.Sprintf("kid=%q present, ensure server maps it safely (no path traversal or SQLi)", kid)
	}
	return true, "no kid header"
}

func checkSkew(tok jwt.Token, now time.Time) (bool, string) {
	if iat, ok := tok.Iat(); ok && iat > now.Unix()+60 {
		return false, fmt.Sprintf("iat %s is in the future, clock skew or forged token", jwt.FormatUnix(iat))
	}
	return true, "timestamps look sane"
}
