package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/minorcell/mini-claude-code/projects/mini-opencode-go/core"
	"github.com/minorcell/mini-claude-code/projects/mini-opencode-go/provider"
)

type pane int

const (
	paneConversation pane = iota
	paneTrace
	paneComposer
)

type transcriptEntry struct {
	Role  string
	Title string
	Body  string
	Meta  string
}

type todoSidebarItem struct {
	Content string
	Status  string
}

type activityStatus string

const (
	activityInfo    activityStatus = "info"
	activityRunning activityStatus = "running"
	activityDone    activityStatus = "done"
	activityError   activityStatus = "error"
)

type activityItem struct {
	ID      string
	Step    int
	Title   string
	Summary string
	Detail  string
	Result  string
	Status  activityStatus
}

type layoutState struct {
	totalWidth         int
	headerHeight       int
	mainHeight         int
	composerHeight     int
	footerHeight       int
	stacked            bool
	conversationWidth  int
	conversationHeight int
	traceWidth         int
	traceHeight        int
}

type turnProgressMsg struct {
	event core.ProgressEvent
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
	maxSteps   int

	width   int
	height  int
	ready   bool
	busy    bool
	status  string
	lastErr error

	activePane    pane
	turnCount     int
	stepCount     int
	lastDuration  time.Duration
	turnStarted   time.Time
	turnCancel    context.CancelFunc
	turnUpdates   chan tea.Msg
	lastAction    string
	queuedPrompt  string
	escArmedUntil time.Time

	fileIndex      []string
	fileIndexReady bool
	fileIndexErr   error
	filePicker     filePickerState

	currentTurnUsage     provider.Usage
	sessionUsage         provider.Usage
	sessionUsageBase     provider.Usage
	currentToolCalls     int
	sessionToolCalls     int
	currentAssistantRuns int
	todoItems            []todoSidebarItem

	input      textarea.Model
	transcript viewport.Model
	activity   viewport.Model
	spinner    spinner.Model
	progress   progress.Model
	help       help.Model
	keys       keyMap
	theme      uiTheme
	layout     layoutState

	entries          []transcriptEntry
	trace            []activityItem
	traceIndexByCall map[string]int
	currentEventSeen map[string]int
}

