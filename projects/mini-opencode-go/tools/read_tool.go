package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

// ReadTool 提供文件读取能力，支持按行号范围读取。
type ReadTool struct {
	root         string
	maxReadBytes int
}

type readArgs struct {
	Path     string `json:"path"`
	Offset   int    `json:"offset,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	MaxBytes int    `json:"max_bytes,omitempty"`
}

// NewReadTool 创建一个文件读取工具实例。
func NewReadTool(root string) *ReadTool {
	return &ReadTool{
		root:         root,
		maxReadBytes: 64 * 1024,
	}
}

// Definition 返回文件读取工具的模型可见定义。
func (t *ReadTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "read",
		Description: "Read file contents. Supports reading by line range (offset/limit) or full file. Returns file content with line numbers.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path to the file.",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "Starting line number (1-indexed). Default is 1.",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum number of lines to read. Default is all lines.",
				},
				"max_bytes": map[string]any{
					"type":        "integer",
					"description": "Maximum bytes to read. Default is 64KB.",
				},
			},
			"required": []string{"path"},
		},
	}
}

// Execute 执行文件读取操作。
func (t *ReadTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args readArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode read arguments: %w", err)
	}

	if strings.TrimSpace(args.Path) == "" {
		return Result{}, fmt.Errorf("path is required")
	}

	target, err := SafeJoin(t.root, args.Path)
	if err != nil {
		return Result{}, err
	}

	maxBytes := t.maxReadBytes
	if args.MaxBytes > 0 {
		maxBytes = args.MaxBytes
	}

	offset := args.Offset
	if offset < 0 {
		offset = 0
	}
	limit := args.Limit
	if limit < 0 {
		limit = 0
	}

	return t.readLines(target, offset, limit, maxBytes)
}

func (t *ReadTool) readLines(target string, offset, limit, maxBytes int) (Result, error) {
	file, err := os.Open(target)
	if err != nil {
		return Result{}, fmt.Errorf("open file %q: %w", target, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxBytes)

	var lines []string
	totalLines := 0
	truncated := false
	bytesRead := 0

	for scanner.Scan() {
		totalLines++
		line := scanner.Text()
		bytesRead += len(line) + 1

		if bytesRead > maxBytes {
			truncated = true
			break
		}

		if offset > 0 && totalLines < offset {
			continue
		}

		if limit > 0 && len(lines) >= limit {
			continue
		}

		lines = append(lines, fmt.Sprintf("%6d\t%s", totalLines, line))
	}

	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("scan file %q: %w", target, err)
	}

	output := strings.Join(lines, "\n")
	if truncated {
		output += "\n... (truncated)"
	}

	return Result{
		Output: output,
		Metadata: map[string]any{
			"path":        target,
			"total_lines": totalLines,
			"read_lines":  len(lines),
			"offset":      offset,
			"limit":       limit,
			"truncated":   truncated,
		},
	}, nil
}
