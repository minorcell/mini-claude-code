package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

// TodoTool 维护当前任务的待办列表，适合复杂任务的分步推进。
type TodoTool struct {
	mu    sync.Mutex
	lists map[string][]todoItem
}

type todoArgs struct {
	Items []todoItem `json:"items"`
}

type todoItem struct {
	ID      string `json:"id,omitempty"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

// NewTodoTool 创建一个 todo 工具实例。
func NewTodoTool() *TodoTool {
	return &TodoTool{
		lists: make(map[string][]todoItem),
	}
}

// Definition 返回 todo 工具的模型可见定义。
func (t *TodoTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "todo",
		Description: "Create or update the active todo list for a complex task. Replace the full list each time, keep at most one item in_progress, and mark completed work explicitly.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"items": map[string]any{
					"type":        "array",
					"description": "The full ordered todo list. Use an empty array to clear it when the task is complete.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id": map[string]any{
								"type":        "string",
								"description": "Optional stable identifier for the item.",
							},
							"content": map[string]any{
								"type":        "string",
								"description": "Short task description.",
							},
							"status": map[string]any{
								"type":        "string",
								"description": "Todo status.",
								"enum":        []string{"pending", "in_progress", "completed"},
							},
						},
						"required": []string{"content", "status"},
					},
				},
			},
			"required": []string{"items"},
		},
	}
}

// Execute 执行一次 todo 工具调用。
func (t *TodoTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args todoArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode todo arguments: %w", err)
	}

	items, err := normalizeTodoItems(args.Items)
	if err != nil {
		return Result{}, err
	}

	sessionID := strings.TrimSpace(invocation.SessionID)
	if sessionID == "" {
		sessionID = "default"
	}

	t.mu.Lock()
	t.lists[sessionID] = items
	t.mu.Unlock()

	completed, inProgress, pending := countTodoStatuses(items)
	output := fmt.Sprintf("updated %d todo item(s)", len(items))
	if len(items) == 0 {
		output = "cleared todo list"
	}

	metadataItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		metadataItems = append(metadataItems, map[string]any{
			"id":      item.ID,
			"content": item.Content,
			"status":  item.Status,
		})
	}

	return Result{
		Output: output,
		Metadata: map[string]any{
			"items":       metadataItems,
			"count":       len(items),
			"completed":   completed,
			"in_progress": inProgress,
			"pending":     pending,
		},
	}, nil
}

func normalizeTodoItems(items []todoItem) ([]todoItem, error) {
	normalized := make([]todoItem, 0, len(items))
	inProgressCount := 0

	for index, item := range items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			return nil, fmt.Errorf("todo item %d content is required", index+1)
		}

		status := strings.TrimSpace(item.Status)
		switch status {
		case "pending", "in_progress", "completed":
		default:
			return nil, fmt.Errorf("todo item %d has unsupported status %q", index+1, item.Status)
		}
		if status == "in_progress" {
			inProgressCount++
		}

		normalized = append(normalized, todoItem{
			ID:      strings.TrimSpace(item.ID),
			Content: content,
			Status:  status,
		})
	}

	if inProgressCount > 1 {
		return nil, fmt.Errorf("todo list can contain at most one in_progress item")
	}

	return normalized, nil
}

func countTodoStatuses(items []todoItem) (completed int, inProgress int, pending int) {
	for _, item := range items {
		switch item.Status {
		case "completed":
			completed++
		case "in_progress":
			inProgress++
		default:
			pending++
		}
	}
	return completed, inProgress, pending
}