func newModel(app App) model {
	theme := newTheme()
	keys := newKeyMap()

	input := textarea.New()
	input.Placeholder = "Ask mini-opencode to inspect, edit, or debug..."
	input.Prompt = ">> "
	input.ShowLineNumbers = false
	input.CharLimit = 4096
	input.SetHeight(3)
	input.FocusedStyle.Base = theme.panelFill()
	input.FocusedStyle.CursorLine = theme.panelFill()
	input.FocusedStyle.EndOfBuffer = theme.dim
	input.FocusedStyle.Prompt = theme.inputPrompt
	input.FocusedStyle.Text = theme.inputText
	input.FocusedStyle.Placeholder = theme.inputPlaceholder
	input.BlurredStyle.Base = theme.panelFill()
	input.BlurredStyle.CursorLine = theme.panelFill()
	input.BlurredStyle.EndOfBuffer = theme.dim
	input.BlurredStyle.Prompt = theme.inputPrompt
	input.BlurredStyle.Text = theme.inputText
	input.BlurredStyle.Placeholder = theme.inputPlaceholder
	input.KeyMap.InsertNewline.SetKeys("ctrl+j")
	input.KeyMap.InsertNewline.SetHelp("ctrl+j", "newline")
	input.Focus()

	spin := spinner.New(
		spinner.WithSpinner(spinner.Line),
		spinner.WithStyle(theme.spinner),
	)

	bar := progress.New(
		progress.WithScaledGradient(theme.teal, theme.gold),
		progress.WithoutPercentage(),
	)

	helpModel := help.New()
	helpModel.ShowAll = false
	helpModel.ShortSeparator = "   "
	helpModel.Styles.ShortKey = theme.helpKey
	helpModel.Styles.ShortDesc = theme.helpDesc
	helpModel.Styles.ShortSeparator = theme.helpDesc
	helpModel.Styles.Ellipsis = theme.helpDesc

	return model{
		agent:      app.Agent,
		session:    app.Session,
		configPath: app.ConfigPath,
		provider:   app.ProviderName,
		providerTy: app.ProviderType,
		modelName:  app.ModelName,
		workspace:  app.Workspace,
		maxSteps:   max(1, app.MaxSteps),
		status:     "Ready for the next prompt",
		activePane: paneComposer,
		input:      input,
		spinner:    spin,
		progress:   bar,
		help:       helpModel,
		keys:       keys,
		theme:      theme,
		entries: []transcriptEntry{
			{
				Role:  "system",
				Title: "SYSTEM",
				Body:  "Studio ready. Enter sends now or queues while working, Esc Esc interrupts, and the mouse wheel scrolls the conversation.",
				Meta:  "boot",
			},
		},
		lastAction:       "Waiting for your prompt.",
		trace:            []activityItem{},
		traceIndexByCall: make(map[string]int),
		currentEventSeen: make(map[string]int),
		todoItems:        []todoSidebarItem{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.spinner.Tick, buildFileIndexCmd(m.workspace))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	cmds = append(cmds, m.updateBubbleComponents(msg)...)

	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.configureLayout()
		return m, tea.Batch(cmds...)
	case fileIndexLoadedMsg:
		m.fileIndex = message.files
		m.fileIndexErr = message.err
		m.fileIndexReady = message.err == nil
		m.syncFilePicker()
		m.refreshViewports()
		return m, tea.Batch(cmds...)
	case tea.MouseMsg:
		handled, cmd := m.handleMouse(message)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if handled {
			return m, tea.Batch(cmds...)
		}
	case tea.KeyMsg:
		if message.String() == "esc" {
			cmds = append(cmds, m.handleEscape())
			return m, tea.Batch(cmds...)
		}
		m.resetEscArm()
		if handled, cmd := m.handleFileCandidateKeys(message); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
		switch {
		case keyMatches(message, m.keys.Quit):
			m.cancelTurn()
			return m, tea.Quit
		case keyMatches(message, m.keys.Send):
			if m.activePane == paneComposer {
				if m.busy {
					cmds = append(cmds, m.queueComposerDraft())
					return m, tea.Batch(cmds...)
				}
				return m.startTurn(cmds)
			}
			return m, tea.Batch(cmds...)
		}
	case turnProgressMsg:
		cmds = append(cmds, m.handleProgress(message.event))
		if m.turnUpdates != nil {
			cmds = append(cmds, waitTurnUpdateCmd(m.turnUpdates))
		}
		return m, tea.Batch(cmds...)
	case turnFinishedMsg:
		cmds = append(cmds, m.handleTurnFinished(message))
		return m, tea.Batch(cmds...)
	}

	if m.canEditComposer() {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
		m.syncFilePicker()
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.width < 72 || m.height < 18 {
		return "Window too small. Resize to at least 72x18."
	}
	if !m.ready {
		return "loading..."
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderMain(),
		m.renderComposer(),
		m.renderFooter(),
	)

	return m.theme.canvasFill().
		Foreground(lipgloss.Color(m.theme.cream)).
		Width(m.width).
		Height(m.height).
		Render(content)
}

func (m *model) updateBubbleComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	spin, cmd := m.spinner.Update(msg)
	m.spinner = spin
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	progressModel, cmd := m.progress.Update(msg)
	if updated, ok := progressModel.(progress.Model); ok {
		m.progress = updated
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (m *model) configureLayout() {
	totalWidth := m.width
	headerHeight := 0
	composerHeight := m.desiredComposerHeight()
	footerHeight := 1
	mainHeight := max(9, m.height-headerHeight-composerHeight-footerHeight)

	stacked := totalWidth < 120
	conversationWidth := totalWidth
	traceWidth := totalWidth
	conversationHeight := mainHeight
	traceHeight := mainHeight

	if stacked {
		traceHeight = max(6, mainHeight/4)
		conversationHeight = max(6, mainHeight-traceHeight-1)
		traceHeight = max(6, mainHeight-conversationHeight-1)
	} else {
		traceWidth = clamp(totalWidth*28/100, 26, 34)
		conversationWidth = max(30, totalWidth-traceWidth-1)
	}

	m.layout = layoutState{
		totalWidth:         totalWidth,
		headerHeight:       headerHeight,
		mainHeight:         mainHeight,
		composerHeight:     composerHeight,
		footerHeight:       footerHeight,
		stacked:            stacked,
		conversationWidth:  conversationWidth,
		conversationHeight: conversationHeight,
		traceWidth:         traceWidth,
		traceHeight:        traceHeight,
	}

	panelWidth := m.panelInnerWidth(totalWidth)
	if !m.ready {
		m.transcript = viewport.New(
			m.panelInnerWidth(conversationWidth),
			m.panelBodyHeight(conversationHeight),
		)
		m.activity = viewport.New(
			m.panelInnerWidth(traceWidth),
			m.panelBodyHeight(traceHeight),
		)
	} else {
		m.transcript.Width = m.panelInnerWidth(conversationWidth)
		m.transcript.Height = m.panelBodyHeight(conversationHeight)
		m.activity.Width = m.panelInnerWidth(traceWidth)
		m.activity.Height = m.panelBodyHeight(traceHeight)
	}
	m.transcript.Style = m.theme.panelFill()
	m.activity.Style = m.theme.panelFill()

	m.progress.Width = clamp(totalWidth/3, 16, 36)
	m.help.Width = totalWidth
	m.input.SetWidth(max(12, panelWidth))
	m.input.SetHeight(m.composerInputRows())
	m.ready = true
	m.refreshViewports()
}

func (m *model) panelInnerWidth(width int) int {
	return max(1, width-m.theme.panelBase.GetHorizontalFrameSize())
}

func (m *model) panelBodyHeight(height int) int {
	return max(1, height-m.theme.panelBase.GetVerticalFrameSize()-1)
}

func (m *model) refreshViewports() {
	m.refreshTranscript()
	m.refreshTrace()
}

func (m *model) refreshTranscript() {
	if !m.ready {
		return
	}

	stickToBottom := m.transcript.AtBottom() || m.busy
	innerWidth := max(16, m.transcript.Width)
	rendered := make([]string, 0, len(m.entries))
	for _, entry := range m.entries {
		rendered = append(rendered, m.renderTranscriptEntry(entry, innerWidth))
	}
	if len(rendered) == 0 {
		rendered = append(rendered, m.theme.dim.Render("No messages yet."))
	}
	m.transcript.SetContent(strings.Join(rendered, "\n"))
	if stickToBottom {
		m.transcript.GotoBottom()
	}
}

func (m *model) refreshTrace() {
	if !m.ready {
		return
	}

	innerWidth := max(16, m.activity.Width)
	m.activity.SetContent(m.renderOverviewBody(innerWidth))
}

func (m *model) desiredComposerHeight() int {
	if m.busy && m.queuedPrompt != "" {
		return 6
	}

	bodyRows := m.composerInputRows()
	if m.filePicker.Active {
		bodyRows += m.filePickerRows()
	}

	panelRows := bodyRows + 1 + m.theme.panelBase.GetVerticalFrameSize()
	return clamp(panelRows, 6, 14)
}

func (m model) composerInputRows() int {
	rows := strings.Count(m.input.Value(), "\n") + 1
	return clamp(rows, 3, 6)
}

func (m model) filePickerRows() int {
	if !m.filePicker.Active {
		return 0
	}
	if m.filePicker.Loading || m.fileIndexErr != nil || len(m.filePicker.Results) == 0 {
		return 2
	}
	return 1 + min(6, len(m.filePicker.Results))
}

func (m *model) syncFilePicker() {
	previousHeight := m.layout.composerHeight
	previousActive := m.filePicker.Active
	previousCount := len(m.filePicker.Results)
	previousQuery := m.filePicker.Query
	previousSelectedPath := ""
	if m.filePicker.Active && len(m.filePicker.Results) > 0 && m.filePicker.Selected < len(m.filePicker.Results) {
		previousSelectedPath = m.filePicker.Results[m.filePicker.Selected].Path
	}

	if !m.canEditComposer() {
		m.filePicker = filePickerState{}
		if m.ready && (previousActive || previousHeight != m.desiredComposerHeight()) {
			m.configureLayout()
		}
		return
	}

	next := detectFilePickerContext(m.input.Value(), m.composerCursorOffset())
	if !next.Active {
		m.filePicker = filePickerState{}
		if m.ready && (previousActive || previousHeight != m.desiredComposerHeight()) {
			m.configureLayout()
		}
		return
	}

	next.Loading = !m.fileIndexReady && m.fileIndexErr == nil
	if m.fileIndexReady {
		next.Results = searchFileCandidates(m.fileIndex, next.Query, 8)
	}

	if next.Query == previousQuery && previousSelectedPath != "" {
		for i, candidate := range next.Results {
			if candidate.Path == previousSelectedPath {
				next.Selected = i
				break
			}
		}
	}

	if next.Selected >= len(next.Results) {
		next.Selected = 0
	}

	m.filePicker = next
	if m.ready && (previousActive != m.filePicker.Active || previousCount != len(m.filePicker.Results) || previousHeight != m.desiredComposerHeight()) {
		m.configureLayout()
	}
}

func (m *model) startTurn(existing []tea.Cmd) (tea.Model, tea.Cmd) {
	return m.dispatchPrompt(m.input.Value(), existing)
}

func (m *model) dispatchPrompt(prompt string, existing []tea.Cmd) (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(prompt)
	if input == "" {
		return *m, tea.Batch(existing...)
	}

	m.entries = append(m.entries, transcriptEntry{
		Role:  "user",
		Title: "YOU",
		Body:  input,
		Meta:  "prompt",
	})
	m.input.SetValue("")
	m.busy = true
	m.turnCount++
	m.status = "Dispatching prompt to the model"
	m.lastErr = nil
	m.stepCount = 0
	m.turnStarted = time.Now()
	m.currentTurnUsage = provider.Usage{}
	m.sessionUsageBase = m.sessionUsage
	m.currentToolCalls = 0
	m.currentAssistantRuns = 0
	m.lastAction = "Prompt dispatched."
	m.filePicker = filePickerState{}
	m.traceIndexByCall = make(map[string]int)
	m.currentEventSeen = make(map[string]int)
	m.resetEscArm()

	ctx, cancel := context.WithCancel(context.Background())
	m.turnCancel = cancel
	m.turnUpdates = make(chan tea.Msg, 128)

	if m.activePane != paneComposer {
		m.input.Blur()
	} else {
		m.input.Focus()
	}
	m.syncFilePicker()
	m.refreshViewports()

	cmds := append(existing,
		m.progress.SetPercent(0),
		runTurnStreamCmd(ctx, m.agent, m.session, input, m.turnUpdates),
		waitTurnUpdateCmd(m.turnUpdates),
	)
	return *m, tea.Batch(cmds...)
}

func (m *model) handleProgress(event core.ProgressEvent) tea.Cmd {
	switch event.Kind {
	case core.ProgressEventStepStarted:
		m.stepCount = event.Step
		m.status = fmt.Sprintf("Step %d/%d: reasoning", event.Step, m.maxSteps)
		m.lastAction = fmt.Sprintf("Step %02d is reasoning.", event.Step)
		m.refreshViewports()
		return m.progress.SetPercent(stepPercent(event.Step, m.maxSteps))
	case core.ProgressEventStepCompleted:
		m.currentTurnUsage = event.TotalUsage
		m.sessionUsage = addUsage(m.sessionUsageBase, event.TotalUsage)
	case core.ProgressEventAssistantMessage:
		text := strings.TrimSpace(event.Content)
		if text == "" {
			return nil
		}
		m.currentAssistantRuns++
		m.entries = append(m.entries, transcriptEntry{
			Role:  "assistant",
			Title: "ASSISTANT",
			Body:  text,
			Meta:  fmt.Sprintf("step %02d", event.Step),
		})
		m.recordTurnEvent(core.Event{
			Kind:    core.EventAssistant,
			Content: text,
		})
		m.status = fmt.Sprintf("Step %d/%d: response ready", event.Step, m.maxSteps)
		m.lastAction = "Assistant produced a response."
	case core.ProgressEventToolStarted:
		callSummary, callDetail := summarizeToolCall(event.ToolName, event.ToolInput)
		m.status = fmt.Sprintf("Step %d/%d: %s", event.Step, m.maxSteps, previewText(callSummary, 48))
		m.lastAction = strings.TrimSpace(callSummary + " " + callDetail)
	case core.ProgressEventToolFinished:
		callSummary, callDetail := summarizeToolCall(event.ToolName, event.ToolInput)
		resultSummary, resultDetail := summarizeToolResult(event.ToolName, event.ToolInput, event.ToolOutput, event.IsError)
		m.currentToolCalls++
		m.sessionToolCalls++
		if event.IsError {
			m.status = fmt.Sprintf("Step %d/%d: %s", event.Step, m.maxSteps, previewText(resultSummary, 48))
		} else {
			m.status = fmt.Sprintf("Step %d/%d: tool finished", event.Step, m.maxSteps)
		}

		bodyLines := []string{callSummary}
		if callDetail != "" {
			bodyLines = append(bodyLines, callDetail)
		}
		bodyLines = append(bodyLines, resultSummary)
		if resultDetail != "" && resultDetail != callDetail {
			bodyLines = append(bodyLines, resultDetail)
		}
		title := strings.ToUpper(event.ToolName)
		if event.IsError {
			title += " ERROR"
		}
		m.entries = append(m.entries, transcriptEntry{
			Role:  "tool",
			Title: title,
			Body:  strings.Join(bodyLines, "\n"),
			Meta:  fmt.Sprintf("step %02d", event.Step),
		})
		m.applyToolSidebarState(event.ToolName, event.ToolOutput)
		m.recordTurnEvent(core.Event{
			Kind:       core.EventTool,
			ToolName:   event.ToolName,
			ToolInput:  event.ToolInput,
			ToolOutput: event.ToolOutput,
			IsError:    event.IsError,
		})
		m.lastAction = strings.TrimSpace(resultSummary + " " + resultDetail)
	}

	m.refreshViewports()
	return nil
}

func (m *model) handleTurnFinished(message turnFinishedMsg) tea.Cmd {
	if m.turnCancel != nil {
		m.turnCancel()
		m.turnCancel = nil
	}
	m.turnUpdates = nil
	m.busy = false
	m.resetEscArm()
	m.lastDuration = time.Since(m.turnStarted)

	m.currentTurnUsage = message.result.Usage
	m.sessionUsage = addUsage(m.sessionUsageBase, message.result.Usage)

	cmd := m.progress.SetPercent(stepPercent(message.result.Steps, m.maxSteps))
	if message.err != nil {
		m.lastErr = message.err
		m.status = fmt.Sprintf("Stopped after %d/%d step(s)", message.result.Steps, m.maxSteps)
		m.entries = append(m.entries, transcriptEntry{
			Role:  "system",
			Title: "ERROR",
			Body:  message.err.Error(),
			Meta:  "runtime",
		})
		m.lastAction = previewText(message.err.Error(), 96)
	} else {
		m.lastErr = nil
		m.status = fmt.Sprintf("Completed in %d step(s) | %s", message.result.Steps, formatDuration(m.lastDuration))
		m.lastAction = "Turn completed cleanly."
	}

	m.replayMissingTurnEvents(message.result)
	m.refreshViewports()
	m.transcript.GotoBottom()

	refreshIndexCmd := buildFileIndexCmd(m.workspace)
	if m.queuedPrompt != "" {
		queued := m.queuedPrompt
		m.queuedPrompt = ""
		m.status = "Dispatching queued prompt to the model"
		m.lastAction = "Queued prompt dispatched."
		returnCmds := []tea.Cmd{cmd, refreshIndexCmd}
		_, nextCmd := m.dispatchPrompt(queued, returnCmds)
		return nextCmd
	}

	m.syncFilePicker()
	if m.activePane == paneComposer && m.composerEditable() {
		focusCmd := m.input.Focus()
		if focusCmd != nil {
			return tea.Batch(cmd, refreshIndexCmd, focusCmd)
		}
	}
	return tea.Batch(cmd, refreshIndexCmd)
}

func (m *model) handleFileCandidateKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	if m.activePane != paneComposer || !m.filePicker.Active {
		return false, nil
	}

	switch {
	case keyMatches(msg, m.keys.Up):
		if len(m.filePicker.Results) == 0 {
			return true, nil
		}
		m.filePicker.Selected--
		if m.filePicker.Selected < 0 {
			m.filePicker.Selected = len(m.filePicker.Results) - 1
		}
		return true, nil
	case keyMatches(msg, m.keys.Down):
		if len(m.filePicker.Results) == 0 {
			return true, nil
		}
		m.filePicker.Selected = (m.filePicker.Selected + 1) % len(m.filePicker.Results)
		return true, nil
	case keyMatches(msg, m.keys.CandidateAccept), keyMatches(msg, m.keys.Send):
		if len(m.filePicker.Results) == 0 {
			return true, nil
		}
		if m.acceptSelectedFileCandidate() {
			return true, nil
		}
		return true, nil
	}

	return false, nil
}

func (m *model) acceptSelectedFileCandidate() bool {
	if !m.filePicker.Active || len(m.filePicker.Results) == 0 || m.filePicker.Selected >= len(m.filePicker.Results) {
		return false
	}

	selected := m.filePicker.Results[m.filePicker.Selected].Path
	currentValue := []rune(m.input.Value())
	replacement := []rune("@" + selected)

	head := append([]rune{}, currentValue[:m.filePicker.Start]...)
	tail := append([]rune{}, currentValue[m.filePicker.End:]...)
	newValue := append(head, replacement...)
	newCursor := len(newValue)

	if m.filePicker.End == len(currentValue) {
		newValue = append(newValue, ' ')
		newCursor++
	}
	newValue = append(newValue, tail...)

	if m.filePicker.End < len(currentValue) && unicode.IsSpace(currentValue[m.filePicker.End]) {
		newCursor++
	}

	m.input.SetValue(string(newValue))
	m.setComposerCursorOffset(newCursor)
	m.lastAction = "Attached file candidate " + selected + "."
	m.syncFilePicker()
	m.refreshViewports()
	return true
}

func (m model) composerCursorOffset() int {
	value := m.input.Value()
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		return 0
	}

	row := clamp(m.input.Line(), 0, len(lines)-1)
	offset := 0
	for i := 0; i < row; i++ {
		offset += len([]rune(lines[i])) + 1
	}
	offset += m.input.LineInfo().CharOffset
	return clamp(offset, 0, len([]rune(value)))
}

