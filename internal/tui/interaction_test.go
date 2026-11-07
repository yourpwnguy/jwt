package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
	"github.com/yourpwnguy/jwt/v2/internal/verify"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	case "ctrl+y":
		return tea.KeyMsg{Type: tea.KeyCtrlY}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+h":
		return tea.KeyMsg{Type: tea.KeyCtrlH}
	case "ctrl+j":
		return tea.KeyMsg{Type: tea.KeyCtrlJ}
	case "ctrl+k":
		return tea.KeyMsg{Type: tea.KeyCtrlK}
	case "ctrl+l":
		return tea.KeyMsg{Type: tea.KeyCtrlL}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func send(t *testing.T, m model, key string) model {
	t.Helper()
	updated, _ := m.Update(keyMsg(key))
	return updated.(model)
}

func setText(m *model, s section, text string) {
	m.section = s
	m.setSectionText(text)
	m.refresh()
}

func TestVimMotionsMoveTheCursor(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	for m.header.Line() > 0 {
		m.header.CursorUp()
	}
	m.header.CursorStart()

	m = send(t, m, "ctrl+l")
	if got := m.header.LineInfo().ColumnOffset; got != 1 {
		t.Fatalf("ctrl+l column = %d, want 1", got)
	}
	m = send(t, m, "ctrl+h")
	if got := m.header.LineInfo().ColumnOffset; got != 0 {
		t.Fatalf("ctrl+h column = %d, want 0", got)
	}

	m = send(t, m, "ctrl+j")
	if got := m.header.Line(); got != 1 {
		t.Fatalf("ctrl+j line = %d, want 1", got)
	}
	m = send(t, m, "ctrl+k")
	if got := m.header.Line(); got != 0 {
		t.Fatalf("ctrl+k line = %d, want 0", got)
	}

	// Arrows are an addition, not a replacement, so they still move.
	m = send(t, m, "down")
	if got := m.header.Line(); got != 1 {
		t.Fatalf("down line = %d, want 1", got)
	}
	m = send(t, m, "up")
	if got := m.header.Line(); got != 0 {
		t.Fatalf("up line = %d, want 0", got)
	}
	m = send(t, m, "right")
	if got := m.header.LineInfo().ColumnOffset; got != 1 {
		t.Fatalf("right column = %d, want 1", got)
	}
	m = send(t, m, "left")
	if got := m.header.LineInfo().ColumnOffset; got != 0 {
		t.Fatalf("left column = %d, want 0", got)
	}
}

func TestCtrlIStaysTab(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "tab") // KeyTab is the same byte as ctrl+i.
	if m.section != sectionPayload {
		t.Fatalf("ctrl+i changed section to %v, want payload", m.section)
	}
}

func TestVimMotionsReachSignatureField(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.section = sectionSignature
	m.focusSection()
	m.signature.SetCursor(0)

	m = send(t, m, "ctrl+l")
	if got := m.signature.Position(); got != 1 {
		t.Fatalf("ctrl+l signature position = %d, want 1", got)
	}
	m = send(t, m, "ctrl+h")
	if got := m.signature.Position(); got != 0 {
		t.Fatalf("ctrl+h signature position = %d, want 0", got)
	}
}

func TestVimMotionsCanBeDisabled(t *testing.T) {
	m, err := newModel(context.Background(), Options{
		Token:     testToken,
		NoVimKeys: true,
		Now:       func() time.Time { return fixedNow },
		Copy:      func(string) error { return nil },
	})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	m.resize(120, 32)
	m.header.CursorStart()

	m = send(t, m, "ctrl+l")
	if got := m.header.LineInfo().ColumnOffset; got != 0 {
		t.Fatalf("ctrl+l moved with vim keys off, column = %d", got)
	}
	m = send(t, m, "right")
	if got := m.header.LineInfo().ColumnOffset; got != 1 {
		t.Fatalf("arrow did not move with vim keys off, column = %d", got)
	}
}

