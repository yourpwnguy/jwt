package tui

// view_editor.go renders the editor pane. Header and payload are the
// textarea widgets, given as much room as the page allows and no box
// around them. The signature pane is a composed page instead: state,
// stamp, input, and the exact-byte facts.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderEditorPane(totalW, totalH int) []string {
	if totalW < 6 || totalH < 1 {
		return nil
	}
	innerW := totalW - 2
	var content []string
	if m.section == sectionSignature {
		content = m.signatureContent(innerW, totalH)
	} else {
		view := m.header.View()
		if m.section == sectionPayload {
			view = m.payload.View()
		}
		content = strings.Split(strings.TrimRight(view, "\n"), "\n")
	}
	content = fitAndPad(content, innerW, totalH)
	lines := make([]string, totalH)
	for i, line := range content {
		lines[i] = " " + line + " "
	}
	return lines
}

// signatureContent builds a document page: the state gets the top of
// the page, the base64 field is the body, and the byte facts are the
// footnote. Optional sections drop away as the window gets shorter,
// but the input is always visible.
func (m model) signatureContent(width, height int) []string {
	state := strings.ToUpper(m.sigState.label())
	color := m.sigColor()
	blurb := m.signatureBlurb()
	if m.headerErr != nil || m.payloadErr != nil {
		state = "UNAVAILABLE"
		color = m.theme.danger
	} else if m.sigErr != nil {
		state = "INVALID"
	}

	lines := make([]string, 0, height)
	if height >= 14 {
		lines = append(lines, m.theme.mutedStyle().Render("SIGNATURE  /  BASE64URL"))
	}

	// The state and stamp share the top of the page. On narrow panes the
	// stamp folds into a single glyph so the words keep their room.
	stampW := 7
	showStamp := width >= 34 && height >= 12
	leftW := width
	if showStamp {
		leftW = width - stampW - 2
	}

	left := []string{
		m.theme.bold(color).Render(state),
		m.theme.fg(color).Render(strings.Repeat("━", min(max(10, len(state)), leftW))),
	}
	if leftW >= 20 {
		wrapped := wrapPlain(blurb, leftW)
		if height < 16 && len(wrapped) > 1 {
			wrapped = wrapped[:1]
		}
		for _, line := range wrapped {
			left = append(left, m.theme.mutedStyle().Render(line))
		}
	}
	if height < 12 {
		left[0] += "  " + m.theme.fg(color).Render(sigGlyph(m.sigState))
	}

	if showStamp {
		stamp := m.signatureStamp()
		h := max(len(left), len(stamp))
		for i := 0; i < h; i++ {
			l := ""
			if i < len(left) {
				l = left[i]
			}
			r := ""
			if i < len(stamp) {
				r = stamp[i]
			}
			lines = append(lines, padLine(l, leftW)+"  "+r)
		}
	} else {
		lines = append(lines, left...)
	}

	lines = append(lines, "")
	if height >= 8 {
		lines = append(lines, m.theme.mutedStyle().Render("BASE64URL"))
	}
	lines = append(lines, padLine(m.signature.View(), width))
	lines = append(lines, m.rule(width, m.theme.hairline))

	if height >= 18 {
		lines = append(lines, "")
	}
	if height >= 14 {
		lines = append(lines, m.signatureFacts(width)...)
	}
	if height >= 10 {
		lines = append(lines, "")
		lines = append(lines, m.theme.mutedStyle().Render("ctrl+s sign · paste a signature to replace"))
	}
	return fitAndPad(lines, width, height)
}

func (m model) signatureFacts(width int) []string {
	sig := m.signature.Value()
	origin := "loaded token"
	switch {
	case m.sigState == sigSigned:
		origin = "this session"
	case m.sigState == sigCustom:
		origin = "pasted value"
	case sig == "":
		origin = "none"
	}
	rows := [][2]string{
		{"BYTES", fmt.Sprintf("%d base64url", len(sig))},
		{"SIGNING INPUT", humanBytes(len(m.signingInput))},
		{"ORIGIN", origin},
	}
	if width >= 28 {
		alg := m.alg
		if alg == "" {
			alg = "missing"
		}
		rows = append(rows, [2]string{"ALG", alg})
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		label := m.theme.mutedStyle().Render(fmt.Sprintf("%-14s", row[0]))
		value := m.theme.fg(m.theme.ink).Render(truncatePlain(row[1], max(4, width-14)))
		out = append(out, label+value)
	}
	return out
}

func (m model) signatureBlurb() string {
	if m.headerErr != nil || m.payloadErr != nil {
		return "Fix the JSON to restore the signature state."
	}
	switch m.sigState {
	case sigSigned:
		return "Signed in this session over the exact header and payload."
	case sigStale:
		return "Header or payload changed after signing. Sign again."
	case sigCustom:
		return "Pasted signature. Its origin is unknown."
	case sigUnsigned:
		return "No signature is attached (alg none or empty)."
	case sigInvalid:
		return "Not valid base64url. Fix the value before copying."
	default:
		return "Loaded with the token. This editor has not checked it."
	}
}

// signatureStamp is the notary mark: the one bordered shape on the
// page. It flashes inverted for a few frames after a successful sign.
func (m model) signatureStamp() []string {
	glyph := sigGlyph(m.sigState)
	color := m.sigColor()
	if m.headerErr != nil || m.payloadErr != nil {
		color = m.theme.danger
		glyph = "!"
	}
	box := []string{
		"╭─────╮",
		"│  " + glyph + "  │",
		"╰─────╯",
	}
	if m.stamp > 0 && m.sigState == sigSigned {
		style := lipgloss.NewStyle().
			Background(fillSuccess).
			Foreground(inkOnFill).
			Bold(true)
		return renderBlock(box, style)
	}
	return renderBlock(box, m.theme.fg(color))
}

func sigGlyph(s signatureState) string {
	switch s {
	case sigSigned:
		return "✓"
	case sigStale, sigInvalid:
		return "✗"
	case sigCustom:
		return "◆"
	case sigUnsigned:
		return "∅"
	default:
		return "◇"
	}
}

func truncatePlain(s string, width int) string {
	if len(s) <= width {
		return s
	}
	if width <= 1 {
		return s[:width]
	}
	return s[:width-1] + "…"
}
