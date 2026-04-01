package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type renderedToolResult struct {
	OK       bool           `json:"ok"`
	Output   string         `json:"output"`
	Metadata map[string]any `json:"metadata"`
}

// summarizeToolCall creates a summary of a tool call for display
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

// summarizeToolResult creates a summary of a tool result for display
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

// parseJSONMap parses a JSON string into a map
func parseJSONMap(raw string) map[string]any {
	out := map[string]any{}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return out
	}
	_ = json.Unmarshal([]byte(trimmed), &out)
	return out
}

// parseRenderedToolResult parses a rendered tool result from JSON
func parseRenderedToolResult(raw string) renderedToolResult {
	result := renderedToolResult{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(raw)), &result)
	return result
}

// previewText creates a preview of text with a limit
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

// truncateMiddle truncates text from the middle
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

// firstLine returns the first line of text
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

// lineCount returns the number of lines in text
func lineCount(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "\n") + 1
}

// stringValue converts any value to string
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

// intValue converts any value to int
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

// summarizeTodoArgs summarizes todo arguments
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

// timeSince returns the duration since a given time
func timeSince(start time.Time) time.Duration {
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}

// formatCount formats a count with appropriate units
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
