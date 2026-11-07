package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
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

// sendText types one rune at a time. A real terminal never delivers
// ":audit" as a single key message, and the mode changes after ":".
func sendText(t *testing.T, m model, text string) model {
	t.Helper()
	for _, r := range text {
		m = send(t, m, string(r))
	}
	return m
}

// beginEdit is the explicit transition tests use when they are about
// insert-mode behavior. The real editor now starts in normal mode.
func beginEdit(m model) model {
	m.mode = modeEdit
	m.focusSection()
	return m
}

func setText(m *model, s section, text string) {
	m.section = s
	m.setSectionText(text)
	m.refresh()
}

func TestStartsInNormalMode(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	if m.mode != modeCommand {
		t.Fatalf("initial mode = %v, want normal", m.mode)
	}
	if !strings.Contains(m.View(), "NORMAL") {
		t.Fatal("initial footer should say NORMAL")
	}
	if !m.header.Focused() {
		t.Fatal("initial cursor should be focused")
	}
	if got := m.header.Cursor.Mode(); got != cursor.CursorStatic {
		t.Fatalf("initial cursor mode = %v, want static", got)
	}
}

func TestCommandModeKeepsACursor(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("mode = %v, want command", m.mode)
	}
	if !strings.Contains(m.View(), "NORMAL") {
		t.Fatal("normal mode should be labelled NORMAL")
	}
	if !m.header.Focused() {
		t.Fatal("header should stay focused in command mode")
	}
	if got := m.header.Cursor.Mode(); got != cursor.CursorStatic {
		t.Fatalf("command cursor mode = %v, want static", got)
	}

	// Esc in normal mode must not fall back into insert mode. Only i,
	// enter, or ctrl+o should do that.
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("esc left normal mode: %v", m.mode)
	}

	m = send(t, m, "i")
	if m.mode != modeEdit {
		t.Fatalf("mode = %v, want edit", m.mode)
	}
	if got := m.header.Cursor.Mode(); got != cursor.CursorBlink {
		t.Fatalf("edit cursor mode = %v, want blink", got)
	}
}

func TestNormalModeHJKLMoveTheCursor(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	// Start on the first JSON key line, which has room to move right.
	m.setActiveCursor(1, 0)
	m = send(t, m, "esc")

	m = send(t, m, "l")
	if got := m.header.LineInfo().ColumnOffset; got != 1 {
		t.Fatalf("l column = %d, want 1", got)
	}
	m = send(t, m, "h")
	if got := m.header.LineInfo().ColumnOffset; got != 0 {
		t.Fatalf("h column = %d, want 0", got)
	}
	m = send(t, m, "j")
	if got := m.header.Line(); got != 2 {
		t.Fatalf("j line = %d, want 2", got)
	}
	m = send(t, m, "k")
	if got := m.header.Line(); got != 1 {
		t.Fatalf("k line = %d, want 1", got)
	}

	// i must resume editing at the position the normal-mode cursor
	// reached, not at the end of the buffer.
	m = send(t, m, "l")
	col := m.header.LineInfo().ColumnOffset
	m = send(t, m, "i")
	if m.mode != modeEdit {
		t.Fatalf("mode = %v, want edit", m.mode)
	}
	if got := m.header.LineInfo().ColumnOffset; got != col {
		t.Fatalf("insert column = %d, want %d", got, col)
	}
}

func TestSignatureAndSignCancelRestoreTheCommandCursor(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.section = sectionSignature
	m.enterCommandMode()
	if !m.signature.Focused() || m.signature.Cursor.Mode() != cursor.CursorStatic {
		t.Fatal("signature cursor should be focused and static in command mode")
	}

	// A sign dialog opened from normal mode must return to a visible
	// static cursor when canceled, not to a blurred editor.
	m = send(t, m, "ctrl+s")
	if m.mode != modeSign {
		t.Fatalf("mode = %v, want sign", m.mode)
	}
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("mode = %v, want command after cancel", m.mode)
	}
	if !m.signature.Focused() || m.signature.Cursor.Mode() != cursor.CursorStatic {
		t.Fatal("cancel should restore the visible command cursor")
	}
}

func TestOverlayCloseRestoresTheCommandCursor(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "esc")
	m = send(t, m, "?")
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("mode = %v, want command", m.mode)
	}
	if !m.header.Focused() || m.header.Cursor.Mode() != cursor.CursorStatic {
		t.Fatal("closing help should restore the visible command cursor")
	}
}

