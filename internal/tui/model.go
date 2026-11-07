package tui

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourpwnguy/jwt/v2/internal/audit"
	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// model.go defines the editor state. The state machine is deliberately
// small: one mode, one active section, four widgets, and the derived
// token state. Everything user-visible is recomputed in refresh().
type model struct {
	ctx    context.Context
	now    func() time.Time
	copyFn func(string) error
	theme  theme

	width, height int
	mode          mode
	section       section
	vimKeys       bool

	header    textarea.Model
	payload   textarea.Model
	signature textinput.Model
	secret    textinput.Model
	command   textinput.Model

	secretSpec   string
	revealSecret bool
	signErr      error
	signReturn   mode

	// pending is the first key of a two-key Neovim command. "f" waits
	// for a character to find, and "c" waits for a motion. lastFind is
	// the target of the most recent f, so ";" can repeat it.
	pending  string
	lastFind rune

	// Original token. Untouched panes keep these exact segments so
	// opening a token never silently changes it.
	loaded          bool
	originalHeader  string
	originalPayload string
	headerDirty     bool
	payloadDirty    bool

	// Signature origin. A stale status compares the current signing
	// input against this snapshot.
	originInput string
	originSig   string
	originKind  signatureOrigin

	// Derived token state.
	currentToken   string
	lastValidToken string
	signingInput   string
	headerErr      error
	payloadErr     error
	sigErr         error
	tokenErr       error
	parsed         jwt.Token
	parsedOK       bool
	alg            string
	sigState       signatureState
	report         audit.Report
	auditOK        bool
	expStatus      jwt.TemporalStatus
	expMessage     string

	startCmd tea.Cmd
	toast    string
	toastK   toastKind
	toastID  int

	overlay viewport.Model
}

type section int

const (
	sectionHeader section = iota
	sectionPayload
	sectionSignature
)

func (s section) String() string {
	switch s {
	case sectionHeader:
		return "HEADER"
	case sectionPayload:
		return "PAYLOAD"
	case sectionSignature:
		return "SIGNATURE"
	default:
		return "HEADER"
	}
}

type mode int

const (
	modeEdit mode = iota
	modeCommand
	modeEx
	modeSign
	modeAudit
	modeHelp
	modeToken
)

type signatureState int

const (
	sigUntouched signatureState = iota
	sigSigned
	sigStale
	sigCustom
	sigUnsigned
	sigInvalid
)

func (s signatureState) label() string {
	switch s {
	case sigSigned:
		return "SIGNED"
	case sigStale:
		return "STALE"
	case sigCustom:
		return "CUSTOM"
	case sigUnsigned:
		return "UNSIGNED"
	case sigInvalid:
		return "INVALID"
	default:
		return "UNTOUCHED"
	}
}

type signatureOrigin int

const (
	originUntouched signatureOrigin = iota
	originSigned
)

type toastKind int

const (
	toastInfo toastKind = iota
	toastSuccess
	toastWarning
	toastDanger
)

const (
	blankHeader  = "{\n  \"alg\": \"HS256\",\n  \"typ\": \"JWT\"\n}"
	blankPayload = "{}"
)

func newModel(ctx context.Context, opts Options) (model, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Copy == nil {
		opts.Copy = func(s string) error { return copyOSC52(os.Stdout, s) }
	}

	m := model{
		ctx:     ctx,
		now:     opts.Now,
		copyFn:  opts.Copy,
		theme:   newTheme(opts.Theme),
		mode:    modeCommand,
		section: sectionHeader,
		vimKeys: !opts.NoVimKeys,
		overlay: viewport.New(0, 0),
	}
	m.header = newTextarea(m.theme, sectionHeader)
	m.payload = newTextarea(m.theme, sectionPayload)
	m.signature = newSignatureInput(m.theme)
	m.secret = newSecretInput(m.theme)
	m.command = newCommandInput(m.theme)

	if strings.TrimSpace(opts.Token) == "" {
		m.loadBlank()
	} else if err := m.loadToken(opts.Token); err != nil {
		return model{}, err
	}

	m.resize(100, 30)
	m.refresh()
	// Normal mode is the default, exactly like Neovim. The first
	// keystroke is a motion or i/enter/a/o, not a character in the
	// buffer.
	m.startCmd = m.enterCommandMode()
	return m, nil
}

// focusSection blurs every widget and focuses the active one with a
// blinking insert-mode cursor. The returned command starts the blink.
func (m *model) focusSection() tea.Cmd {
	m.pending = ""
	m.blurAll()
	m.setActiveCursorMode(cursor.CursorBlink)
	return m.focusActive()
}

// enterCommandMode is the normal-mode transition. The cursor stays
// visible as a static block on the current character, exactly where i
// will put you back into editing. Blurring everything would make the
// cursor vanish and leave no position marker on the page.
func (m *model) enterCommandMode() tea.Cmd {
	m.mode = modeCommand
	m.pending = ""
	m.blurAll()
	m.setActiveCursorMode(cursor.CursorStatic)
	return m.focusActive()
}

// focusNormal is enterCommandMode without changing the current mode.
// Section switches use it so normal mode stays normal.
func (m *model) focusNormal() tea.Cmd {
	m.blurAll()
	m.setActiveCursorMode(cursor.CursorStatic)
	return m.focusActive()
}

// focusActive focuses the widget for the active section, without
// changing its cursor mode. Callers set blink or static first.
func (m *model) focusActive() tea.Cmd {
	switch m.section {
	case sectionHeader:
		return m.header.Focus()
	case sectionPayload:
		return m.payload.Focus()
	case sectionSignature:
		return m.signature.Focus()
	}
	return nil
}

// setActiveCursorMode changes only the active widget's cursor. The
// returned blink command is intentionally ignored: focusActive starts
// the blink loop for insert mode, and static mode schedules nothing.
func (m *model) setActiveCursorMode(mode cursor.Mode) {
	switch m.section {
	case sectionHeader:
		m.header.Cursor.SetMode(mode)
	case sectionPayload:
		m.payload.Cursor.SetMode(mode)
	case sectionSignature:
		m.signature.Cursor.SetMode(mode)
	}
}

func (m *model) blurAll() {
	m.header.Blur()
	m.payload.Blur()
	m.signature.Blur()
	m.secret.Blur()
	m.command.Blur()
}
