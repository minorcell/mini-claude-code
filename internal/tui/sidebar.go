package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// metricRow represents a row in the sidebar metrics display
type metricRow struct {
	Label string
	Value string
}

// renderOverviewBody renders the overview panel with usage statistics and status
func (m model) renderOverviewBody(width int) string {
	turnUsageTotal := m.currentTurnUsage.InputTokens + m.currentTurnUsage.OutputTokens
	sessionUsageTotal := m.sessionUsage.InputTokens + m.sessionUsage.OutputTokens
	elapsed := m.lastDuration
	if m.busy {
		elapsed = timeSince(m.turnStarted)
	}

	contextLines := []string{
		m.theme.badge("CONTEXT", m.theme.sand, m.theme.ink),
		m.theme.entryText.Bold(true).Width(width).Render(fmt.Sprintf("%s tokens", formatCount(sessionUsageTotal))),
		m.theme.traceText.Width(width).Render(previewText(m.status, max(20, width))),
		m.renderSidebarRows(width, []metricRow{
			{Label: "turn", Value: formatCount(turnUsageTotal)},
			{Label: "input", Value: formatCount(m.currentTurnUsage.InputTokens)},
			{Label: "output", Value: formatCount(m.currentTurnUsage.OutputTokens)},
			{Label: "step", Value: fmt.Sprintf("%02d / %02d", m.stepCount, m.maxSteps)},
			{Label: "tools", Value: fmt.Sprintf("%d", m.currentToolCalls)},
			{Label: "elapsed", Value: formatDuration(elapsed)},
		}),
	}
	if m.lastAction != "" {
		contextLines = append(contextLines, m.theme.dim.Width(width).Render(previewText(m.lastAction, 96)))
	}

	blocks := []string{
		lipgloss.JoinVertical(lipgloss.Left, contextLines...),
	}
	if len(m.todoItems) > 0 {
		blocks = append(blocks, "", m.renderTodoSection(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

// renderSidebarRows renders multiple metric rows in the sidebar
func (m model) renderSidebarRows(width int, rows []metricRow) string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		labelWidth := clamp(width/2, 8, 12)
		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.traceDetail.Width(labelWidth).Render(strings.ToUpper(row.Label)),
			m.theme.traceText.Width(max(1, width-labelWidth)).Align(lipgloss.Right).Render(row.Value),
		)
		lines = append(lines, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderTodoSection renders the todo items section
func (m model) renderTodoSection(width int) string {
	lines := []string{
		m.theme.badge("TODO", m.theme.teal, m.theme.ink),
	}

	for _, item := range m.todoItems {
		lines = append(lines, lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.traceText.Render(todoStatusIcon(item.Status)+" "),
			m.theme.entryText.Width(max(8, width-4)).Render(previewText(item.Content, max(12, width-4))),
		))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// todoStatusIcon returns the appropriate icon for todo status
func todoStatusIcon(status string) string {
	switch status {
	case "completed":
		return "[✓]"
	case "in_progress":
		return "[•]"
	default:
		return "[ ]"
	}
}