func TestVimMotionsMoveTheCursor(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = beginEdit(m)
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
	m = beginEdit(m)
	m = send(t, m, "tab") // KeyTab is the same byte as ctrl+i.
	if m.section != sectionPayload {
		t.Fatalf("ctrl+i changed section to %v, want payload", m.section)
	}
}

func TestVimMotionsReachSignatureField(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.section = sectionSignature
	m = beginEdit(m)
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
	m = beginEdit(m)
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
	m = beginEdit(m)
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
	m = beginEdit(m)
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
	m = beginEdit(m)
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
	m = beginEdit(m)
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
	m = beginEdit(m)
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
	m = beginEdit(m)
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
	content := strings.Join(m.helpContent(120), "\n")
	for _, want := range []string{
		"MOVE",
		"h j k l",
		"w / b",
		"f{char} / ;",
		"gg / G",
		"c{motion}",
		"d{motion}",
		":sign",
		"ctrl+hjkl",
		"ctrl+i is Tab",
		"--no-vim-keys",
		"starts in normal mode",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("help missing %q", want)
		}
	}

	m.vimKeys = false
	content = strings.Join(m.helpContent(120), "\n")
	if !strings.Contains(content, "INSERT CTRL MOTIONS ARE OFF") {
		t.Error("help should say when the insert motions are disabled")
	}
}

func TestCommandModeAndOverlays(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("mode = %v, want command", m.mode)
	}
	m = sendText(t, m, ":audit")
	m = send(t, m, "enter")
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
	if !strings.Contains(m.View(), "MOVE") {
		t.Logf("help view:\n%s", m.View())
		t.Error("help view missing section")
	}
	m = send(t, m, "esc")
	m = sendText(t, m, ":token")
	m = send(t, m, "enter")
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

func TestNormalModeWordMotions(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText("{\n  \"alg\": \"HS256\"\n}")
	m.setActiveCursor(1, 0)
	m = send(t, m, "esc")

	m = send(t, m, "w")
	if got := m.header.LineInfo().ColumnOffset; got != 2 {
		t.Fatalf("w to quote column = %d, want 2", got)
	}
	m = send(t, m, "w")
	if got := m.header.LineInfo().ColumnOffset; got != 3 {
		t.Fatalf("w to alg column = %d, want 3", got)
	}
	m = send(t, m, "b")
	if got := m.header.LineInfo().ColumnOffset; got != 2 {
		t.Fatalf("b to quote column = %d, want 2", got)
	}

	// b also crosses the newline back to the opening brace.
	m = send(t, m, "b")
	if line, col := m.header.Line(), m.header.LineInfo().ColumnOffset; line != 0 || col != 0 {
		t.Fatalf("b across line = %d:%d, want 0:0", line, col)
	}
}

func TestNormalModeLineMotions(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText(`{"a":1}`)
	m.setActiveCursor(0, 0)
	m = send(t, m, "esc")

	m = send(t, m, "$")
	if got := m.header.LineInfo().ColumnOffset; got != 6 {
		t.Fatalf("$ column = %d, want 6", got)
	}
	m = send(t, m, "l")
	if got := m.header.LineInfo().ColumnOffset; got != 6 {
		t.Fatalf("l past end column = %d, want 6", got)
	}
	m = send(t, m, "0")
	if got := m.header.LineInfo().ColumnOffset; got != 0 {
		t.Fatalf("0 column = %d, want 0", got)
	}
}

func TestNormalModeFindAndRepeat(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText(`  "alg": "HS256"`)
	m.setActiveCursor(0, 0)
	m = send(t, m, "esc")

	m = send(t, m, "f")
	if m.pending != "f" {
		t.Fatalf("pending = %q, want f", m.pending)
	}
	m = send(t, m, `"`)
	if got := m.header.LineInfo().ColumnOffset; got != 2 {
		t.Fatalf("f\" column = %d, want 2", got)
	}
	m = send(t, m, ";")
	if got := m.header.LineInfo().ColumnOffset; got != 6 {
		t.Fatalf("; column = %d, want 6", got)
	}
	m = send(t, m, ";")
	if got := m.header.LineInfo().ColumnOffset; got != 9 {
		t.Fatalf("second ; column = %d, want 9", got)
	}
}

