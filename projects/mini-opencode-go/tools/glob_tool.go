package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/minorcell/mini-claude-code/projects/mini-opencode-go/provider"
)

// GlobTool 按文件名模式匹配查找文件。
type GlobTool struct {
	root string
}

type globArgs struct {
	Path       string `json:"path"`
	Pattern    string `json:"pattern"`
	MaxResults int    `json:"max_results,omitempty"`
}

// NewGlobTool 创建一个 glob 搜索工具实例。
func NewGlobTool(root string) *GlobTool {
	return &GlobTool{
		root: root,
	}
}

// Definition 返回 glob 工具的模型可见定义。
func (t *GlobTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "glob",
		Description: "Find files by glob pattern. Supports *, **, ?, [abc] patterns. Returns matching file paths sorted alphabetically.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path to search in. Default is '.'.",
				},
				"pattern": map[string]any{
					"type":        "string",
					"description": "Glob pattern (e.g., '*.go', '**/*.txt', 'src/**/*.js').",
				},
				"max_results": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results. Default is 100.",
				},
			},
			"required": []string{"pattern"},
		},
	}
}

// Execute 执行 glob 搜索。
func (t *GlobTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args globArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode glob arguments: %w", err)
	}

	if strings.TrimSpace(args.Pattern) == "" {
		return Result{}, fmt.Errorf("pattern is required")
	}

	searchPath := args.Path
	if strings.TrimSpace(searchPath) == "" {
		searchPath = "."
	}

	target, err := SafeJoin(t.root, searchPath)
	if err != nil {
		return Result{}, err
	}

	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 100
	}

	return t.glob(target, args.Pattern, maxResults)
}

func (t *GlobTool) glob(dir, pattern string, maxResults int) (Result, error) {
	fullPattern := filepath.Join(dir, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return Result{}, fmt.Errorf("glob pattern %q: %w", pattern, err)
	}

	sort.Strings(matches)

	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	lines := make([]string, 0, len(matches))
	for _, match := range matches {
		relPath, err := filepath.Rel(dir, match)
		if err != nil {
			relPath = match
		}

		info, err := os.Stat(match)
		suffix := ""
		size := ""
		if err == nil {
			if info.IsDir() {
				suffix = "/"
			}
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}

		lines = append(lines, relPath+suffix+size)
	}

	return Result{
		Output: strings.Join(lines, "\n"),
		Metadata: map[string]any{
			"path":    dir,
			"pattern": pattern,
			"count":   len(lines),
		},
	}, nil
}
