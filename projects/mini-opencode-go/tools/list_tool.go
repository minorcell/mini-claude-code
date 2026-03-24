package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

// ListTool 提供目录列表和文件信息查询能力。
type ListTool struct {
	root string
}

type listArgs struct {
	Path    string `json:"path"`
	Recurse bool   `json:"recurse,omitempty"`
	All     bool   `json:"all,omitempty"`
}

// NewListTool 创建一个目录列表工具实例。
func NewListTool(root string) *ListTool {
	return &ListTool{
		root: root,
	}
}

// Definition 返回目录列表工具的模型可见定义。
func (t *ListTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "list",
		Description: "List directory contents. Supports recursive listing and showing hidden files.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path to the directory. Use '.' for current directory.",
				},
				"recurse": map[string]any{
					"type":        "boolean",
					"description": "List contents recursively. Default is false.",
				},
				"all": map[string]any{
					"type":        "boolean",
					"description": "Include hidden files (starting with '.'). Default is false.",
				},
			},
			"required": []string{"path"},
		},
	}
}

// Execute 执行目录列表操作。
func (t *ListTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args listArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode list arguments: %w", err)
	}

	if strings.TrimSpace(args.Path) == "" {
		args.Path = "."
	}

	target, err := SafeJoin(t.root, args.Path)
	if err != nil {
		return Result{}, err
	}

	if args.Recurse {
		return t.listRecursive(target, args.All)
	}
	return t.list(target, args.All)
}

func (t *ListTool) list(target string, showAll bool) (Result, error) {
	entries, err := os.ReadDir(target)
	if err != nil {
		return Result{}, fmt.Errorf("list directory %q: %w", target, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !showAll && strings.HasPrefix(name, ".") {
			continue
		}

		suffix := ""
		if entry.IsDir() {
			suffix = "/"
		}

		info, err := entry.Info()
		size := ""
		if err == nil {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}

		lines = append(lines, name+suffix+size)
	}

	return Result{
		Output: strings.Join(lines, "\n"),
		Metadata: map[string]any{
			"path":  target,
			"count": len(lines),
		},
	}, nil
}

func (t *ListTool) listRecursive(target string, showAll bool) (Result, error) {
	var lines []string

	err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(target, path)
		if err != nil {
			return err
		}

		if !showAll && strings.HasPrefix(info.Name(), ".") && relPath != "." {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		suffix := ""
		if info.IsDir() {
			suffix = "/"
		}

		size := fmt.Sprintf(" (%d bytes)", info.Size())
		lines = append(lines, relPath+suffix+size)
		return nil
	})

	if err != nil {
		return Result{}, fmt.Errorf("walk directory %q: %w", target, err)
	}

	return Result{
		Output: strings.Join(lines, "\n"),
		Metadata: map[string]any{
			"path":  target,
			"count": len(lines),
		},
	}, nil
}
