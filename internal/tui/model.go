package tui

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/yourpwnguy/jwt/v2/internal/audit"
	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

// model.go defines the editor state. The state machine is deliberately
// small: one mode, one active section, three widgets, and the derived
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

	secretSpec   string
	revealSecret bool
	signErr      error
	signReturn   mode

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

	// stamp is the short letterpress animation after a successful
	// signature. Zero means idle.
	stamp   int
	stampID int

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
		mode:    modeEdit,
		section: sectionHeader,
		vimKeys: !opts.NoVimKeys,
		overlay: viewport.New(0, 0),
	}
	m.header = newTextarea(m.theme, sectionHeader)
	m.payload = newTextarea(m.theme, sectionPayload)
	m.signature = newSignatureInput(m.theme)
	m.secret = newSecretInput(m.theme)

	if strings.TrimSpace(opts.Token) == "" {
		m.loadBlank()
	} else if err := m.loadToken(opts.Token); err != nil {
		return model{}, err
	}

	m.resize(100, 30)
	m.refresh()
	m.startCmd = m.header.Focus()
	return m, nil
}

// focusSection blurs every widget and focuses the active one. The
// returned command starts the cursor blink.
func (m *model) focusSection() tea.Cmd {
	m.header.Blur()
	m.payload.Blur()
	m.signature.Blur()
	m.secret.Blur()

	var cmd tea.Cmd
	switch m.section {
	case sectionHeader:
		cmd = m.header.Focus()
	case sectionPayload:
		cmd = m.payload.Focus()
	case sectionSignature:
		cmd = m.signature.Focus()
	}
	return cmd
}

func (m *model) blurAll() {
	m.header.Blur()
	m.payload.Blur()
	m.signature.Blur()
	m.secret.Blur()
}
