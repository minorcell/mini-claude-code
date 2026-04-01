package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderComposer renders the composer panel with input and file candidates
func (m model) renderComposer() string {
	hint := "Enter sends | Ctrl+J newline"
	if m.busy && m.queuedPrompt != "" {
		hint = "Esc recalls queued draft | Esc Esc interrupts"
	} else if m.filePicker.Active {
		hint = "Ctrl+J newline | up/down select | tab/enter accept"
	} else if m.busy {
		hint = "Enter queues | Ctrl+J newline | Esc Esc interrupts"
	}

	bodyLines := []string{}
	switch {
	case m.busy && m.queuedPrompt != "":
		bodyLines = append(bodyLines,
			m.theme.dim.Render("One queued message is waiting for the current run to finish."),
			m.renderQueuedPrompt(max(16, m.panelInnerWidth(m.layout.totalWidth))),
		)
	default:
		bodyLines = append(bodyLines, m.input.View())
		if m.filePicker.Active {
			bodyLines = append(bodyLines, m.renderFileCandidateList(max(16, m.panelInnerWidth(m.layout.totalWidth))))
		}
	}
	body := lipgloss.JoinVertical(lipgloss.Left, bodyLines...)

	return m.renderPanel(
		"Composer",
		hint,
		body,
		m.layout.totalWidth,
		m.layout.composerHeight,
		m.activePane == paneComposer,
		m.theme.coral,
	)
}

// renderQueuedPrompt renders the queued prompt preview
func (m model) renderQueuedPrompt(width int) string {
	preview := strings.TrimSpace(m.queuedPrompt)
	if preview == "" {
		preview = "(empty)"
	}

	lines := []string{
		m.theme.panelMeta.Render("queued next message"),
		m.theme.entryText.Width(width).Render(preview),
	}

	return m.theme.panelFill().
		PaddingLeft(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// footerHint returns the appropriate footer hint based on current state
func (m model) footerHint() string {
	switch {
	case m.busy && m.queuedPrompt != "":
		return "esc recalls queued draft | esc esc interrupts | wheel scrolls conversation"
	case m.busy:
		return "esc esc interrupts | @ opens file candidates"
	default:
		return "@ opens file candidates | wheel scrolls conversation"
	}
}
