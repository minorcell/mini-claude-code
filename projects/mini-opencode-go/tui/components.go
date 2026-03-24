package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type renderedToolResult struct {
	OK       bool           `json:"ok"`
	Output   string         `json:"output"`
	Metadata map[string]any `json:"metadata"`
}

func (m model) renderHeader() string {
	innerWidth := max(1, m.layout.totalWidth-m.panelBorderWidth())
	innerHeight := max(0, m.layout.headerHeight-m.panelBorderHeight())
	space := m.theme.panelFill().Render(" ")

	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.theme.title.Render("MINI CLAUDE CODE"),
		space+space,
		m.renderStatusBadge(),
	)

	metaLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.theme.badge("provider "+previewText(m.provider, 18), m.theme.teal, m.theme.ink),
		space,
		m.theme.badge("type "+previewText(m.providerTy, 18), m.theme.sage, m.theme.ink),
		space,
		m.theme.badge("model "+previewText(m.modelName, 28), m.theme.gold, m.theme.ink),
	)

	workspaceLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.theme.metaKey.Render("workspace "),
		m.theme.metaValue.Render(truncateMiddle(m.workspace, m.layout.totalWidth-12)),
	)

	progressLabel := m.theme.metaKey.Render(fmt.Sprintf("steps %02d/%02d ", m.stepCount, m.maxSteps))
	progressLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		progressLabel,
		m.progress.View(),
		space+space,
		m.theme.metaValue.Render("focus "+m.activePaneLabel()),
	)

	body := m.theme.panelFill().
		Width(innerWidth).
		Height(innerHeight).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			titleLine,
			metaLine,
			workspaceLine,
			progressLine,
		))

	return m.theme.panelFrame(true, m.theme.teal).
		Width(innerWidth).
		Height(innerHeight).
		Render(body)
}

func (m model) renderMain() string {
	columnGap := m.theme.canvasFill().Render(" ")
	rowGap := m.theme.canvasFill().Width(m.layout.totalWidth).Render("")

	conversation := m.renderPanel(
		"Conversation",
		fmt.Sprintf("%d message(s) | scroll %d%%", len(m.entries), int(m.transcript.ScrollPercent()*100)),
		m.transcript.View(),
		m.layout.conversationWidth,
		m.layout.conversationHeight,
		m.activePane == paneConversation,
		m.theme.gold,
	)
	trace := m.renderPanel(
		"Context",
		fmt.Sprintf("step %02d/%02d", m.stepCount, m.maxSteps),
		m.activity.View(),
		m.layout.traceWidth,
		m.layout.traceHeight,
		false,
		m.theme.teal,
	)

	if m.layout.stacked {
		return lipgloss.JoinVertical(lipgloss.Left, conversation, rowGap, trace)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, conversation, columnGap, trace)
}

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

func (m model) renderFooter() string {
	space := m.theme.canvasFill().Render(" ")
	helpLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.help.View(m.keys),
		space+space+space,
		m.theme.dim.Render(m.footerHint()),
	)
	body := helpLine
	if m.lastErr != nil {
		body = m.theme.error.Render("error: " + m.lastErr.Error())
	}

	return m.theme.canvasFill().
		Width(m.layout.totalWidth).
		Height(m.layout.footerHeight).
		Render(body)
}

func (m model) renderPanel(title string, hint string, body string, width int, height int, focused bool, accent string) string {
	innerWidth := max(1, width-m.theme.panelBase.GetHorizontalFrameSize())
	innerHeight := max(0, height-m.panelBorderHeight())
	space := m.theme.panelFill().Render(" ")
	titleBadge := m.theme.badge(title, accent, m.theme.ink)
	hintWidth := max(0, innerWidth-lipgloss.Width(titleBadge)-1)
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleBadge,
		space,
		m.theme.panelMeta.Copy().Width(hintWidth).Align(lipgloss.Right).Render(hint),
	)

	content := m.theme.panelFill().
		Width(innerWidth).
		Height(innerHeight).
		Render(lipgloss.JoinVertical(lipgloss.Left, header, body))
	return m.theme.panelFrame(focused, accent).
		Width(max(1, width-m.panelBorderWidth())).
		Height(innerHeight).
		Render(content)
}

func (m model) panelBorderWidth() int {
	return m.theme.panelBase.GetHorizontalBorderSize() + m.theme.panelBase.GetHorizontalMargins()
}

func (m model) panelBorderHeight() int {
	return m.theme.panelBase.GetVerticalBorderSize() + m.theme.panelBase.GetVerticalMargins()
}

