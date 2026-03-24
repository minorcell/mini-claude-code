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

type FileSystemTool struct {
	root         string
	maxReadBytes int
}

type fileSystemArgs struct {
	Action  string `json:"action"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Append  bool   `json:"append,omitempty"`
}

func NewFileSystemTool(root string) *FileSystemTool {
	return &FileSystemTool{
		root:         root,
		maxReadBytes: 64 * 1024,
	}
}

func (t *FileSystemTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "filesystem",
		Description: "Read, write, list, stat, or create directories inside the workspace.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type": "string",
					"enum": []string{"read", "write", "list", "mkdir", "stat"},
				},
				"path": map[string]any{
					"type":        "string",
					"description": "Workspace-relative path.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write when action is write.",
				},
				"append": map[string]any{
					"type":        "boolean",
					"description": "Append to the file instead of replacing it.",
				},
			},
			"required": []string{"action", "path"},
		},
	}
}

func (t *FileSystemTool) Execute(_ context.Context, invocation Invocation) (Result, error) {
	var args fileSystemArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode filesystem arguments: %w", err)
	}
	if strings.TrimSpace(args.Action) == "" {
		return Result{}, fmt.Errorf("filesystem action is required")
	}

	target, err := SafeJoin(t.root, args.Path)
	if err != nil {
		return Result{}, err
	}

	switch args.Action {
	case "read":
		return t.read(target)
	case "write":
		return t.write(target, args.Content, args.Append)
	case "list":
		return t.list(target)
	case "mkdir":
		return t.mkdir(target)
	case "stat":
		return t.stat(target)
	default:
		return Result{}, fmt.Errorf("unsupported filesystem action %q", args.Action)
	}
}

func (t *FileSystemTool) read(target string) (Result, error) {
	data, err := os.ReadFile(target)
	if err != nil {
		return Result{}, fmt.Errorf("read file %q: %w", target, err)
	}

	truncated := false
	if len(data) > t.maxReadBytes {
		data = data[:t.maxReadBytes]
		truncated = true
	}

	return Result{
		Output: string(data),
		Metadata: map[string]any{
			"path":      target,
			"truncated": truncated,
		},
	}, nil
}

func (t *FileSystemTool) write(target string, content string, appendMode bool) (Result, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Result{}, fmt.Errorf("create parent directory for %q: %w", target, err)
	}

	flag := os.O_CREATE | os.O_WRONLY
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

	return Result{
		Output: fmt.Sprintf("wrote %d bytes to %s", written, target),
		Metadata: map[string]any{
			"path":   target,
			"bytes":  written,
			"append": appendMode,
		},
	}, nil
}

func (t *FileSystemTool) list(target string) (Result, error) {
	entries, err := os.ReadDir(target)
	if err != nil {
		return Result{}, fmt.Errorf("list directory %q: %w", target, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		suffix := ""
		if entry.IsDir() {
			suffix = "/"
		}
		lines = append(lines, entry.Name()+suffix)
	}

	return Result{
		Output: strings.Join(lines, "\n"),
		Metadata: map[string]any{
			"path":  target,
			"count": len(lines),
		},
	}, nil
}

func (t *FileSystemTool) mkdir(target string) (Result, error) {
	if err := os.MkdirAll(target, 0o755); err != nil {
		return Result{}, fmt.Errorf("create directory %q: %w", target, err)
	}
	return Result{
		Output: fmt.Sprintf("created directory %s", target),
		Metadata: map[string]any{
			"path": target,
		},
	}, nil
}

func (t *FileSystemTool) stat(target string) (Result, error) {
	info, err := os.Stat(target)
	if err != nil {
		return Result{}, fmt.Errorf("stat %q: %w", target, err)
	}

	return Result{
		Output: fmt.Sprintf("%s (%d bytes)", info.Mode().String(), info.Size()),
		Metadata: map[string]any{
			"path":     target,
			"name":     info.Name(),
			"size":     info.Size(),
			"mode":     info.Mode().String(),
			"modified": info.ModTime().Format("2006-01-02 15:04:05"),
			"is_dir":   info.IsDir(),
		},
	}, nil
}