func TestNormalModeOpenAndChangeLines(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText("{\n  \"a\": 1\n}")
	m.setActiveCursor(1, 0)
	m = send(t, m, "esc")

	m = send(t, m, "o")
	if m.mode != modeEdit {
		t.Fatalf("o mode = %v, want edit", m.mode)
	}
	if got := strings.Split(m.header.Value(), "\n"); len(got) != 4 || got[2] != "" {
		t.Fatalf("o lines = %#v, want an empty line below", got)
	}
	if got := m.header.Line(); got != 2 {
		t.Fatalf("o cursor line = %d, want 2", got)
	}

	// O opens above and keeps the original line intact below it.
	m = send(t, m, "esc")
	m.setActiveCursor(0, 0)
	m = send(t, m, "O")
	lines := strings.Split(m.header.Value(), "\n")
	if len(lines) != 5 || lines[0] != "" || lines[1] != "{" {
		t.Fatalf("O lines = %#v, want an empty line above the brace", lines)
	}
}

func TestNormalModeChangeMotion(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText(`  "alg": "HS256"`)
	m.setActiveCursor(0, 3)
	m = send(t, m, "esc")

	m = send(t, m, "c")
	if m.pending != "c" {
		t.Fatalf("pending = %q, want c", m.pending)
	}
	m = send(t, m, "w")
	if m.mode != modeEdit {
		t.Fatalf("cw mode = %v, want edit", m.mode)
	}
	if got := m.header.Value(); got != `  "": "HS256"` {
		t.Fatalf("cw value = %q", got)
	}
	if got := m.header.LineInfo().ColumnOffset; got != 3 {
		t.Fatalf("cw column = %d, want 3", got)
	}

	m = send(t, m, "esc")
	m = send(t, m, "c")
	m = send(t, m, "c")
	if got := m.header.Value(); got != "" {
		t.Fatalf("cc value = %q, want empty line", got)
	}
}

func TestNormalModeAppend(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText(`{}`)
	m.setActiveCursor(0, 0)
	m = send(t, m, "esc")

	m = send(t, m, "a")
	if m.mode != modeEdit {
		t.Fatalf("a mode = %v, want edit", m.mode)
	}
	if got := m.header.LineInfo().ColumnOffset; got != 1 {
		t.Fatalf("a column = %d, want 1", got)
	}
}

func TestNormalModeSectionSwitchStaysNormal(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "esc")
	m = send(t, m, "2")
	if m.mode != modeCommand {
		t.Fatalf("2 mode = %v, want normal", m.mode)
	}
	if m.section != sectionPayload {
		t.Fatalf("2 section = %v, want payload", m.section)
	}
	if !m.payload.Focused() || m.payload.Cursor.Mode() != cursor.CursorStatic {
		t.Fatal("normal mode section switch lost the static cursor")
	}
	m = send(t, m, "tab")
	if m.section != sectionSignature || m.mode != modeCommand {
		t.Fatalf("tab = section %v mode %v, want signature normal", m.section, m.mode)
	}
}

func TestNormalModeGGAndG(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText("{\n  \"a\": 1,\n  \"b\": 2\n}")
	m.setActiveCursor(1, 0)
	m = send(t, m, "esc")

	m = send(t, m, "g")
	if m.pending != "g" {
		t.Fatalf("pending = %q, want g", m.pending)
	}
	m = send(t, m, "g")
	if got := m.header.Line(); got != 0 {
		t.Fatalf("gg line = %d, want 0", got)
	}
	m = send(t, m, "G")
	if got := m.header.Line(); got != 3 {
		t.Fatalf("G line = %d, want 3", got)
	}
}

func TestNormalModeDD(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText("{\n  \"a\": 1,\n  \"b\": 2\n}")
	m.setActiveCursor(1, 0)
	m = send(t, m, "esc")

	m = send(t, m, "d")
	if m.pending != "d" {
		t.Fatalf("pending = %q, want d", m.pending)
	}
	m = send(t, m, "d")
	if got := m.header.Value(); got != "{\n  \"b\": 2\n}" {
		t.Fatalf("dd value = %q", got)
	}
	if got := m.header.Line(); got != 1 {
		t.Fatalf("dd line = %d, want 1", got)
	}
}

func TestNormalModeDiwAndDaw(t *testing.T) {
	inner := testModel(t, "", 120, 32)
	inner.section = sectionHeader
	inner.setSectionText(`{"alg":"HS256"}`)
	inner.setActiveCursor(0, 2)
	inner = send(t, inner, "esc")
	inner = send(t, inner, "d")
	inner = send(t, inner, "i")
	if inner.pending != "di" {
		t.Fatalf("pending = %q, want di", inner.pending)
	}
	inner = send(t, inner, "w")
	if got := inner.header.Value(); got != `{"":"HS256"}` {
		t.Fatalf("diw value = %q", got)
	}

	around := testModel(t, "", 120, 32)
	around.section = sectionHeader
	around.setSectionText(`{"name": "John Doe"}`)
	around.setActiveCursor(0, 10)
	around = send(t, around, "esc")
	around = send(t, around, "d")
	around = send(t, around, "a")
	around = send(t, around, "w")
	if got := around.header.Value(); got != `{"name": "Doe"}` {
		t.Fatalf("daw value = %q", got)
	}
}