func (m *model) setComposerCursorOffset(offset int) {
	value := m.input.Value()
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		m.input.SetCursor(0)
		return
	}

	offset = clamp(offset, 0, len([]rune(value)))
	targetRow := 0
	targetCol := 0
	remaining := offset

	for i, line := range lines {
		lineLen := len([]rune(line))
		if remaining <= lineLen {
			targetRow = i
			targetCol = remaining
			break
		}

		targetRow = i
		targetCol = lineLen
		remaining -= lineLen + 1
	}

	for m.input.Line() > 0 {
		m.input.CursorStart()
		m.input.CursorUp()
	}
	m.input.CursorStart()
	for m.input.Line() < targetRow {
		m.input.CursorDown()
	}
	m.input.SetCursor(targetCol)
}

func (m *model) setActivePane(next pane) tea.Cmd {
	m.activePane = next
	m.syncFilePicker()
	if next == paneComposer && m.composerEditable() {
		return m.input.Focus()
	}
	m.input.Blur()
	return nil
}

func (m *model) handleMouse(msg tea.MouseMsg) (bool, tea.Cmd) {
	hovered := m.paneAt(msg.X, msg.Y)

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scrollViewportForPane(hovered, -3)
		return true, nil
	case tea.MouseButtonWheelDown:
		m.scrollViewportForPane(hovered, 3)
		return true, nil
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return false, nil
		}
		switch hovered {
		case paneConversation:
			return true, m.setActivePane(paneConversation)
		case paneComposer:
			if m.composerEditable() {
				return true, m.setActivePane(paneComposer)
			}
			return true, nil
		default:
			return false, nil
		}
	default:
		return false, nil
	}
}

