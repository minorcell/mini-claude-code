package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var panelBorder = lipgloss.NormalBorder()

type uiTheme struct {
	canvas string
	panel  string
	ink    string
	cream  string
	sand   string
	teal   string
	gold   string
	coral  string
	sage   string
	muted  string
	ash    string

	panelBase        lipgloss.Style
	title            lipgloss.Style
	subtitle         lipgloss.Style
	metaKey          lipgloss.Style
	metaValue        lipgloss.Style
	panelMeta        lipgloss.Style
	dim              lipgloss.Style
	error            lipgloss.Style
	spinner          lipgloss.Style
	inputPrompt      lipgloss.Style
	inputText        lipgloss.Style
	inputPlaceholder lipgloss.Style
	helpKey          lipgloss.Style
	helpDesc         lipgloss.Style
	entryMeta        lipgloss.Style
	entryText        lipgloss.Style
	traceText        lipgloss.Style
	traceDetail      lipgloss.Style
	traceResult      lipgloss.Style
	statusText       lipgloss.Style
}

func newTheme() uiTheme {
	panel := lipgloss.Color("#201B18")

	return uiTheme{
		canvas: "#14110F",
		panel:  "#201B18",
		ink:    "#1F1A17",
		cream:  "#FFF9F0",
		sand:   "#F3E9DC",
		teal:   "#6C9A8B",
		gold:   "#E6B655",
		coral:  "#D97B66",
		sage:   "#A8B89A",
		muted:  "#8E8478",
		ash:    "#5F574F",
		panelBase: lipgloss.NewStyle().
			BorderStyle(panelBorder).
			Background(panel).
			Padding(0, 1),
		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E6B655")),
		subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		metaKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		metaValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF9F0")),
		panelMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		dim: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D97B66")),
		spinner: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E6B655")),
		inputPrompt: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#D97B66")),
		inputText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF9F0")),
		inputPlaceholder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		helpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D97B66")),
		helpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		entryMeta: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		entryText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF9F0")),
		traceText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF9F0")),
		traceDetail: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8E8478")),
		traceResult: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3E9DC")),
		statusText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF9F0")),
	}
}

func (t uiTheme) panelFrame(focused bool, accent string) lipgloss.Style {
	borderColor := t.ash
	if focused {
		borderColor = accent
	}
	return t.panelBase.BorderForeground(lipgloss.Color(borderColor))
}

func (t uiTheme) panelFill() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(t.panel))
}

func (t uiTheme) canvasFill() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(t.canvas))
}

func (t uiTheme) badge(label string, background string, foreground string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(foreground)).
		Background(lipgloss.Color(background)).
		Padding(0, 1).
		Render(label)
}
