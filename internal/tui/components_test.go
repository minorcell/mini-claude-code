package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFormatCount(t *testing.T) {
	tests := []struct {
		value int
		want  string
	}{
		{value: 0, want: "-"},
		{value: 999, want: "999"},
		{value: 1200, want: "1.2k"},
		{value: 24000, want: "24k"},
		{value: 1250000, want: "1.2m"},
	}

	for _, test := range tests {
		if got := formatCount(test.value); got != test.want {
			t.Fatalf("formatCount(%d) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestRenderedSectionsFitAllocatedLayout(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "wide", width: 160, height: 42},
		{name: "stacked", width: 100, height: 36},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newModel(App{MaxSteps: 24})
			m.width = test.width
			m.height = test.height
			m.configureLayout()

			if got := lipgloss.Width(m.renderMain()); got > m.layout.totalWidth {
				t.Fatalf("renderMain width = %d, want <= %d", got, m.layout.totalWidth)
			}
			if got := lipgloss.Height(m.renderMain()); got > m.layout.mainHeight {
				t.Fatalf("renderMain height = %d, want <= %d", got, m.layout.mainHeight)
			}
			if got := lipgloss.Width(m.renderComposer()); got > m.layout.totalWidth {
				t.Fatalf("renderComposer width = %d, want <= %d", got, m.layout.totalWidth)
			}
			if got := lipgloss.Height(m.renderComposer()); got > m.layout.composerHeight {
				t.Fatalf("renderComposer height = %d, want <= %d", got, m.layout.composerHeight)
			}
		})
	}
}
