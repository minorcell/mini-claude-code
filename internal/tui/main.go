package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// renderMain renders the main layout with conversation and trace panels
func (m model) renderMain() string {
	columnGap := m.theme.canvasFill().Render(" ")
	rowGap := m.theme.canvasFill().Width(m.layout.totalWidth).Render("")

	conversation := m.renderPanel(
		"Conversation",
		fmt.Sprintf("%d message(s) | scroll %d%%", len(m.entries), int(m.transcript.ScrollPercent()*100)),
		m.transcript.View(),
		m.layout.conversationWidth,
		m.layout.conversationHeight,
		m.activePane == paneConversation,
		m.theme.gold,
	)
	trace := m.renderPanel(
		"Context",
		fmt.Sprintf("step %02d/%02d", m.stepCount, m.maxSteps),
		m.activity.View(),
		m.layout.traceWidth,
		m.layout.traceHeight,
		false,
		m.theme.teal,
	)

	if m.layout.stacked {
		return lipgloss.JoinVertical(lipgloss.Left, conversation, rowGap, trace)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, conversation, columnGap, trace)
}

// renderFooter renders the footer with help text and error messages
func (m model) renderFooter() string {
	space := m.theme.canvasFill().Render(" ")
	helpLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.help.View(m.keys),
		space+space+space,
		m.theme.dim.Render(m.footerHint()),
	)
	body := helpLine
	if m.lastErr != nil {
		body = m.theme.error.Render("error: " + m.lastErr.Error())
	}

	return m.theme.canvasFill().
		Width(m.layout.totalWidth).
		Height(m.layout.footerHeight).
		Render(body)
}

// renderPanel renders a generic panel with title, hint, and body
func (m model) renderPanel(title string, hint string, body string, width int, height int, focused bool, accent string) string {
	innerWidth := max(1, width-m.theme.panelBase.GetHorizontalFrameSize())
	innerHeight := max(0, height-m.panelBorderHeight())
	space := m.theme.panelFill().Render(" ")
	titleBadge := m.theme.badge(title, accent, m.theme.ink)
	hintWidth := max(0, innerWidth-lipgloss.Width(titleBadge)-1)
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleBadge,
		space,
		m.theme.panelMeta.Width(hintWidth).Align(lipgloss.Right).Render(hint),
	)

	content := m.theme.panelFill().
		Width(innerWidth).
		Height(innerHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, body))
	return m.theme.panelFrame(focused, accent).
		Width(max(1, width-m.panelBorderWidth())).
		Height(innerHeight).
		Render(content)
}

// panelBorderWidth returns the horizontal border width of panels
func (m model) panelBorderWidth() int {
	return m.theme.panelBase.GetHorizontalBorderSize() + m.theme.panelBase.GetHorizontalMargins()
}

// panelBorderHeight returns the vertical border height of panels
func (m model) panelBorderHeight() int {
	return m.theme.panelBase.GetVerticalBorderSize() + m.theme.panelBase.GetVerticalMargins()
}
