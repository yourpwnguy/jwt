package tui

// layout.go owns terminal arithmetic. The page has a fixed masthead,
// a section index, the body, and a footer. Wide terminals get a
// vertical index rail, medium ones get a horizontal tab row, and
// narrow ones stack the editor over a short token strip.
//
// Keeping every subtraction here makes the 120, 100, 80, and 60 column
// layouts use the same numbers, so a resize bug is a math bug in one
// file instead of a surprise in five renderers.

const (
	minWidth  = 60
	minHeight = 16

	wideLayoutMin   = 108
	wideHeightMin   = 28
	mediumLayoutMin = 84

	gutter = 2
)

type layoutMode int

const (
	layoutWide layoutMode = iota
	layoutMedium
	layoutNarrow
)

type layout struct {
	width, height int
	tooSmall      bool
	mode          layoutMode
	compact       bool

	margin   int
	contentW int
	bodyH    int
	overlayH int

	railW int

	editorW     int
	editorTextW int
	panelW      int
	panelTextW  int

	stackEditorH int
	stackStripH  int

	topPad    int
	bottomPad int
	gap       int
	showTabs  bool
}

func (m model) layout() layout {
	l := layout{width: m.width, height: m.height}
	if l.width < minWidth || l.height < minHeight {
		l.tooSmall = true
		return l
	}

	l.compact = l.height < 22
	if l.compact {
		// At the minimum height every row matters. The body still has
		// its own gap between editor and strip, but the page chrome
		// drops the outer air so the footer is never the thing that
		// falls off the bottom.
		l.topPad, l.bottomPad, l.gap = 0, 0, 0
	} else {
		l.topPad, l.bottomPad, l.gap = 1, 1, 1
	}

	if l.width >= wideLayoutMin && l.height >= wideHeightMin {
		l.mode = layoutWide
	} else if l.width >= mediumLayoutMin {
		l.mode = layoutMedium
	} else {
		l.mode = layoutNarrow
	}

	if l.width >= mediumLayoutMin {
		l.margin = 2
	} else {
		l.margin = 1
	}
	l.contentW = l.width - 2*l.margin

	switch l.mode {
	case layoutWide:
		// Masthead, rule, body, rule, footer, plus the outer air.
		l.bodyH = l.height - 8
		if l.compact {
			l.bodyH = l.height - 6
		}
		l.railW = 15
		l.panelW = clamp(l.contentW*32/100, 30, 38)
		l.editorW = l.contentW - l.railW - 2*gutter - l.panelW
	case layoutMedium:
		// Masthead, rule, tabs, tab rule, body, rule, footer, air.
		l.bodyH = l.height - 10
		if l.compact {
			l.bodyH = l.height - 6
		}
		l.showTabs = true
		l.panelW = clamp(l.contentW*38/100, 28, 38)
		l.editorW = l.contentW - gutter - l.panelW
	case layoutNarrow:
		l.bodyH = l.height - 10
		if l.compact {
			l.bodyH = l.height - 6
		}
		l.showTabs = true
		l.editorW = l.contentW
		l.panelW = l.contentW

		l.stackEditorH = l.bodyH * 60 / 100
		if l.stackEditorH < 6 {
			l.stackEditorH = 6
		}
		l.stackStripH = l.bodyH - l.stackEditorH - 1
		if l.stackStripH < 3 {
			l.stackStripH = 3
			l.stackEditorH = l.bodyH - l.stackStripH - 1
		}
	}

	if l.editorW < 20 {
		l.editorW = 20
	}
	l.editorTextW = l.editorW - 2
	if l.editorTextW < 16 {
		l.editorTextW = 16
	}
	l.panelTextW = l.panelW - 5
	if l.panelTextW < 12 {
		l.panelTextW = 12
	}

	l.overlayH = l.height - (6 + l.topPad + l.bottomPad)
	return l
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
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

	editorW := l.editorTextW
	editorH := l.bodyH
	if l.mode == layoutNarrow && !l.tooSmall {
		editorH = l.stackEditorH
	}
	if l.tooSmall {
		editorW, editorH = 30, 6
	}
	if editorW < 16 {
		editorW = 16
	}
	if editorH < 3 {
		editorH = 3
	}

	m.header.SetWidth(editorW)
	m.payload.SetWidth(editorW)
	m.header.SetHeight(editorH)
	m.payload.SetHeight(editorH)

	// The text input draws a prompt and a cursor beyond the value.
	m.signature.Width = max(8, editorW-4)
	m.secret.Width = max(16, min(52, l.contentW-18))

	m.overlay.Width = max(20, l.contentW)
	m.overlay.Height = max(4, l.overlayH)
}
