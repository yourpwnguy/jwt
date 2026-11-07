package tui

// view_chrome.go renders the top status bar, the section tabs, and the
// footer. These are the parts that stay put while the cards change.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderTopBar() string {
	left := m.theme.bold(m.theme.header).Render("JWT")
	if m.width >= 70 {
		dots := m.theme.fg(fillHeader).Render("●") +
			m.theme.fg(fillPayload).Render("●") +
			m.theme.fg(fillSignature).Render("●")
		left += " " + dots
	}
	if m.width >= 95 {
		left += "  " + m.theme.mutedStyle().Render("token studio")
	}

	alg := m.alg
	if alg == "" {
		alg = "NO ALG"
	}
	algFill := fillHeader
	if strings.EqualFold(alg, "none") || alg == "NO ALG" {
		algFill = fillDanger
	}
	right := m.theme.pill(strings.ToUpper(alg), algFill)
	if m.width >= 72 {
		right += "  " + m.theme.pill(m.sigLabel(), m.sigFill())
	} else {
		right += "  " + m.theme.fg(m.sigColor()).Render(strings.ToLower(m.sigLabel()))
	}
	if m.width >= 80 {
		right += "  " + m.theme.pill(m.auditBadge(), m.auditFill())
	} else {
		right += "  " + m.theme.fg(m.auditColor()).Render(m.auditCompact())
	}
	return joinLR(left, right, m.width)
}

func (m model) renderTabs() string {
	tabs := make([]string, 0, 3)
	for s := sectionHeader; s <= sectionSignature; s++ {
		label := s.String()
		if m.sectionError(s) != nil {
			label += " ✗"
		}
		if s == m.section {
			tabs = append(tabs, m.theme.pill("● "+label, m.theme.sectionFill(s)))
		} else {
			tabs = append(tabs, m.theme.mutedStyle().Render("○ "+label))
		}
	}
	left := strings.Join(tabs, "  ")
	right := m.theme.mutedStyle().Render("exact bytes · no re-marshal")
	if err := m.activeError(); err != nil {
		right = m.theme.fg(m.theme.danger).Render(err.Error())
	}
	if m.width < 100 {
		right = ""
	}
	return joinLRKeepLeft(left, right, m.width)
}

func (m model) activeError() error {
	if err := m.sectionError(m.section); err != nil {
		return err
	}
	return m.tokenErr
}

func (m model) sectionError(s section) error {
	switch s {
	case sectionHeader:
		return m.headerErr
	case sectionPayload:
		return m.payloadErr
	case sectionSignature:
		return m.sigErr
	}
	return nil
}

// sigLabel avoids calling an incomplete token "invalid signature" when
// the real problem is JSON. The tab mark points at the broken pane.
func (m model) sigLabel() string {
	if m.headerErr != nil || m.payloadErr != nil {
		return "JSON ERROR"
	}
	return m.sigState.label()
}

func (m model) renderFooter() string {
	label := "EDIT"
	var fill lipgloss.TerminalColor = fillSuccess
	hints := "ctrl+s sign · ctrl+y copy · ctrl+o commands · tab section"
	if m.width < 80 {
		hints = "ctrl+s sign · ctrl+y copy · ctrl+o cmds"
	}
	switch m.mode {
	case modeCommand:
		label, fill = "COMMAND", fillWarning
		hints = "i edit · s sign · y copy · f format · a audit · v token · ? help · q quit"
		if m.width < 90 {
			hints = "i edit · s sign · y copy · a audit · q quit"
		}
	case modeSign:
		label, fill = "SIGN", fillHeader
		hints = "enter sign · ctrl+r reveal · esc cancel"
	case modeAudit:
		label, fill = "AUDIT", m.auditFill()
		hints = "↑↓ scroll · y copy · esc close"
	case modeHelp:
		label, fill = "HELP", fillSignature
		hints = "↑↓ scroll · esc close"
	case modeToken:
		label, fill = "TOKEN", fillSignature
		hints = "↑↓ scroll · y copy · esc close"
	}
	left := m.theme.pill(label, fill) + "   " + m.theme.mutedStyle().Render(hints)
	right := ""
	if m.toast != "" {
		right = m.theme.fg(m.toastColor()).Render(m.toast)
	}
	return joinLR(left, right, m.width)
}

func (m model) toastColor() lipgloss.TerminalColor {
	switch m.toastK {
	case toastSuccess:
		return m.theme.success
	case toastWarning:
		return m.theme.warning
	case toastDanger:
		return m.theme.danger
	default:
		return m.theme.signature
	}
}

func (m model) sigFill() lipgloss.TerminalColor {
	switch m.sigState {
	case sigSigned:
		return fillSuccess
	case sigStale, sigInvalid:
		return fillDanger
	case sigUnsigned, sigCustom:
		return fillWarning
	default:
		return fillSignature
	}
}

func (m model) auditFill() lipgloss.TerminalColor {
	if !m.auditOK {
		return fillWarning
	}
	if m.report.FailedCount() > 0 {
		return fillDanger
	}
	return fillSuccess
}

func (m model) auditBadge() string {
	if !m.auditOK {
		return "AUDIT PAUSED"
	}
	failed := m.report.FailedCount()
	if failed == 0 {
		return fmt.Sprintf("✓ %d/%d", len(m.report.Findings), len(m.report.Findings))
	}
	return fmt.Sprintf("✗ %d ISSUE%s", failed, strings.ToUpper(plural(failed)))
}

// auditCompact is the narrow top-bar form. It keeps the count visible
// when there is no room for a filled badge.
func (m model) auditCompact() string {
	if !m.auditOK {
		return "audit paused"
	}
	failed := m.report.FailedCount()
	if failed == 0 {
		return fmt.Sprintf("%d/%d", len(m.report.Findings), len(m.report.Findings))
	}
	return fmt.Sprintf("✗%d", failed)
}
