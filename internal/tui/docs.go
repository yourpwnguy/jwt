package tui

// docs.go is the live property documentation in the inspector card.
// It reads the JSON key on the cursor line (or the nearest key above a
// value line) and explains the standard JWT claim. Custom keys get an
// honest "no standard meaning" note instead of a made-up description.
//
// This is presentation only. It never changes the token and it never
// validates a signature; the audit rules still own that job.

import (
	"strings"
	"unicode"
)

type claimDoc struct {
	typ  string
	text string
}

var headerClaims = map[string]claimDoc{
	"alg":      {"string", "Signing algorithm. HS256/384/512 share a secret; RS/PS/ES/EdDSA use a key pair."},
	"typ":      {"string", `Token type. The JWT convention is "JWT".`},
	"kid":      {"string", "Key ID. Tells the verifier which key to select."},
	"jku":      {"string", "URL for a JWK set. Never fetch it blindly; it is a key-confusion and SSRF risk."},
	"jwk":      {"object", "Embedded public key. The verifier must trust it out of band."},
	"x5u":      {"string", "URL for an X.509 certificate chain. Treat remote URLs as untrusted."},
	"x5c":      {"array", "Embedded X.509 certificate chain."},
	"x5t":      {"string", "SHA-1 thumbprint of the signing certificate."},
	"x5t#S256": {"string", "SHA-256 thumbprint of the signing certificate."},
	"cty":      {"string", "Content type. Needed only for nested JWTs."},
	"crit":     {"array", "Critical headers. Every entry must be understood by the verifier."},
	"enc":      {"string", "JWE content encryption algorithm. This editor does not decrypt JWE."},
	"zip":      {"string", "JWE compression algorithm."},
}

var payloadClaims = map[string]claimDoc{
	"iss":                {"string", "Issuer. Who created the token."},
	"sub":                {"string", "Subject. The user or entity the token is about."},
	"aud":                {"string or array", "Audience. Who the token is meant for."},
	"exp":                {"number", "Expiration time, seconds since the Unix epoch. After this the token is invalid."},
	"nbf":                {"number", "Not before, seconds since the Unix epoch. The token is invalid before this."},
	"iat":                {"number", "Issued at, seconds since the Unix epoch."},
	"jti":                {"string", "Unique token ID. Useful for replay detection and revocation."},
	"scope":              {"string", "OAuth scopes granted to the token."},
	"scp":                {"string or array", "OAuth scopes in the Azure AD form."},
	"client_id":          {"string", "OAuth client that requested the token."},
	"azp":                {"string", "Authorized party. The client the token was issued to."},
	"nonce":              {"string", "OpenID Connect nonce. Binds the token to one login request."},
	"auth_time":          {"number", "When the user actually authenticated."},
	"acr":                {"string", "Authentication context class reference."},
	"amr":                {"array", "Authentication methods used."},
	"sid":                {"string", "Session ID."},
	"name":               {"string", "Display name profile claim."},
	"preferred_username": {"string", "Preferred username profile claim."},
	"email":              {"string", "Email profile claim."},
	"groups":             {"array", "Group membership claim. Authorisation semantics are app-specific."},
	"roles":              {"array", "Role membership claim. Authorisation semantics are app-specific."},
}

// propertyDocs is rendered inside the inspector card. It returns nil
// when there is no key to explain, so the pane does not waste rows on
// an empty hint.
func (m model) propertyDocs(width int) []string {
	if m.section != sectionHeader && m.section != sectionPayload {
		return nil
	}
	key, value := m.keyAtCursor()
	if key == "" {
		return nil
	}
	doc, ok := claimDocFor(m.section, key)
	if !ok {
		doc = claimDoc{
			typ:  "any",
			text: "Custom claim. No standard JWT meaning; document it in your own schema.",
		}
	}

	lines := []string{
		m.theme.bold(m.theme.header).Render("PROPERTY  ") +
			m.theme.bold(m.theme.sectionColor(m.section)).Render(key),
	}
	for _, line := range wrapPlain(doc.text, width) {
		lines = append(lines, m.theme.mutedStyle().Render(line))
	}
	meta := doc.typ
	if meta != "" {
		meta = "type " + meta
	}
	if value != "" {
		if meta != "" {
			meta += " · "
		}
		meta += "value " + value
	}
	if meta != "" {
		lines = append(lines, m.theme.mutedStyle().Render(meta))
	}
	return lines
}

func claimDocFor(s section, key string) (claimDoc, bool) {
	if s == sectionHeader {
		doc, ok := headerClaims[key]
		return doc, ok
	}
	doc, ok := payloadClaims[key]
	return doc, ok
}

// keyAtCursor returns the key that owns the cursor line, plus the value
// written after it when the value is on the same line. If the cursor is
// on a bare value line, we walk upward to the nearest key so an array
// element still explains its array key.
func (m model) keyAtCursor() (string, string) {
	lines := strings.Split(m.activeText(), "\n")
	if len(lines) == 0 {
		return "", ""
	}
	line := clamp(m.activeLine(), 0, len(lines)-1)
	trimmed := strings.TrimSpace(lines[line])
	if strings.HasPrefix(trimmed, "}") || strings.HasPrefix(trimmed, "]") {
		return "", ""
	}
	if key, pos, ok := keyOnLine(lines[line], m.activeCol()); ok {
		return key, valueAfterKey(lines[line], pos)
	}
	for i := line - 1; i >= 0 && line-i <= 40; i-- {
		if key, _, ok := keyOnLine(lines[i], 1<<30); ok {
			return key, ""
		}
	}
	return "", ""
}

// keyOnLine scans one line for JSON member keys. It returns the last
// key at or before col, which keeps compact JSON usable. The position is
// a rune index so the value extractor does not have to reparse.
func keyOnLine(line string, col int) (string, int, bool) {
	runes := []rune(line)
	type found struct {
		name string
		pos  int
	}
	var keys []found
	inString := false
	escape := false
	start := -1

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if inString {
			switch {
			case escape:
				escape = false
			case r == '\\':
				escape = true
			case r == '"':
				j := i + 1
				for j < len(runes) && unicode.IsSpace(runes[j]) {
					j++
				}
				if j < len(runes) && runes[j] == ':' {
					keys = append(keys, found{string(runes[start+1 : i]), start})
				}
				inString = false
			}
			continue
		}
		if r == '"' {
			inString = true
			start = i
		}
	}
	if len(keys) == 0 {
		return "", 0, false
	}
	chosen := keys[0]
	for _, key := range keys {
		if key.pos <= col {
			chosen = key
		}
	}
	return chosen.name, chosen.pos, true
}

func valueAfterKey(line string, keyPos int) string {
	runes := []rune(line)
	colon := -1
	inString := false
	escape := false
	for i := keyPos; i < len(runes); i++ {
		r := runes[i]
		if inString {
			switch {
			case escape:
				escape = false
			case r == '\\':
				escape = true
			case r == '"':
				inString = false
			}
			continue
		}
		if r == '"' {
			inString = true
			continue
		}
		if r == ':' {
			colon = i
			break
		}
	}
	if colon < 0 {
		return ""
	}
	value := strings.TrimSpace(string(runes[colon+1:]))
	value = strings.TrimSuffix(value, ",")
	value = strings.TrimSpace(value)
	out := []rune(value)
	if len(out) > 32 {
		return string(out[:31]) + "…"
	}
	return value
}
