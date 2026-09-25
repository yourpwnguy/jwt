// Package audit answers "is this token risky?" without any network
// or secrets. It's pure static analysis over header + payload — safe
// to run on untrusted tokens, in CI, on a whole dump.
//
// Why offline only? Anything needing a secret or a server belongs in
// verify (trust) or a future live-probe. Audit must never exfiltrate
// token contents, so it takes no URLs and makes no requests.
package audit

import (
	"context"
	"time"

	"github.com/yourpwnguy/jwt/internal/jwt"
)

// Severity grades a failed check. Ordering matters: CI gates use
// "fail when severity >= X", so keep Low < Medium < High < Critical.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Rank orders severities for --fail-on comparisons.
func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// Finding is one rule's verdict. Pass=true means "no issue";
// Pass=false means "needs attention" with Detail explaining why.
type Finding struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Severity Severity `json:"severity"`
	Pass     bool     `json:"pass"`
	Detail   string   `json:"detail"`
}

// Report bundles all findings for one token.
type Report struct {
	Findings []Finding `json:"findings"`
}

// Run evaluates every rule. It respects ctx so batch audits over
// thousands of tokens stay cancellable. Rules are pure functions —
// no I/O, no shared state — so Run is safe for concurrent use.
func Run(ctx context.Context, tok jwt.Token, now time.Time) Report {
	if now.IsZero() {
		now = time.Now()
	}
	findings := make([]Finding, 0, len(rules))
	for _, r := range rules {
		if err := ctx.Err(); err != nil {
			break
		}
		pass, detail := r.check(tok, now)
		findings = append(findings, Finding{
			ID: r.id, Title: r.title, Severity: r.severity,
			Pass: pass, Detail: detail,
		})
	}
	return Report{Findings: findings}
}

// FailedAtOrAbove reports whether any failing finding meets the threshold.
// Used for exit codes: --fail-on high fails on high+critical only.
func (r Report) FailedAtOrAbove(min Severity) bool {
	for _, f := range r.Findings {
		if !f.Pass && f.Severity.Rank() >= min.Rank() {
			return true
		}
	}
	return false
}

// FailedCount counts findings needing attention (any severity).
func (r Report) FailedCount() int {
	n := 0
	for _, f := range r.Findings {
		if !f.Pass {
			n++
		}
	}
	return n
}
