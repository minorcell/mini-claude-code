package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m model) renderTranscriptEntry(entry transcriptEntry, width int) string {
	accent := m.theme.muted
	switch entry.Role {
	case "user":
		accent = m.theme.teal
	case "assistant":
		accent = m.theme.gold
	case "tool":
		accent = m.theme.sage
	case "system":
		accent = m.theme.coral
	}

	metaWidth := max(0, width-lipgloss.Width(entry.Title)-4)
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.theme.badge(entry.Title, accent, m.theme.ink),
		m.theme.panelFill().Render(" "),
		m.theme.entryMeta.Width(metaWidth).Align(lipgloss.Right).Render(entry.Meta),
	)

	body := m.theme.entryText.Width(width).Render(entry.Body)
	card := m.theme.panelFill().
		MarginBottom(1)
	if entry.Role == "system" {
		body = m.theme.dim.Width(width).Render(entry.Body)
	}
	if entry.Role == "tool" {
		body = m.theme.traceResult.Width(width).Render(entry.Body)
	}
	if entry.Role == "user" || entry.Role == "assistant" {
		body = m.theme.entryText.Width(width).Render(entry.Body)
	}

	return card.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, header, body))
}
