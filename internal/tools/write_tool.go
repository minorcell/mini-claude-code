package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/minorcell/mini-claude-code/internal/provider"
)

// WriteTool 提供文件写入能力，支持创建新文件和覆盖/追加内容。
type WriteTool struct {
	root string
}

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Append  bool   `json:"append,omitempty"`
	Create  bool   `json:"create,omitempty"`
}

// NewWriteTool 创建一个文件写入工具实例。
func NewWriteTool(root string) *WriteTool {
	return &WriteTool{
		root: root,
	}
}

// Definition 返回文件写入工具的模型可见定义。
func (t *WriteTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "write",
		Description: "Write content to a file. Creates parent directories automatically. Supports overwrite (default) or append mode.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path to the file.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file.",
				},
				"append": map[string]any{
					"type":        "boolean",
					"description": "Append to the file instead of replacing it. Default is false.",
				},
				"create": map[string]any{
					"type":        "boolean",
					"description": "Create the file if it doesn't exist. Default is true.",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

// Execute 执行文件写入操作。
func (t *WriteTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args writeArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode write arguments: %w", err)
	}

	if strings.TrimSpace(args.Path) == "" {
		return Result{}, fmt.Errorf("path is required")
	}

	target, err := SafeJoin(t.root, args.Path)
	if err != nil {
		return Result{}, err
	}

	create := true
	if v, ok := invocation.ParsedArgs["create"].(bool); ok {
		create = v
	}

	return t.write(target, args.Content, args.Append, create)
}

func (t *WriteTool) write(target string, content string, appendMode, create bool) (Result, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Result{}, fmt.Errorf("create parent directory for %q: %w", target, err)
	}

	flag := os.O_WRONLY
	if create {
		flag |= os.O_CREATE
	}
	if appendMode {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	file, err := os.OpenFile(target, flag, 0o644)
	if err != nil {
		return Result{}, fmt.Errorf("open file %q: %w", target, err)
	}
	defer file.Close()

	written, err := file.WriteString(content)
	if err != nil {
		return Result{}, fmt.Errorf("write file %q: %w", target, err)
	}

	action := "wrote"
	if appendMode {
		action = "appended"
	}

	return Result{
		Output: fmt.Sprintf("%s %d bytes to %s", action, written, target),
		Metadata: map[string]any{
			"path":    target,
			"bytes":   written,
			"append":  appendMode,
			"created": create,
		},
	}, nil
}
