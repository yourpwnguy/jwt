package output

import (
	"encoding/json"
	"fmt"

	"github.com/yourpwnguy/jwt/internal/jwt"
)

// claim.go implements "--claim sub": print one value for scripting.
//
// Why raw strings print unquoted: `USER=$(jwt --claim sub)` must yield
// `123`, not `"123"`. Non-strings print as compact JSON so numbers,
// bools, objects, and arrays all survive round-trips.

// ClaimValue renders a single dot-path claim for stdout.
// It returns the text (without trailing newline) or an error if missing.
func ClaimValue(tok jwt.Token, path string) (string, error) {
	v, ok := tok.Claim(path)
	if !ok {
		return "", fmt.Errorf("claim %q not found", path)
	}
	// Strings print raw — the 99% scripting case.
	if s, ok := v.(string); ok {
		return s, nil
	}
	// Everything else as compact JSON (no indent, no color).
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to encode claim %q: %w", path, err)
	}
	return string(data), nil
}
