package audit

import (
	"strings"
	"time"

	"github.com/yourpwnguy/jwt/internal/jwt"
)

// sensitive.go isolates PII/secret scanning so the keyword list can
// evolve without touching rule plumbing.
//
// Why substring + lowercase? Real claims are messy: "user_password",
// "SSN", "creditCardNumber". Exact matching would miss most leaks;
// substring overcounts slightly, which is the safe direction for a
// scanner — a false "review this" beats a missed password dump.

// highRisk leaks credentials or identifiers that must never sit in a JWT.
var highRiskKeys = []string{"password", "passwd", "pwd", "secret", "ssn", "private_key", "credit", "card_number", "cvv"}

// lowRisk is PII worth knowing about (logs love to leak it) but not
// automatically alarming — hence LOW severity, not MEDIUM.
var lowRiskKeys = []string{"email", "phone", "address", "dob", "birth"}

func checkSensitive(tok jwt.Token, _ time.Time) (bool, string) {
	// Walk payload recursively — sensitive data hides in nested objects
	// like {"user": {"password": "..."}}.
	keys := collectKeys(tok.Payload)
	for _, k := range keys {
		lower := strings.ToLower(k)
		for _, risky := range highRiskKeys {
			if strings.Contains(lower, risky) {
				return false, "payload contains key like \"" + k + "\", never put secrets in JWTs (they are readable by anyone holding the token)"
			}
		}
	}
	for _, k := range keys {
		lower := strings.ToLower(k)
		for _, pii := range lowRiskKeys {
			if strings.Contains(lower, pii) {
				return false, "payload contains PII key \"" + k + "\", ensure logs don't retain tokens"
			}
		}
	}
	return true, "no sensitive keys detected"
}

// collectKeys flattens all map keys recursively. Non-map values are
// ignored; arrays are traversed for objects they may contain.
func collectKeys(v any) []string {
	var out []string
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			out = append(out, k)
			out = append(out, collectKeys(val)...)
		}
	case []any:
		for _, item := range t {
			out = append(out, collectKeys(item)...)
		}
	}
	return out
}
