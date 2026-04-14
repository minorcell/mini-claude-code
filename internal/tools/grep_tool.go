package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/minorcell/mini-claude-code/internal/provider"
)

// GrepTool 在文件内容中搜索匹配的文本。
type GrepTool struct {
	root string
}

type grepArgs struct {
	Path       string `json:"path"`
	Pattern    string `json:"pattern"`
	Glob       string `json:"glob,omitempty"`
	Regex      bool   `json:"regex,omitempty"`
	IgnoreCase bool   `json:"ignore_case,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
	Context    int    `json:"context,omitempty"`
}

// NewGrepTool 创建一个 grep 搜索工具实例。
func NewGrepTool(root string) *GrepTool {
	return &GrepTool{
		root: root,
	}
}

// Definition 返回 grep 工具的模型可见定义。
func (t *GrepTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "grep",
		Description: "Search file contents for matching text or regex patterns. Returns file:line: content format.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path to search in. Default is '.'.",
				},
				"pattern": map[string]any{
					"type":        "string",
					"description": "Text or regex pattern to search for.",
				},
				"glob": map[string]any{
					"type":        "string",
					"description": "File glob pattern to filter which files to search (e.g., '*.go', '*.md').",
				},
				"regex": map[string]any{
					"type":        "boolean",
					"description": "Treat pattern as regex. Default is false.",
				},
				"ignore_case": map[string]any{
					"type":        "boolean",
					"description": "Case-insensitive search. Default is false.",
				},
				"max_results": map[string]any{
					"type":        "integer",
					"description": "Maximum number of matches to return. Default is 50.",
				},
				"context": map[string]any{
					"type":        "integer",
					"description": "Number of context lines before/after each match. Default is 0.",
				},
			},
			"required": []string{"pattern"},
		},
	}
}

// Execute 执行 grep 搜索。
func (t *GrepTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args grepArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode grep arguments: %w", err)
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
		maxResults = 50
	}

	return t.grep(target, args.Pattern, args.Glob, args.Regex, args.IgnoreCase, maxResults, args.Context)
}

func (t *GrepTool) grep(dir, pattern, fileGlob string, isRegex, ignoreCase bool, maxResults, contextLines int) (Result, error) {
	var re *regexp.Regexp
	var err error

	if isRegex {
		if ignoreCase {
			pattern = "(?i)" + pattern
		}
		re, err = regexp.Compile(pattern)
		if err != nil {
			return Result{}, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
		}
	}

	var results []string
	matchCount := 0

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if matchCount >= maxResults {
			return filepath.SkipAll
		}

		if fileGlob != "" {
			matched, err := filepath.Match(fileGlob, filepath.Base(path))
			if err != nil || !matched {
				return nil
			}
		}

		matches, searchErr := t.searchFile(path, pattern, re, ignoreCase, dir, contextLines)
		if searchErr != nil {
			return searchErr
		}
		for _, m := range matches {
			if matchCount >= maxResults {
				break
			}
			results = append(results, m)
			matchCount++
		}

		return nil
	})

	if err != nil {
		return Result{}, fmt.Errorf("walk directory %q: %w", dir, err)
	}

	output := strings.Join(results, "\n")
	if len(results) == 0 {
		output = "No matches found."
	}

	return Result{
		Output: output,
		Metadata: map[string]any{
			"path":    dir,
			"pattern": pattern,
			"count":   len(results),
		},
	}, nil
}

func (t *GrepTool) searchFile(filePath, pattern string, re *regexp.Regexp, ignoreCase bool, baseDir string, contextLines int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil
	}
	defer file.Close()

	var fileLines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fileLines = append(fileLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file %q: %w", filePath, err)
	}

	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		relPath = filePath
	}

	var lines []string
	if contextLines <= 0 {
		for i, line := range fileLines {
			if lineMatched(line, pattern, re, ignoreCase) {
				lines = append(lines, fmt.Sprintf("%s:%d: %s", relPath, i+1, line))
			}
		}
		return lines, nil
	}

	selectedLineNumbers := make(map[int]struct{})
	for i, line := range fileLines {
		if !lineMatched(line, pattern, re, ignoreCase) {
			continue
		}

		start := i - contextLines
		if start < 0 {
			start = 0
		}
		end := i + contextLines
		if end >= len(fileLines) {
			end = len(fileLines) - 1
		}
		for j := start; j <= end; j++ {
			selectedLineNumbers[j] = struct{}{}
		}
	}

	for i, line := range fileLines {
		if _, ok := selectedLineNumbers[i]; !ok {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s:%d: %s", relPath, i+1, line))
	}

	return lines, nil
}

func lineMatched(line, pattern string, re *regexp.Regexp, ignoreCase bool) bool {
	if re != nil {
		return re.MatchString(line)
	}
	if ignoreCase {
		return strings.Contains(strings.ToLower(line), strings.ToLower(pattern))
	}
	return strings.Contains(line, pattern)
}
