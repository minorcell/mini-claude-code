// Package tui 提供基于 Bubble Tea 的终端交互界面。
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/minorcell/mini-claude-code/internal/core"
)

// App 聚合启动 TUI 所需的运行时依赖和展示信息。
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

// Run 启动 TUI 主循环。
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