func (m *model) scrollViewportForPane(target pane, lines int) {
	viewportModel := &m.transcript
	if target == paneConversation {
		viewportModel = &m.transcript
	}
	if lines < 0 {
		viewportModel.LineUp(-lines)
		return
	}
	viewportModel.LineDown(lines)
}

func (m model) paneAt(x int, y int) pane {
	mainTop := m.layout.headerHeight
	if y < mainTop {
		return paneComposer
	}

	if m.layout.stacked {
		if y < mainTop+m.layout.conversationHeight {
			return paneConversation
		}
		if y < mainTop+m.layout.conversationHeight+1+m.layout.traceHeight {
			return paneTrace
		}
	} else {
		if y < mainTop+m.layout.mainHeight {
			if x < m.layout.conversationWidth {
				return paneConversation
			}
			if x >= m.layout.conversationWidth+1 {
				return paneTrace
			}
		}
	}

	composerTop := mainTop + m.layout.mainHeight
	if y >= composerTop && y < composerTop+m.layout.composerHeight {
		return paneComposer
	}

	return paneComposer
}

func (m *model) cancelTurn() {
	if m.turnCancel != nil {
		m.turnCancel()
	}
}

func (m model) composerEditable() bool {
	return !m.busy || m.queuedPrompt == ""
}

