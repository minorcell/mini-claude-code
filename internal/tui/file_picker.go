package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// renderFileCandidateList renders the file candidate list for the file picker
func (m model) renderFileCandidateList(width int) string {
	if m.filePicker.Loading {
		return m.theme.dim.Render("Indexing workspace files...")
	}
	if m.fileIndexErr != nil {
		return m.theme.error.Render("File index error: " + previewText(m.fileIndexErr.Error(), 72))
	}
	if len(m.filePicker.Results) == 0 {
		return m.theme.dim.Render("No files match @" + m.filePicker.Query)
	}

	lines := []string{
		m.theme.panelMeta.Render("file candidates"),
	}
	for i, candidate := range m.filePicker.Results {
		if i >= 6 {
			break
		}
		prefix := "  "
		baseStyle := m.theme.traceText
		dirStyle := m.theme.traceDetail
		if i == m.filePicker.Selected {
			prefix = "> "
			baseStyle = baseStyle.Bold(true)
			dirStyle = dirStyle.Foreground(lipgloss.Color(m.theme.gold))
		}

		baseWidth := width - 4
		if candidate.Dir != "" {
			baseWidth = max(8, width-lipgloss.Width(candidate.Dir)-7)
		}
		line := prefix + baseStyle.Width(baseWidth).Render(candidate.Base)
		if candidate.Dir != "" {
			line += m.theme.panelFill().Render(" ") + dirStyle.Render(candidate.Dir)
		}
		lines = append(lines, line)
	}

	return m.theme.panelFill().
		PaddingLeft(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}
