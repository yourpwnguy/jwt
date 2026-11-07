package tui

// load.go turns a parsed token into editor text and back. It is the
// only place that knows about original segments and pretty printing.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

func (m *model) loadToken(raw string) error {
	tok, err := jwt.Parse(raw)
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	parts := strings.Split(tok.Raw, ".")
	if len(parts) != 3 {
		return fmt.Errorf("tui: token has %d parts, want 3", len(parts))
	}

	header, err := prettySegment(parts[0])
	if err != nil {
		return fmt.Errorf("tui: header: %w", err)
	}
	payload, err := prettySegment(parts[1])
	if err != nil {
		return fmt.Errorf("tui: payload: %w", err)
	}

	m.header.SetValue(header)
	m.payload.SetValue(payload)
	m.signature.SetValue(parts[2])
	m.loaded = true
	m.originalHeader = parts[0]
	m.originalPayload = parts[1]
	m.headerDirty = false
	m.payloadDirty = false
	m.originInput = parts[0] + "." + parts[1]
	m.originSig = parts[2]
	m.originKind = originUntouched
	m.alg = tok.Alg()
	return nil
}

func (m *model) loadBlank() {
	m.header.SetValue(blankHeader)
	m.payload.SetValue(blankPayload)
	m.signature.SetValue("")
	m.loaded = false
	m.headerDirty = false
	m.payloadDirty = false
	m.originInput = ""
	m.originSig = ""
	m.originKind = originUntouched
	m.alg = "HS256"
}

// prettySegment turns one base64url JSON segment into indented JSON.
// json.Indent is used instead of unmarshal/marshal because it preserves
// key order and duplicate keys.
func prettySegment(seg string) (string, error) {
	data, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// segment returns the exact base64 segment for one pane. Untouched
// panes keep the token's original bytes; edited panes are encoded from
// the text currently on screen.
func (m model) segment(s section) string {
	switch s {
	case sectionHeader:
		if m.loaded && !m.headerDirty {
			return m.originalHeader
		}
		return encodeSegment(m.header.Value())
	case sectionPayload:
		if m.loaded && !m.payloadDirty {
			return m.originalPayload
		}
		return encodeSegment(m.payload.Value())
	case sectionSignature:
		return m.signature.Value()
	default:
		return ""
	}
}

func encodeSegment(text string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(text))
}

// focusSection blurs every widget and focuses the active one. The
// returned command starts the cursor blink.
