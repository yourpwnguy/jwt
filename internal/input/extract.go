package input

import (
	"regexp"
	"strings"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// extract.go finds JWTs buried in logs, headers, and curl dumps.
//
// Why regex + validate instead of just regex? A loose pattern matches
// version strings like "1.2.3". Validating each candidate with jwt.Parse
// filters those out, so --extract output is trustworthy without the
// caller thinking about it.

// jwtPattern matches three dot-separated base64url-ish runs.
// The signature may be empty (alg=none yields a trailing dot), so the
// third group allows zero length. Padding "=" is allowed because some
// producers emit it even though RFC 7519 says not to.
//
// Note: no trailing \b — a token ending with "." (empty sig) ends in a
// non-word char, so \b would never match there. The leading \b is enough
// to anchor the start; the character class itself delimits the end.
var jwtPattern = regexp.MustCompile(`\b[A-Za-z0-9_-]+={0,2}\.[A-Za-z0-9_-]+={0,2}\.[A-Za-z0-9_\-+/=]*`)

// ExtractTokens scans text and returns every substring that parses as a JWT.
//
// Duplicates are preserved (log order matters for forensics); callers
// that want uniqueness can dedupe. An empty result is not an error —
// it just means no tokens were found.
func ExtractTokens(text string) []string {
	raw := jwtPattern.FindAllString(text, -1)
	if len(raw) == 0 {
		return nil
	}
	// Pre-size to avoid regrowth; most candidates survive validation.
	out := make([]string, 0, len(raw))
	for _, c := range raw {
		// Trim trailing dots artifacts? No — "h.p." (empty sig) is
		// valid for alg=none, so keep it verbatim.
		candidate := strings.TrimSpace(c)
		if candidate == "" {
			continue
		}
		// Full parse is the cheapest reliable validator and also
		// rejects JWE (5 parts) and non-object segments.
		if _, err := jwt.Parse(candidate); err != nil {
			continue
		}
		out = append(out, candidate)
		if len(out) >= MaxTokens {
			break
		}
	}
	return out
}
