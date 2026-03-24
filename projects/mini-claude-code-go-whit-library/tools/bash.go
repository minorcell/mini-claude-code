package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

type BashTool struct {
	root            string
	defaultTimeout  time.Duration
	maxOutputLength int
}

type bashArgs struct {
	Command        string `json:"command"`
	WorkingDir     string `json:"working_dir,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

func NewBashTool(root string) *BashTool {
	return &BashTool{
		root:            root,
		defaultTimeout:  20 * time.Second,
		maxOutputLength: 64 * 1024,
	}
}

func (t *BashTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "bash",
		Description: "Run a shell command inside the workspace.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Shell command to execute.",
				},
				"working_dir": map[string]any{
					"type":        "string",
					"description": "Optional workspace-relative working directory.",
				},
				"timeout_seconds": map[string]any{
					"type":        "integer",
					"description": "Optional timeout in seconds, capped internally.",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (t *BashTool) Execute(ctx context.Context, invocation Invocation) (Result, error) {
	var args bashArgs
	if err := json.Unmarshal(invocation.Arguments, &args); err != nil {
		return Result{}, fmt.Errorf("decode bash arguments: %w", err)
	}
	if strings.TrimSpace(args.Command) == "" {
		return Result{}, fmt.Errorf("bash command is required")
	}

	workingDir := t.root
	if strings.TrimSpace(args.WorkingDir) != "" {
		var err error
		workingDir, err = SafeJoin(t.root, args.WorkingDir)
		if err != nil {
			return Result{}, err
		}
	}

	timeout := t.defaultTimeout
	if args.TimeoutSeconds > 0 {
		timeout = time.Duration(args.TimeoutSeconds) * time.Second
		if timeout > 2*time.Minute {
			timeout = 2 * time.Minute
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "/bin/sh", "-lc", args.Command)
	cmd.Dir = workingDir
	output, err := cmd.CombinedOutput()
	outputText, truncated := trimOutput(output, t.maxOutputLength)

	metadata := map[string]any{
		"command":         args.Command,
		"working_dir":     workingDir,
		"timeout_seconds": int(timeout.Seconds()),
		"truncated":       truncated,
	}
	if outputText == "" {
		outputText = "(no output)"
	}

	if runCtx.Err() == context.DeadlineExceeded {
		return Result{
			Output:   outputText,
			Metadata: metadata,
		}, fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		metadata["error"] = err.Error()
		return Result{
			Output:   outputText,
			Metadata: metadata,
		}, fmt.Errorf("command failed: %w", err)
	}

	return Result{
		Output:   outputText,
		Metadata: metadata,
	}, nil
}

func trimOutput(data []byte, limit int) (string, bool) {
	if len(data) <= limit {
		return string(data), false
	}
	return string(data[:limit]), true
}