func (m model) canEditComposer() bool {
	return m.activePane == paneComposer && m.composerEditable()
}

func (m *model) queueComposerDraft() tea.Cmd {
	if !m.busy {
		return nil
	}
	if m.queuedPrompt != "" {
		m.status = "One queued prompt is already waiting"
		m.lastAction = "Press Esc to pull the queued prompt back into the composer."
		return nil
	}

	draft := m.input.Value()
	if strings.TrimSpace(draft) == "" {
		m.status = "Current turn is still running"
		m.lastAction = "Type a message, then press Enter to queue it."
		return nil
	}

	m.queuedPrompt = draft
	m.input.SetValue("")
	m.input.Blur()
	m.status = "Queued one prompt behind the current run"
	m.lastAction = "Press Esc to restore the queued prompt for editing."
	m.resetEscArm()
	m.syncFilePicker()
	m.refreshViewports()
	return nil
}

func (m *model) restoreQueuedPrompt() tea.Cmd {
	if m.queuedPrompt == "" {
		return nil
	}

	m.input.SetValue(m.queuedPrompt)
	m.input.CursorEnd()
	m.queuedPrompt = ""
	m.activePane = paneComposer
	m.status = "Queued prompt restored to the composer"
	m.lastAction = "Edit the restored draft, or press Enter to queue it again."
	m.resetEscArm()
	m.syncFilePicker()
	m.refreshViewports()
	return m.input.Focus()
}

