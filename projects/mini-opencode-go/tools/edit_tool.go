package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

// EditTool 提供文件内容精确编辑能力。
type EditTool struct {
	root string
}

type editArgs struct {
	Path       string `json:"path"`
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
	All        bool   `json:"all,omitempty"`
}

// NewEditTool 创建一个文件编辑工具实例。
func NewEditTool(root string) *EditTool {
	return &EditTool{
		root: root,
	}
}

// Definition 返回文件编辑工具的模型可见定义。
func (t *EditTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "edit",
		Description: "Edit file content by replacing old_content with new_content. Use for precise, targeted edits to existing files.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path to the file.",
				},
				"old_content": map[string]any{
					"type":        "string",
					"description": "The exact content to find and replace. Must be unique in the file unless 'all' is true.",
				},
				"new_content": map[string]any{
					"type":        "string",
					"description": "The content to replace old_content with.",
				},
				"all": map[string]any{
					"type":        "boolean",
					"description": "Replace all occurrences of old_content. Default is false.",
				},
			},
			"required": []string{"path", "old_content", "new_content"},
		},
	}
}

// Execute 执行文件编辑操作。
func (t *EditTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args editArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode edit arguments: %w", err)
	}

	if strings.TrimSpace(args.Path) == "" {
		return Result{}, fmt.Errorf("path is required")
	}

	if args.OldContent == "" {
		return Result{}, fmt.Errorf("old_content is required")
	}

	target, err := SafeJoin(t.root, args.Path)
	if err != nil {
		return Result{}, err
	}

	return t.edit(target, args.OldContent, args.NewContent, args.All)
}

func (t *EditTool) edit(target, oldContent, newContent string, all bool) (Result, error) {
	data, err := os.ReadFile(target)
	if err != nil {
		return Result{}, fmt.Errorf("read file %q: %w", target, err)
	}

	content := string(data)
	count := strings.Count(content, oldContent)

	if count == 0 {
		return Result{}, fmt.Errorf("old_content not found in %q", target)
	}

	if !all && count > 1 {
		return Result{}, fmt.Errorf("found %d occurrences of old_content in %q, use 'all' to replace all", count, target)
	}

	var newFileContent string
	if all {
		newFileContent = strings.ReplaceAll(content, oldContent, newContent)
	} else {
		newFileContent = strings.Replace(content, oldContent, newContent, 1)
	}

	if err := os.WriteFile(target, []byte(newFileContent), 0o644); err != nil {
		return Result{}, fmt.Errorf("write file %q: %w", target, err)
	}

	replacedCount := count
	if !all {
		replacedCount = 1
	}

	return Result{
		Output: fmt.Sprintf("replaced %d occurrence(s) in %s", replacedCount, target),
		Metadata: map[string]any{
			"path":           target,
			"replaced_count": replacedCount,
			"total_found":    count,
			"all":            all,
		},
	}, nil
}
