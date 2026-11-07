package tui

// layout.go owns terminal arithmetic. Keeping it in one place makes
// the 120-column and 80-column layouts use the same numbers, and makes
// resize bugs easy to spot instead of scattering magic subtractions.

const (
	minWidth      = 60
	minHeight     = 16
	wideLayoutMin = 100
	gutter        = 2
	bodyTop       = 3 // top bar, blank, tabs
	bodyBottom    = 2 // blank line, footer
)

type layout struct {
	width, height int
	tooSmall      bool
	wide          bool

	bodyHeight int
	leftWidth  int
	rightWidth int

	editorInnerW int
	editorInnerH int

	inspectorInnerW int
	inspectorInnerH int

	stackEditorH int
	stackTokenH  int
}

func (m model) layout() layout {
	l := layout{width: m.width, height: m.height}
	if l.width < minWidth || l.height < minHeight {
		l.tooSmall = true
		return l
	}
	l.wide = l.width >= wideLayoutMin
	l.bodyHeight = l.height - bodyTop - bodyBottom

	if l.wide {
		l.leftWidth = int(float64(l.width) * 0.54)
		if l.leftWidth < 42 {
			l.leftWidth = 42
		}
		if maxLeft := l.width - 38 - gutter; l.leftWidth > maxLeft {
			l.leftWidth = maxLeft
		}
		l.rightWidth = l.width - l.leftWidth - gutter
		l.editorInnerW = l.leftWidth - 4
		l.editorInnerH = l.bodyHeight - 2
		l.inspectorInnerW = l.rightWidth - 4
		l.inspectorInnerH = l.bodyHeight - 2
		return l
	}

	l.stackEditorH = int(float64(l.bodyHeight) * 0.62)
	if l.stackEditorH < 8 {
		l.stackEditorH = 8
	}
	if l.stackEditorH > l.bodyHeight-5 {
		l.stackEditorH = l.bodyHeight - 5
	}
	l.stackTokenH = l.bodyHeight - l.stackEditorH - 1
	if l.stackTokenH < 4 {
		l.stackTokenH = 4
		l.stackEditorH = l.bodyHeight - l.stackTokenH - 1
	}
	l.editorInnerW = l.width - 4
	l.editorInnerH = l.stackEditorH - 2
	l.inspectorInnerW = l.width - 4
	l.inspectorInnerH = l.stackTokenH - 2
	return l
}

// resize applies widget widths and heights for the current terminal.
// It does not change token state; the next View call uses the new
// layout, and the widgets scroll internally.
func (m *model) resize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	m.width, m.height = width, height
	l := m.layout()

	editorW := l.editorInnerW
	editorH := l.editorInnerH
	if l.tooSmall {
		editorW = 10
		editorH = 3
	} else if !l.wide {
		editorW = l.width - 4
		editorH = l.stackEditorH - 2
	}
	if editorW < 10 {
		editorW = 10
	}
	if editorH < 3 {
		editorH = 3
	}

	m.header.SetWidth(editorW)
	m.payload.SetWidth(editorW)
	m.header.SetHeight(editorH)
	m.payload.SetHeight(editorH)
	// The textinput cursor takes one cell beyond its value, so leave
	// three columns for the prompt and cursor inside the card.
	m.signature.Width = editorW - 3
	if m.signature.Width < 8 {
		m.signature.Width = 8
	}
	m.secret.Width = max(16, min(52, m.width-24))

	m.overlay.Width = max(20, m.width-10)
	m.overlay.Height = max(4, m.height-13)
}