func (m *model) handleEscape() tea.Cmd {
	if m.queuedPrompt != "" {
		return m.restoreQueuedPrompt()
	}
	if !m.busy {
		return nil
	}

	now := time.Now()
	if !m.escArmedUntil.IsZero() && now.Before(m.escArmedUntil) {
		m.cancelTurn()
		m.resetEscArm()
		m.status = "Interrupt requested for the current run"
		m.lastAction = "Waiting for the current turn to stop."
		return nil
	}

	m.escArmedUntil = now.Add(1500 * time.Millisecond)
	m.status = "Press Esc again to interrupt the current run"
	m.lastAction = "Interrupt is armed for 1.5 seconds."
	return nil
}

func (m *model) resetEscArm() {
	m.escArmedUntil = time.Time{}
}

func (m *model) applyToolSidebarState(name string, rawOutput string) {
	if name != "todo" {
		return
	}
	items, ok := parseTodoToolItems(rawOutput)
	if !ok {
		return
	}
	m.todoItems = items
}

func (m *model) recordTurnEvent(event core.Event) {
	if m.currentEventSeen == nil {
		m.currentEventSeen = make(map[string]int)
	}
	m.currentEventSeen[turnEventKey(event)]++
}

func (m *model) replayMissingTurnEvents(result core.TurnResult) {
	for _, event := range result.Events {
		key := turnEventKey(event)
		if remaining := m.currentEventSeen[key]; remaining > 0 {
			m.currentEventSeen[key] = remaining - 1
			continue
		}

		switch event.Kind {
		case core.EventAssistant:
			text := strings.TrimSpace(event.Content)
			if text == "" {
				continue
			}
			m.entries = append(m.entries, transcriptEntry{
				Role:  "assistant",
				Title: "ASSISTANT",
				Body:  text,
				Meta:  fmt.Sprintf("step %02d", max(1, result.Steps)),
			})
		case core.EventTool:
			callSummary, callDetail := summarizeToolCall(event.ToolName, event.ToolInput)
			resultSummary, resultDetail := summarizeToolResult(event.ToolName, event.ToolInput, event.ToolOutput, event.IsError)
			bodyLines := []string{callSummary}
			if callDetail != "" {
				bodyLines = append(bodyLines, callDetail)
			}
			bodyLines = append(bodyLines, resultSummary)
			if resultDetail != "" && resultDetail != callDetail {
				bodyLines = append(bodyLines, resultDetail)
			}
			title := strings.ToUpper(event.ToolName)
			if event.IsError {
				title += " ERROR"
			}
			m.entries = append(m.entries, transcriptEntry{
				Role:  "tool",
				Title: title,
				Body:  strings.Join(bodyLines, "\n"),
				Meta:  fmt.Sprintf("step %02d", max(1, result.Steps)),
			})
			m.applyToolSidebarState(event.ToolName, event.ToolOutput)
		}
	}

	m.currentEventSeen = make(map[string]int)
}

