package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourpwnguy/jwt/v2/internal/secret"
	"github.com/yourpwnguy/jwt/v2/internal/sign"
)

// sign.go owns the signing dialog. v1 supports HS256/384/512 and
// alg=none. Asymmetric tokens can still be edited and copied with a
// stale signature, and the Signature tab accepts a pasted signature.
//
// The secret is stored as the original spec (literal or @path), not as
// decoded bytes. It never leaves the process and is never rendered in
// plain text unless the user explicitly reveals it.

func (m *model) startSign() tea.Cmd {
	if m.headerErr != nil || m.payloadErr != nil {
		return m.showToast("fix the JSON first", toastDanger)
	}

	alg := strings.ToUpper(strings.TrimSpace(m.alg))
	if alg == "" {
		return m.showToast("add an alg to the header", toastDanger)
	}
	if alg == "NONE" {
		if m.signature.Value() == "" {
			return m.showToast("already unsigned", toastInfo)
		}
		m.signature.SetValue("")
		m.refresh()
		return m.showToast("signature cleared (alg none)", toastWarning)
	}
	if alg != "HS256" && alg != "HS384" && alg != "HS512" {
		return m.showToast(alg+" signing is not in this build, paste a signature instead", toastWarning)
	}

	m.signReturn = m.mode
	m.signErr = nil
	m.revealSecret = false
	m.secret.EchoMode = textinput.EchoPassword
	m.secret.SetValue(m.secretSpec)
	m.secret.CursorEnd()
	m.mode = modeSign
	m.blurAll()
	return m.secret.Focus()
}

func (m *model) handleSignKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = m.signReturn
		m.signErr = nil
		if m.mode == modeEdit {
			return m.focusSection()
		}
		m.blurAll()
		return nil
	case "ctrl+r":
		m.revealSecret = !m.revealSecret
		if m.revealSecret {
			m.secret.EchoMode = textinput.EchoNormal
		} else {
			m.secret.EchoMode = textinput.EchoPassword
		}
		return nil
	case "enter":
		return m.submitSign()
	default:
		return m.updateSecret(msg)
	}
}

func (m *model) submitSign() tea.Cmd {
	// Do not trim the secret. HMAC keys are arbitrary bytes, and a
	// literal secret with leading or trailing spaces is legal. The
	// verify CLI treats it exactly the same way.
	spec := m.secret.Value()
	if spec == "" {
		m.signErr = fmt.Errorf("secret is empty")
		return nil
	}

	key, err := secret.LoadHMAC(spec)
	if err != nil {
		m.signErr = err
		return nil
	}

	alg := strings.ToUpper(strings.TrimSpace(m.alg))
	sig, err := sign.HMAC(m.signingInput, alg, key)
	if err != nil {
		m.signErr = err
		return nil
	}

	m.signature.SetValue(sig)
	m.originInput = m.signingInput
	m.originSig = sig
	m.originKind = originSigned
	m.secretSpec = spec
	m.signErr = nil
	m.mode = m.signReturn
	m.refresh()

	toast := m.showToast("signed "+alg, toastSuccess)
	stamp := m.startStamp()
	if m.mode == modeEdit {
		return tea.Batch(toast, m.focusSection(), stamp)
	}
	m.blurAll()
	return tea.Batch(toast, stamp)
}
