package tui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/yourpwnguy/jwt/v2/internal/jwt"
)

const testToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
	"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." +
	"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

var fixedNow = time.Unix(1_700_000_000, 0)

func testModel(t *testing.T, token string, width, height int) model {
	t.Helper()
	lipgloss.SetColorProfile(termenv.Ascii)
	lipgloss.SetHasDarkBackground(true)
	m, err := newModel(context.Background(), Options{
		Token: token,
		Now:   func() time.Time { return fixedNow },
		Copy:  func(string) error { return nil },
	})
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(model)
}

func TestViewSmoke(t *testing.T) {
	sizes := []struct {
		w, h int
	}{
		{120, 32},
		{100, 30},
		{80, 24},
		{60, 16},
		{50, 12},
	}
	for _, size := range sizes {
		m := testModel(t, testToken, size.w, size.h)
		view := m.View()
		if view == "" {
			t.Fatalf("%dx%d: empty view", size.w, size.h)
		}
		if got := lipgloss.Height(view); got != size.h && size.w >= minWidth && size.h >= minHeight {
			t.Fatalf("%dx%d: height = %d, want %d", size.w, size.h, got, size.h)
		}
		if size.w < minWidth || size.h < minHeight {
			if !strings.Contains(view, "too small") {
				t.Fatalf("%dx%d: missing too-small message", size.w, size.h)
			}
		}
	}
}

func TestSmallLayoutShowsToken(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {60, 16}} {
		m := testModel(t, testToken, size.w, size.h)
		if !strings.Contains(m.View(), testToken[:20]) {
			t.Errorf("%dx%d: token is not visible", size.w, size.h)
		}
	}
}

func TestEveryPageFitsTheWindow(t *testing.T) {
	sizes := []struct{ w, h int }{
		{120, 32}, {108, 28}, {100, 30}, {84, 16}, {80, 24}, {60, 16},
	}
	pages := []struct {
		name     string
		footer   string
		masthead bool
		open     func(*model)
	}{
		{"edit", "EDIT", true, func(m *model) { m.mode = modeEdit; m.focusSection() }},
		{"normal", "NORMAL", true, func(m *model) { m.enterCommandMode() }},
		{"sign", "SIGN", false, func(m *model) { m.startSign() }},
		{"help", "HELP", false, func(m *model) { m.openHelp() }},
		{"audit", "AUDIT", false, func(m *model) { m.openAudit() }},
		{"token", "TOKEN", false, func(m *model) { m.openToken() }},
	}
	for _, size := range sizes {
		for _, page := range pages {
			m := testModel(t, testToken, size.w, size.h)
			page.open(&m)
			view := m.View()
			if got := lipgloss.Height(view); got != size.h {
				t.Errorf("%s %dx%d: height = %d", page.name, size.w, size.h, got)
			}
			if !strings.Contains(view, page.footer) {
				t.Errorf("%s %dx%d: footer is clipped", page.name, size.w, size.h)
			}
			if page.masthead && !strings.Contains(view, "JWT") {
				t.Errorf("%s %dx%d: masthead is clipped", page.name, size.w, size.h)
			}
			for i, line := range strings.Split(view, "\n") {
				if got := lipgloss.Width(line); got != size.w {
					t.Fatalf("%s %dx%d line %d: width = %d", page.name, size.w, size.h, i+1, got)
				}
			}
		}
	}
}

func TestResizeWhileOverlaysOpen(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	m.startSign()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if view := updated.(model).View(); view == "" {
		t.Fatal("sign modal vanished after resize")
	}

	m = testModel(t, testToken, 120, 32)
	m = send(t, m, "esc")
	m.openAudit()
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 16})
	if view := updated.(model).View(); view == "" {
		t.Fatal("audit overlay vanished after resize")
	}
}

func TestLoadedTokenShowsHonestState(t *testing.T) {
	m := testModel(t, testToken, 120, 32)
	view := m.View()
	for _, want := range []string{"HEADER", "PAYLOAD", "SIGNATURE", "HS256", "ENCODED TOKEN", "UNTOUCHED"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
	if m.currentToken != testToken {
		t.Fatalf("untouched token changed:\n got %s\nwant %s", m.currentToken, testToken)
	}
	if m.sigState != sigUntouched {
		t.Fatalf("sigState = %v, want untouched", m.sigState)
	}
}

func TestBlankStart(t *testing.T) {
	m := testModel(t, "", 120, 32)
	if !strings.Contains(m.currentToken, ".") {
		t.Fatalf("blank token not assembled: %q", m.currentToken)
	}
	if m.sigState != sigUnsigned {
		t.Fatalf("blank sigState = %v, want unsigned", m.sigState)
	}
	if !strings.Contains(m.View(), "UNSIGNED") {
		t.Fatal("blank view should show UNSIGNED")
	}
}

func TestDeepJSONIsNotTruncatedOnLoad(t *testing.T) {
	// A compact segment can pretty-print to a much larger editor value.
	// If the textarea had a CharLimit at the segment cap, Bubbles would
	// silently cut the JSON while loading. This pins the no-truncation
	// rule with a 200-level object.
	const depth = 400
	header := `{"alg":"HS256","x":` + strings.Repeat(`{"x":`, depth) + `1` + strings.Repeat(`}`, depth) + `}`
	token := jwt.Encode([]byte(header), []byte(`{}`), "")

	m := testModel(t, token, 120, 32)
	value := m.header.Value()
	if len(value) <= jwt.MaxSegmentSize {
		t.Fatalf("pretty header is only %d bytes; test did not exceed the cap", len(value))
	}
	if !json.Valid([]byte(value)) {
		t.Fatal("loaded header was truncated into invalid JSON")
	}
	if view := m.View(); view == "" {
		t.Fatal("deep JSON view is empty")
	}
}