func (m model) renderTranscriptEntry(entry transcriptEntry, width int) string {
	accent := m.theme.muted
	switch entry.Role {
	case "user":
		accent = m.theme.teal
	case "assistant":
		accent = m.theme.gold
	case "tool":
		accent = m.theme.sage
	case "system":
		accent = m.theme.coral
	}

	metaWidth := max(0, width-lipgloss.Width(entry.Title)-4)
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.theme.badge(entry.Title, accent, m.theme.ink),
		m.theme.panelFill().Render(" "),
		m.theme.entryMeta.Copy().Width(metaWidth).Align(lipgloss.Right).Render(entry.Meta),
	)

	body := m.theme.entryText.Copy().Width(width).Render(entry.Body)
	card := m.theme.panelFill().
		MarginBottom(1)
	if entry.Role == "system" {
		body = m.theme.dim.Copy().Width(width).Render(entry.Body)
	}
	if entry.Role == "tool" {
		body = m.theme.traceResult.Copy().Width(width).Render(entry.Body)
	}
	if entry.Role == "user" || entry.Role == "assistant" {
		body = m.theme.entryText.Copy().Width(width).Render(entry.Body)
	}

	return card.Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, header, body))
}

func (m model) renderTraceItem(item activityItem, width int) string {
	statusBadge := m.renderTraceStatus(item.Status)
	stepBadge := m.theme.badge(fmt.Sprintf("%02d", item.Step), m.theme.sand, m.theme.ink)
	titleWidth := max(0, width-lipgloss.Width(stepBadge)-lipgloss.Width(statusBadge)-2)
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		stepBadge,
		m.theme.panelFill().Render(" "),
		m.theme.traceText.Copy().Width(titleWidth).Render(strings.ToUpper(item.Title)),
		m.theme.panelFill().Render(" "),
		statusBadge,
	)

	lines := []string{
		header,
		m.theme.traceText.Copy().Width(width).Render(item.Summary),
	}
	if item.Detail != "" {
		lines = append(lines, m.theme.traceDetail.Copy().Width(width).Render(item.Detail))
	}
	if item.Result != "" {
		lines = append(lines, m.theme.traceResult.Copy().Width(width).Render(item.Result))
	}

	borderColor := m.theme.ash
	if item.Status == activityRunning {
		borderColor = m.theme.gold
	}
	if item.Status == activityError {
		borderColor = m.theme.coral
	}

	return m.theme.panelFill().
		BorderStyle(lipgloss.Border{Left: "|"}).
		BorderForeground(lipgloss.Color(borderColor)).
		PaddingLeft(1).
		MarginBottom(1).
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) renderOverviewBody(width int) string {
	turnUsageTotal := m.currentTurnUsage.InputTokens + m.currentTurnUsage.OutputTokens
	sessionUsageTotal := m.sessionUsage.InputTokens + m.sessionUsage.OutputTokens
	elapsed := m.lastDuration
	if m.busy {
		elapsed = timeSince(m.turnStarted)
	}

	contextLines := []string{
		m.theme.badge("CONTEXT", m.theme.sand, m.theme.ink),
		m.theme.entryText.Copy().Bold(true).Width(width).Render(fmt.Sprintf("%s tokens", formatCount(sessionUsageTotal))),
		m.theme.traceText.Copy().Width(width).Render(previewText(m.status, max(20, width))),
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
		contextLines = append(contextLines, m.theme.dim.Copy().Width(width).Render(previewText(m.lastAction, 96)))
	}

	blocks := []string{
		lipgloss.JoinVertical(lipgloss.Left, contextLines...),
	}
	if len(m.todoItems) > 0 {
		blocks = append(blocks, "", m.renderTodoSection(width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

type metricRow struct {
	Label string
	Value string
}

func (m model) renderOverviewSection(width int, title string, rows []metricRow) string {
	lines := []string{
		m.theme.badge(strings.ToUpper(title), m.theme.sand, m.theme.ink),
	}
	for _, row := range rows {
		labelWidth := clamp(width/2, 8, 12)
		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.traceDetail.Copy().Width(labelWidth).Render(strings.ToUpper(row.Label)),
			m.theme.traceText.Copy().Width(max(1, width-labelWidth)).Align(lipgloss.Right).Render(row.Value),
		)
		lines = append(lines, line)
	}

	return m.theme.panelFill().
		BorderStyle(lipgloss.Border{Left: "|"}).
		BorderForeground(lipgloss.Color(m.theme.teal)).
		PaddingLeft(1).
		MarginBottom(1).
		Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) renderSidebarRows(width int, rows []metricRow) string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		labelWidth := clamp(width/2, 8, 12)
		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.traceDetail.Copy().Width(labelWidth).Render(strings.ToUpper(row.Label)),
			m.theme.traceText.Copy().Width(max(1, width-labelWidth)).Align(lipgloss.Right).Render(row.Value),
		)
		lines = append(lines, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m model) renderTodoSection(width int) string {
	lines := []string{
		m.theme.badge("TODO", m.theme.teal, m.theme.ink),
	}

	for _, item := range m.todoItems {
		lines = append(lines, lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.theme.traceText.Render(todoStatusIcon(item.Status)+" "),
			m.theme.entryText.Copy().Width(max(8, width-4)).Render(previewText(item.Content, max(12, width-4))),
		))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

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
			baseStyle = baseStyle.Copy().Bold(true)
			dirStyle = dirStyle.Copy().Foreground(lipgloss.Color(m.theme.gold))
		}

		baseWidth := width - 4
		if candidate.Dir != "" {
			baseWidth = max(8, width-lipgloss.Width(candidate.Dir)-7)
		}
		line := prefix + baseStyle.Copy().Width(baseWidth).Render(candidate.Base)
		if candidate.Dir != "" {
			line += m.theme.panelFill().Render(" ") + dirStyle.Render(candidate.Dir)
		}
		lines = append(lines, line)
	}

	return m.theme.panelFill().
		PaddingLeft(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m model) renderQueuedPrompt(width int) string {
	preview := strings.TrimSpace(m.queuedPrompt)
	if preview == "" {
		preview = "(empty)"
	}

	lines := []string{
		m.theme.panelMeta.Render("queued next message"),
		m.theme.entryText.Copy().Width(width).Render(preview),
	}

	return m.theme.panelFill().
		PaddingLeft(1).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

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

func formatQueued(prompt string) string {
	if strings.TrimSpace(prompt) == "" {
		return "0 / 1"
	}
	return "1 / 1"
}

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

func (m model) renderStatusBadge() string {
	label := "READY"
	background := m.theme.sage
	foreground := m.theme.ink
	if m.busy {
		label = m.spinner.View() + " WORKING"
		background = m.theme.gold
	}
	if m.lastErr != nil {
		label = "ERROR"
		background = m.theme.coral
	}
	return m.theme.badge(label, background, foreground)
}

func (m model) renderTraceStatus(status activityStatus) string {
	switch status {
	case activityRunning:
		return m.theme.badge("running", m.theme.gold, m.theme.ink)
	case activityDone:
		return m.theme.badge("done", m.theme.sage, m.theme.ink)
	case activityError:
		return m.theme.badge("error", m.theme.coral, m.theme.cream)
	default:
		return m.theme.badge("info", m.theme.sand, m.theme.ink)
	}
}

func (m model) activePaneLabel() string {
	switch m.activePane {
	case paneConversation:
		return "conversation"
	case paneTrace:
		return "trace"
	default:
		return "composer"
	}
}

func summarizeToolCall(name string, rawInput string) (string, string) {
	args := parseJSONMap(rawInput)
	switch name {
	case "bash":
		command := stringValue(args["command"])
		workingDir := stringValue(args["working_dir"])
		if workingDir != "" {
			return previewText(command, 84), "dir " + previewText(workingDir, 36)
		}
		return previewText(command, 84), "shell command"
	case "todo":
		count, inProgress, completed := summarizeTodoArgs(args)
		summary := fmt.Sprintf("todo %d item(s)", count)
		if count == 0 {
			return "todo clear", "task checklist"
		}
		return summary, fmt.Sprintf("%d in progress | %d completed", inProgress, completed)
	case "filesystem":
		action := stringValue(args["action"])
		path := stringValue(args["path"])
		summary := strings.TrimSpace(action + " " + path)
		if action == "write" {
			return previewText(summary, 84), fmt.Sprintf("%d byte payload", len(stringValue(args["content"])))
		}
		return previewText(summary, 84), "workspace filesystem"
	case "webfetch":
		url := stringValue(args["url"])
		return "fetch " + truncateMiddle(url, 72), "remote request"
	default:
		return previewText(name+" "+rawInput, 84), "tool invocation"
	}
}

func summarizeToolResult(name string, rawInput string, rawOutput string, isError bool) (string, string) {
	args := parseJSONMap(rawInput)
	result := parseRenderedToolResult(rawOutput)
	action := stringValue(args["action"])

	switch name {
	case "bash":
		if isError {
			return "command failed", previewText(firstLine(result.Output), 96)
		}
		if strings.TrimSpace(result.Output) == "(no output)" {
			return "command finished with no output", ""
		}
		return fmt.Sprintf("command finished | %d line(s)", lineCount(result.Output)), previewText(firstLine(result.Output), 96)
	case "todo":
		count := intValue(result.Metadata["count"])
		completed := intValue(result.Metadata["completed"])
		inProgress := intValue(result.Metadata["in_progress"])
		if isError {
			return "todo update failed", previewText(firstLine(result.Output), 96)
		}
		if count == 0 {
			return "todo cleared", ""
		}
		return fmt.Sprintf("todo updated | %d item(s)", count), fmt.Sprintf("%d completed | %d in progress", completed, inProgress)
	case "filesystem":
		switch action {
		case "read":
			return fmt.Sprintf("read complete | %d line(s)", lineCount(result.Output)), previewText(firstLine(result.Output), 96)
		case "write":
			return previewText(result.Output, 96), ""
		case "list":
			if count := intValue(result.Metadata["count"]); count > 0 {
				return fmt.Sprintf("listed %d item(s)", count), previewText(firstLine(result.Output), 96)
			}
			return "directory listed", previewText(firstLine(result.Output), 96)
		case "mkdir":
			return previewText(result.Output, 96), ""
		case "stat":
			return previewText(result.Output, 96), ""
		default:
			return previewText(result.Output, 96), ""
		}
	case "webfetch":
		status := stringValue(result.Metadata["status"])
		contentType := stringValue(result.Metadata["content_type"])
		summary := "fetch complete"
		if status != "" {
			summary = status
		}
		if contentType != "" {
			return summary, previewText(contentType+" | "+firstLine(result.Output), 96)
		}
		return summary, previewText(firstLine(result.Output), 96)
	default:
		if isError {
			return "tool failed", previewText(firstLine(result.Output), 96)
		}
		return "tool finished", previewText(firstLine(result.Output), 96)
	}
}

func parseJSONMap(raw string) map[string]any {
	out := map[string]any{}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return out
	}
	_ = json.Unmarshal([]byte(trimmed), &out)
	return out
}

func parseRenderedToolResult(raw string) renderedToolResult {
	result := renderedToolResult{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(raw)), &result)
	return result
}

