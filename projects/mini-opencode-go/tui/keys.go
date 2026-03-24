package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type keyMap struct {
	Send            key.Binding
	CandidateAccept key.Binding
	Up              key.Binding
	Down            key.Binding
	Quit            key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send/queue"),
		),
		CandidateAccept: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "accept candidate"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("up", "select previous"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("down", "select next"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Send, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.Quit},
	}
}

func keyMatches(msg tea.KeyMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}