func TestEditMarksSignatureStale(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "tab")
	if m.section != sectionPayload {
		t.Fatalf("section = %v, want payload", m.section)
	}
	m = send(t, m, " ")
	if m.payloadErr != nil {
		t.Fatalf("leading whitespace should stay valid JSON: %v", m.payloadErr)
	}
	if m.sigState != sigStale {
		t.Fatalf("sigState = %v, want stale", m.sigState)
	}
	if m.currentToken == testToken {
		t.Fatal("token did not change after editing the payload")
	}
	if !strings.Contains(m.View(), "STALE") {
		t.Fatal("view should show STALE")
	}
}

func TestInvalidJSONKeepsLastValidToken(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "tab")
	m = send(t, m, "x")

	if m.payloadErr == nil {
		t.Fatal("payload should be invalid")
	}
	if m.currentToken != "" {
		t.Fatalf("currentToken = %q, want empty while invalid", m.currentToken)
	}
	if m.previewToken() != testToken {
		t.Fatalf("preview = %q, want last valid token", m.previewToken())
	}
	if m.alg != "HS256" {
		t.Fatalf("alg = %q, want HS256 from the valid header", m.alg)
	}
	if m.auditOK {
		t.Fatal("audit should be paused while JSON is invalid")
	}
	view := m.View()
	if !strings.Contains(view, "LAST VALID TOKEN") {
		t.Error("view should label the fallback token")
	}
	if strings.Contains(view, "NO ALG") {
		t.Error("valid header should not show NO ALG")
	}
	if !strings.Contains(view, "AUDIT PAUSED") {
		t.Error("view should show a paused audit")
	}
	if !strings.Contains(view, "JSON ERROR") {
		t.Error("view should label the JSON problem, not a bad signature")
	}
	if !strings.Contains(view, "PAYLOAD ✗") {
		t.Logf("view:\n%s", view)
		t.Error("payload tab should carry an error marker")
	}
}