func TestAutoPairQuotes(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText(`{}`)
	m.setActiveCursor(0, 1)
	m = beginEdit(m)

	m = send(t, m, `"`)
	if got := m.header.Value(); got != `{""}` {
		t.Fatalf("pair value = %q, want { \"\" }", got)
	}
	if got := m.header.LineInfo().ColumnOffset; got != 2 {
		t.Fatalf("pair cursor = %d, want 2", got)
	}

	m = send(t, m, "a")
	m = send(t, m, `"`)
	if got := m.header.Value(); got != `{"a"}` {
		t.Fatalf("closing value = %q, want {\"a\"}", got)
	}
	if got := m.header.LineInfo().ColumnOffset; got != 4 {
		t.Fatalf("closing cursor = %d, want 4", got)
	}
}

func TestAutoPairBrackets(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionPayload
	m.setSectionText("")
	m.setActiveCursor(0, 0)
	m = beginEdit(m)

	m = send(t, m, "{")
	if got := m.payload.Value(); got != `{}` {
		t.Fatalf("brace value = %q, want {}", got)
	}
	if got := m.payload.LineInfo().ColumnOffset; got != 1 {
		t.Fatalf("brace cursor = %d, want 1", got)
	}
}

func TestAutoPairLeavesEscapedQuoteAlone(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText("\"a\\")
	m.setActiveCursor(0, 3)
	m = beginEdit(m)

	m = send(t, m, `"`)
	if got := m.header.Value(); got != "\"a\\\"" {
		t.Fatalf("escaped value = %q, want a single quote", got)
	}
}

func TestAutoPairInsideStringDoesNotCloseBrackets(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText(`{"a": "x`)
	m.setActiveCursor(0, 8)
	m = beginEdit(m)

	m = send(t, m, "[")
	if got := m.header.Value(); got != `{"a": "x[` {
		t.Fatalf("inside-string value = %q, want a single bracket", got)
	}
}

func TestPropertyDocsFollowTheCursor(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.setActiveCursor(1, 0) // "alg" in the header
	view := m.View()
	if !strings.Contains(view, "PROPERTY") || !strings.Contains(view, "Signing algorithm") {
		t.Fatalf("header docs missing alg:\n%s", view)
	}

	m.section = sectionPayload
	m.setActiveCursor(1, 0) // "sub"
	view = m.View()
	if !strings.Contains(view, "Subject") {
		t.Fatalf("payload docs missing sub:\n%s", view)
	}

	m.section = sectionPayload
	m.setSectionText(`{"custom": 1}`)
	m.setActiveCursor(0, 0)
	view = m.View()
	if !strings.Contains(view, "Custom claim") {
		t.Fatalf("custom claim docs missing:\n%s", view)
	}
}

func TestCommandLineActions(t *testing.T) {
	m := testModel(t, "", 120, 32)
	m.section = sectionHeader
	m.setSectionText(`{"alg":"HS256"}`)
	m.header.CursorStart()
	m = send(t, m, "esc")

	m = sendText(t, m, ":format")
	m = send(t, m, "enter")
	if m.mode != modeCommand {
		t.Fatalf(":format mode = %v, want normal", m.mode)
	}
	if !strings.Contains(m.header.Value(), "\n") {
		t.Fatalf(":format did not indent: %q", m.header.Value())
	}

	m = sendText(t, m, ":sign")
	m = send(t, m, "enter")
	if m.mode != modeSign {
		t.Fatalf(":sign mode = %v, want sign", m.mode)
	}
	m = send(t, m, "esc")
	if m.mode != modeCommand {
		t.Fatalf("sign cancel mode = %v, want normal", m.mode)
	}

	m = sendText(t, m, ":nope")
	m = send(t, m, "enter")
	if !strings.Contains(m.toast, "unknown command") {
		t.Fatalf("toast = %q, want unknown command", m.toast)
	}
}

func TestCommandLineQuits(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m = send(t, m, "esc")
	m = sendText(t, m, ":q")
	cmd := m.runEx()
	if cmd == nil {
		t.Fatal(":q returned nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal(":q did not quit")
	}
}