func previewText(value string, limit int) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if normalized == "" {
		return ""
	}
	if limit <= 3 || len(normalized) <= limit {
		return normalized
	}
	return normalized[:limit-3] + "..."
}

func truncateMiddle(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 6 {
		return value[:limit]
	}
	head := (limit - 3) / 2
	tail := limit - 3 - head
	return value[:head] + "..." + value[len(value)-tail:]
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return value[:idx]
	}
	return value
}

func lineCount(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "\n") + 1
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch actual := value.(type) {
	case string:
		return actual
	default:
		return fmt.Sprint(actual)
	}
}

func intValue(value any) int {
	switch actual := value.(type) {
	case int:
		return actual
	case int32:
		return int(actual)
	case int64:
		return int(actual)
	case float64:
		return int(actual)
	default:
		return 0
	}
}

func summarizeTodoArgs(args map[string]any) (count int, inProgress int, completed int) {
	rawItems, ok := args["items"].([]any)
	if !ok {
		return 0, 0, 0
	}
	for _, rawItem := range rawItems {
		itemMap, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		count++
		switch stringValue(itemMap["status"]) {
		case "completed":
			completed++
		case "in_progress":
			inProgress++
		}
	}
	return count, inProgress, completed
}

func timeSince(start time.Time) time.Duration {
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}

func formatCount(value int) string {
	if value <= 0 {
		return "-"
	}
	if value < 1000 {
		return fmt.Sprintf("%d", value)
	}

	units := []string{"k", "m", "b"}
	scaled := float64(value)
	unitIndex := -1
	for scaled >= 1000 && unitIndex < len(units)-1 {
		scaled /= 1000
		unitIndex++
	}

	if scaled >= 10 {
		return fmt.Sprintf("%.0f%s", scaled, units[unitIndex])
	}

	formatted := fmt.Sprintf("%.1f", scaled)
	formatted = strings.TrimSuffix(formatted, ".0")
	return formatted + units[unitIndex]
}