func TestSignRoundTrip(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	setText(&m, sectionPayload, " "+m.payload.Value())
	if m.sigState != sigStale {
		t.Fatalf("setup: sigState = %v, want stale", m.sigState)
	}

	cmd := m.startSign()
	if cmd == nil || m.mode != modeSign {
		t.Fatalf("startSign did not open the dialog: mode=%v", m.mode)
	}
	m.secret.SetValue("your-256-bit-secret")
	m.submitSign()

	if m.mode != modeEdit {
		t.Fatalf("mode = %v, want edit after signing", m.mode)
	}
	if m.sigState != sigSigned {
		t.Fatalf("sigState = %v, want signed", m.sigState)
	}
	if m.stamp != stampFrames {
		t.Fatalf("stamp = %d, want %d after signing", m.stamp, stampFrames)
	}

	tok, err := jwt.Parse(m.currentToken)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	res, err := verify.Verify(context.Background(), tok, verify.Options{Secret: "your-256-bit-secret"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !res.SignatureValid {
		t.Fatal("signed token did not verify")
	}
}

func TestSignShortcutAndCancel(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "ctrl+s")
	if m.mode != modeSign {
		t.Fatalf("ctrl+s mode = %v, want sign", m.mode)
	}
	m = send(t, m, "esc")
	if m.mode != modeEdit {
		t.Fatalf("esc mode = %v, want edit", m.mode)
	}
	if m.signErr != nil {
		t.Fatalf("signErr = %v, want nil after cancel", m.signErr)
	}
}

func TestNoneAndUnsupportedSigning(t *testing.T) {
	none := jwt.Encode([]byte(`{"alg":"none"}`), []byte(`{}`), "abc")
	m := testModel(t, none, 120, 32)
	if m.sigState != sigUnsigned {
		t.Fatalf("none sigState = %v, want unsigned", m.sigState)
	}
	m.startSign()
	if m.signature.Value() != "" {
		t.Fatal("alg=none should clear the signature")
	}
	if !strings.Contains(m.toast, "cleared") {
		t.Fatalf("toast = %q, want cleared message", m.toast)
	}

	rs := jwt.Encode([]byte(`{"alg":"RS256"}`), []byte(`{}`), "")
	m = testModel(t, rs, 120, 32)
	m.startSign()
	if m.mode != modeEdit {
		t.Fatalf("RS256 should not open the sign dialog, mode=%v", m.mode)
	}
	if !strings.Contains(m.toast, "not in this build") {
		t.Fatalf("toast = %q, want unsupported message", m.toast)
	}
}

func TestSignWithSecretFileAndPEMGuard(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte("your-256-bit-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := testModel(t, testToken, 120, 32)
	m.startSign()
	m.secret.SetValue("@" + path)
	m.submitSign()
	if m.signErr != nil {
		t.Fatalf("sign with @file: %v", m.signErr)
	}
	if m.sigState != sigSigned {
		t.Fatalf("sigState = %v, want signed", m.sigState)
	}
	tok, err := jwt.Parse(m.currentToken)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	res, err := verify.Verify(context.Background(), tok, verify.Options{Secret: "your-256-bit-secret"})
	if err != nil || !res.SignatureValid {
		t.Fatalf("secret file token did not verify: %+v %v", res, err)
	}

	pem := testModel(t, testToken, 120, 32)
	pem.startSign()
	pem.secret.SetValue("-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----")
	pem.submitSign()
	if pem.signErr == nil {
		t.Fatal("PEM public key must be rejected as an HMAC secret")
	}
	if pem.mode != modeSign {
		t.Fatalf("mode = %v, want sign dialog to stay open on error", pem.mode)
	}
	if !strings.Contains(pem.signErr.Error(), "PEM") {
		t.Fatalf("error = %v, want PEM confusion message", pem.signErr)
	}
}

func TestCopyUsesCurrentTokenAndWarns(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	var copied string
	m.copyFn = func(s string) error {
		copied = s
		return nil
	}

	cmd := m.copyCurrent()
	if cmd == nil {
		t.Fatal("copyCurrent returned nil command")
	}
	msg := cmd()
	updated, _ := m.Update(msg)
	m = updated.(model)
	if copied != testToken {
		t.Fatalf("copied %q, want %q", copied, testToken)
	}
	if !strings.Contains(m.toast, "copied") {
		t.Fatalf("toast = %q, want copied", m.toast)
	}

	// A stale token must say so in the toast.
	setText(&m, sectionPayload, " "+m.payload.Value())
	cmd = m.copyCurrent()
	if cmd == nil {
		t.Fatal("stale copy returned nil command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(model)
	if !strings.Contains(m.toast, "signature stale") {
		t.Fatalf("toast = %q, want stale warning", m.toast)
	}
}

func TestHelpExplainsMotionsAndReservedKeys(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.openHelp()
	view := m.View()
	for _, want := range []string{
		"MOVE WITHOUT LEAVING INSERT MODE",
		"ctrl+h",
		"ctrl+l",
		"ctrl+i is Tab",
		"--no-vim-keys",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("help missing %q", want)
		}
	}

	m.vimKeys = false
	m.openHelp()
	if off := m.View(); !strings.Contains(off, "VIM MOTIONS ARE OFF") {
		t.Error("help should say when the motions are disabled")
	}
}

func TestCommandModeAndOverlays(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("mode = %v, want command", m.mode)
	}
	m = send(t, m, "a")
	if m.mode != modeAudit {
		t.Fatalf("mode = %v, want audit", m.mode)
	}
	if !strings.Contains(m.View(), "AUDIT") {
		t.Error("audit view missing title")
	}
	m = send(t, m, "esc")
	m = send(t, m, "?")
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	if !strings.Contains(m.View(), "EDITING") {
		t.Logf("help view:\n%s", m.View())
		t.Error("help view missing section")
	}
	m = send(t, m, "esc")
	m = send(t, m, "v")
	if m.mode != modeToken {
		t.Fatalf("mode = %v, want token", m.mode)
	}
	if !strings.Contains(m.View(), "ENCODED") && !strings.Contains(m.View(), testToken[:16]) {
		t.Error("token view missing token")
	}
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("mode = %v, want command after close", m.mode)
	}
}

func TestQuitKeys(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	cmd := m.handleKey(keyMsg("ctrl+c"))
	if cmd == nil {
		t.Fatal("ctrl+c returned nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+c did not return tea.Quit")
	}

	m = send(t, m, "esc")
	cmd = m.handleCommandKey(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q returned nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q did not return tea.Quit")
	}
}

func TestSignatureTabStates(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.section = sectionSignature
	m.focusSection()

	setText(&m, sectionSignature, "abc")
	if m.sigErr != nil {
		t.Fatalf("abc should be valid base64url: %v", m.sigErr)
	}
	if m.sigState != sigCustom {
		t.Fatalf("sigState = %v, want custom", m.sigState)
	}

	setText(&m, sectionSignature, "a.b")
	if m.sigErr == nil {
		t.Fatal("dotted signature should be rejected")
	}
	if m.sigState != sigInvalid {
		t.Fatalf("sigState = %v, want invalid", m.sigState)
	}
}

func TestFormatAndCompactPreserveKeyOrder(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	setText(&m, sectionHeader, `{"typ":"JWT","alg":"HS256"}`)

	if cmd := m.formatActive(); cmd == nil {
		t.Fatal("formatActive returned nil")
	}
	formatted := m.header.Value()
	if !strings.Contains(formatted, "\n") {
		t.Fatalf("format did not indent: %q", formatted)
	}
	if strings.Index(formatted, `"typ"`) > strings.Index(formatted, `"alg"`) {
		t.Fatalf("format reordered keys: %q", formatted)
	}
	if m.sigState != sigStale {
		t.Fatalf("format should stale the signature, got %v", m.sigState)
	}

	m.compactActive()
	if strings.Contains(m.header.Value(), "\n") {
		t.Fatalf("compact left newlines: %q", m.header.Value())
	}
	if strings.Index(m.header.Value(), `"typ"`) > strings.Index(m.header.Value(), `"alg"`) {
		t.Fatalf("compact reordered keys: %q", m.header.Value())
	}
}

func TestUntouchedPanesKeepOriginalSegments(t *testing.T) {
	rawHeader := "{\n  \"alg\": \"HS256\"\n}"
	rawPayload := "{\n  \"sub\": \"1\"\n}"
	raw := jwt.Encode([]byte(rawHeader), []byte(rawPayload), "abc")

	m := testModel(t, raw, 120, 32)
	if m.currentToken != raw {
		t.Fatalf("untouched token changed:\n got %q\nwant %q", m.currentToken, raw)
	}
	if !strings.Contains(m.header.Value(), "\n") {
		t.Fatal("editor should show pretty JSON")
	}

	// Edit the payload and prove the header segment is still the exact
	// original bytes, not a re-encoding of the pretty editor view.
	setText(&m, sectionPayload, " "+m.payload.Value())
	origHeaderSeg := strings.SplitN(raw, ".", 3)[0]
	newHeaderSeg := strings.SplitN(m.currentToken, ".", 3)[0]
	if newHeaderSeg != origHeaderSeg {
		t.Fatalf("header segment changed: got %q want %q", newHeaderSeg, origHeaderSeg)
	}
}

func TestCopyRefusesInvalidJSON(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	setText(&m, sectionPayload, "x")
	cmd := m.copyCurrent()
	if cmd == nil {
		t.Fatal("copyCurrent returned nil")
	}
	if !strings.Contains(m.toast, "fix JSON") {
		t.Fatalf("toast = %q, want fix JSON warning", m.toast)
	}
}

func TestSignRejectsEmptySecret(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.startSign()
	m.secret.SetValue("")
	m.submitSign()
	if m.signErr == nil {
		t.Fatal("empty secret should be rejected")
	}
	if m.mode != modeSign {
		t.Fatalf("mode = %v, want sign dialog to stay open", m.mode)
	}
}

func TestFormatRejectsInvalidJSON(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	setText(&m, sectionHeader, `{"alg":`)
	m.formatActive()
	if !strings.Contains(m.toast, "line") {
		t.Fatalf("toast = %q, want syntax error", m.toast)
	}
	if m.header.Value() != `{"alg":` {
		t.Fatal("format changed invalid input")
	}
}
