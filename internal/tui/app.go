package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/minorcell/mini-claude-code/internal/core"
)

type App struct {
	Agent        *core.Agent
	Session      *core.Session
	ConfigPath   string
	ProviderName string
	ProviderType string
	ModelName    string
	MaxSteps     int
	Workspace    string
}

func Run(app App) error {
	program := tea.NewProgram(
		newModel(app),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}