func turnEventKey(event core.Event) string {
	return strings.Join([]string{
		string(event.Kind),
		event.Content,
		event.ToolName,
		event.ToolInput,
		event.ToolOutput,
		strconv.FormatBool(event.IsError),
	}, "\x00")
}

func parseTodoToolItems(rawOutput string) ([]todoSidebarItem, bool) {
	result := parseRenderedToolResult(rawOutput)
	rawItems, ok := result.Metadata["items"].([]any)
	if !ok {
		return nil, false
	}

	items := make([]todoSidebarItem, 0, len(rawItems))
	for _, rawItem := range rawItems {
		itemMap, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		content := strings.TrimSpace(stringValue(itemMap["content"]))
		status := strings.TrimSpace(stringValue(itemMap["status"]))
		if content == "" || status == "" {
			continue
		}
		items = append(items, todoSidebarItem{
			Content: content,
			Status:  status,
		})
	}

	return items, true
}

func runTurnStreamCmd(ctx context.Context, agent *core.Agent, session *core.Session, input string, updates chan<- tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer close(updates)
			result, err := agent.RunTurnWithObserver(ctx, session, input, func(event core.ProgressEvent) {
				updates <- turnProgressMsg{event: event}
			})
			updates <- turnFinishedMsg{result: result, err: err}
		}()
		return nil
	}
}

func waitTurnUpdateCmd(updates <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-updates
		if !ok {
			return nil
		}
		return msg
	}
}

func stepPercent(steps int, maxSteps int) float64 {
	if maxSteps <= 0 {
		return 0
	}
	percent := float64(steps) / float64(maxSteps)
	if percent < 0 {
		return 0
	}
	if percent > 1 {
		return 1
	}
	return percent
}

func formatDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0s"
	}
	if duration < time.Second {
		return "1s"
	}
	return duration.Truncate(time.Second).String()
}

func clamp(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func addUsage(base provider.Usage, delta provider.Usage) provider.Usage {
	return provider.Usage{
		InputTokens:  base.InputTokens + delta.InputTokens,
		OutputTokens: base.OutputTokens + delta.OutputTokens,
	}
}
