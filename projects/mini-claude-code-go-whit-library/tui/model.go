package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/core"
)

type transcriptEntry struct {
	role    string
	content string
}

type turnFinishedMsg struct {
	result core.TurnResult
	err    error
}

type model struct {
	agent      *core.Agent
	session    *core.Session
	configPath string
	provider   string
	providerTy string
	modelName  string
	workspace  string

	width    int
	height   int
	ready    bool
	busy     bool
	status   string
	lastErr  error
	input    textinput.Model
	viewport viewport.Model
	entries  []transcriptEntry
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F4D35E"))
	roleStyles = map[string]lipgloss.Style{
		"user":      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#9AD1D4")),
		"assistant": lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EE964B")),
		"tool":      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#C6DABF")),
		"system":    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F1F1F1")),
	}
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F1F1F1")).
			Background(lipgloss.Color("#283D3B")).
			Padding(0, 1)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
)

func newModel(app App) model {
	input := textinput.New()
	input.Placeholder = "Ask mini-claude-code..."
	input.Prompt = "> "
	input.Focus()

	return model{
		agent:      app.Agent,
		session:    app.Session,
		configPath: app.ConfigPath,
		provider:   app.ProviderName,
		providerTy: app.ProviderType,
		modelName:  app.ModelName,
		workspace:  app.Workspace,
		status:     "Ready",
		input:      input,
		entries: []transcriptEntry{
			{
				role:    "system",
				content: "mini-claude-code is ready. Press Enter to send, Esc to quit.",
			},
		},
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.configureLayout()
		return m, nil
	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.busy {
				return m, nil
			}
			input := strings.TrimSpace(m.input.Value())
			if input == "" {
				return m, nil
			}

			m.entries = append(m.entries, transcriptEntry{role: "user", content: input})
			m.input.SetValue("")
			m.busy = true
			m.lastErr = nil
			m.status = "Thinking..."
			m.refreshViewport()
			return m, runTurnCmd(m.agent, m.session, input)
		}
	case turnFinishedMsg:
		m.busy = false
		if message.err != nil {
			m.lastErr = message.err
			m.entries = append(m.entries, transcriptEntry{
				role:    "system",
				content: "Error: " + message.err.Error(),
			})
			m.status = "Error"
			m.refreshViewport()
			return m, nil
		}

		for _, event := range message.result.Events {
			switch event.Kind {
			case core.EventAssistant:
				m.entries = append(m.entries, transcriptEntry{
					role:    "assistant",
					content: event.Content,
				})
			case core.EventTool:
				m.entries = append(m.entries, transcriptEntry{
					role: "tool",
					content: fmt.Sprintf("[%s]\nargs:\n%s\n\nresult:\n%s",
						event.ToolName,
						event.ToolInput,
						event.ToolOutput,
					),
				})
			}
		}

		m.status = fmt.Sprintf("Done in %d step(s)", message.result.Steps)
		m.refreshViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "loading..."
	}

	header := titleStyle.Render("mini-claude-code") + "\n" + statusStyle.Render(
		fmt.Sprintf("provider=%s type=%s model=%s workspace=%s", m.provider, m.providerTy, m.modelName, m.workspace),
	)

	footer := statusStyle.Render("config: " + m.configPath)
	if m.lastErr != nil {
		footer += "\n" + errorStyle.Render(m.lastErr.Error())
	}
	if m.busy {
		footer += "\n" + statusStyle.Render(m.status)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		m.viewport.View(),
		m.input.View(),
		footer,
	)
}

func (m *model) configureLayout() {
	inputHeight := 1
	headerHeight := 2
	footerHeight := 2
	if m.lastErr != nil || m.busy {
		footerHeight++
	}

	viewportHeight := m.height - headerHeight - inputHeight - footerHeight
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	m.viewport = viewport.New(m.width, viewportHeight)
	m.ready = true
	m.refreshViewport()
}

func (m *model) refreshViewport() {
	if !m.ready {
		return
	}

	var lines []string
	for _, entry := range m.entries {
		style, ok := roleStyles[entry.role]
		if !ok {
			style = lipgloss.NewStyle()
		}
		lines = append(lines, style.Render(strings.ToUpper(entry.role)))
		lines = append(lines, entry.content)
		lines = append(lines, "")
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoBottom()
}

func runTurnCmd(agent *core.Agent, session *core.Session, input string) tea.Cmd {
	return func() tea.Msg {
		result, err := agent.RunTurn(context.Background(), session, input)
		return turnFinishedMsg{
			result: result,
			err:    err,
		}
	}
}
