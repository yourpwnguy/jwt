package tui

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yourpwnguy/jwt/v2/internal/audit"
	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// refresh.go is the single place that turns editor text into token
// state. Every keystroke, format, sign, and resize path ends here, so
// the token, signature badge, and audit can never disagree.

// refresh validates both JSON panes, assembles the current token, and
// runs the audit when the token parses. Invalid JSON keeps the last
// encodable token visible instead of blinking the inspector empty.
func (m *model) refresh() {
	m.headerErr = validateObject(m.header.Value(), "header")
	m.payloadErr = validateObject(m.payload.Value(), "payload")
	m.sigErr = validateSignature(m.signature.Value())

	if m.headerErr != nil || m.payloadErr != nil {
		m.currentToken = ""
		m.signingInput = ""
		m.parsedOK = false
		m.auditOK = false
		m.tokenErr = nil
		if m.headerErr == nil {
			m.alg = algFromHeader(m.header.Value())
		} else {
			m.alg = ""
		}
		m.sigState = sigInvalid
		m.expStatus = jwt.TemporalNoExp
		m.expMessage = ""
		return
	}

	m.signingInput = m.segment(sectionHeader) + "." + m.segment(sectionPayload)
	m.currentToken = m.signingInput + "." + m.segment(sectionSignature)
	m.lastValidToken = m.currentToken

	if len(m.currentToken) > jwt.MaxTokenSize {
		m.tokenErr = fmt.Errorf("token is %s, limit is %s", humanBytes(len(m.currentToken)), humanBytes(jwt.MaxTokenSize))
		m.parsedOK = false
		m.auditOK = false
		m.alg = algFromHeader(m.header.Value())
		m.sigState = m.computeSigState()
		return
	}

	tok, err := jwt.Parse(m.currentToken)
	if err != nil {
		m.tokenErr = err
		m.parsedOK = false
		m.auditOK = false
		m.alg = algFromHeader(m.header.Value())
		m.sigState = m.computeSigState()
		m.expStatus = jwt.TemporalNoExp
		m.expMessage = ""
		return
	}

	m.tokenErr = nil
	m.parsed = tok
	m.parsedOK = true
	m.alg = tok.Alg()
	m.runAudit(m.now())
	m.sigState = m.computeSigState()
}

// runAudit evaluates the offline rules against a parsed token. It is
// cheap enough to run on every keystroke and every one-second tick.
func (m *model) runAudit(now time.Time) {
	if !m.parsedOK {
		m.auditOK = false
		return
	}
	m.report = audit.Run(m.ctx, m.parsed, now)
	m.auditOK = true
	m.expStatus, m.expMessage = m.parsed.CheckTime(now, 0)
}

func (m *model) tick(now time.Time) {
	if !m.parsedOK {
		return
	}
	m.runAudit(now)
}

// computeSigState compares the current signing input and signature
// against the last origin. It never claims a signature is valid: this
// package has no key unless the user just signed with one.
func (m model) computeSigState() signatureState {
	sig := m.signature.Value()
	if m.sigErr != nil {
		return sigInvalid
	}
	if strings.EqualFold(m.alg, "none") || sig == "" {
		return sigUnsigned
	}
	if m.signingInput != m.originInput {
		return sigStale
	}
	if sig != m.originSig {
		return sigCustom
	}
	if m.originKind == originSigned {
		return sigSigned
	}
	return sigUntouched
}

// validateObject rejects anything that is not a JSON object and caps
// the segment size before allocating. The parser has the same limit,
// but checking here gives a friendly editor error instead of a parse
// failure deep in the token.
func validateObject(text, name string) error {
	if len(text) > jwt.MaxSegmentSize {
		return fmt.Errorf("%s is %s, limit is %s", name, humanBytes(len(text)), humanBytes(jwt.MaxSegmentSize))
	}
	var v any
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return jsonError(text, err)
	}
	if v == nil {
		return fmt.Errorf("%s is JSON null, want an object", name)
	}
	if _, ok := v.(map[string]any); !ok {
		return fmt.Errorf("%s must be a JSON object", name)
	}
	return nil
}

// validateSignature checks the third segment is base64url text. Empty
// is allowed because alg=none and unsigned test tokens exist.
func validateSignature(sig string) error {
	if sig == "" {
		return nil
	}
	if len(sig) > jwt.MaxSegmentSize {
		return fmt.Errorf("signature is %s, limit is %s", humanBytes(len(sig)), humanBytes(jwt.MaxSegmentSize))
	}
	if strings.ContainsAny(sig, " \t\r\n.") {
		return fmt.Errorf("signature must be base64url, no dots or spaces")
	}
	if _, err := base64.RawURLEncoding.DecodeString(sig); err != nil {
		return fmt.Errorf("signature is not valid base64url: %v", err)
	}
	return nil
}

// jsonError turns a byte offset into a line and column a human can find.
// Columns are byte-based; JSON syntax errors are ASCII in practice.
func jsonError(text string, err error) error {
	var syn *json.SyntaxError
	if !errors.As(err, &syn) {
		return err
	}
	line, col := lineCol(text, syn.Offset)
	return fmt.Errorf("line %d, col %d: %s", line, col, syn.Error())
}

func lineCol(text string, offset int64) (line, col int) {
	if offset < 1 {
		offset = 1
	}
	if offset > int64(len(text))+1 {
		offset = int64(len(text)) + 1
	}
	line, col = 1, 1
	for i := int64(0); i < offset-1; i++ {
		if text[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

// algFromHeader gets a useful badge value even when the signature
// segment is malformed and full parsing fails.
func algFromHeader(text string) string {
	var h struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal([]byte(text), &h) != nil {
		return ""
	}
	return h.Alg
}

func humanBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f KB", float64(n)/1024)
}

// tokenProblems collects everything that currently prevents a full
// audit. The tab row shows the active one; the full audit screen shows
// all of them.
func (m model) tokenProblems() []string {
	var out []string
	if m.headerErr != nil {
		out = append(out, "header: "+m.headerErr.Error())
	}
	if m.payloadErr != nil {
		out = append(out, "payload: "+m.payloadErr.Error())
	}
	if m.sigErr != nil {
		out = append(out, "signature: "+m.sigErr.Error())
	}
	if m.tokenErr != nil {
		out = append(out, "token: "+m.tokenErr.Error())
	}
	return out
}
