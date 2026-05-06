package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type WorkspaceInterceptor struct {
	Root string
}

func (i WorkspaceInterceptor) Before(_ context.Context, invocation *Invocation) error {
	switch invocation.ToolName {
	case "filesystem":
		if path, ok := stringArg(invocation.ParsedArgs, "path"); ok {
			_, err := SafeJoin(i.Root, path)
			return err
		}
	case "bash":
		if workingDir, ok := stringArg(invocation.ParsedArgs, "working_dir"); ok && strings.TrimSpace(workingDir) != "" {
			_, err := SafeJoin(i.Root, workingDir)
			return err
		}
	}
	return nil
}

func (WorkspaceInterceptor) After(_ context.Context, _ Invocation, _ *Result, _ error) {}

type ShellSafetyInterceptor struct{}

func (ShellSafetyInterceptor) Before(_ context.Context, invocation *Invocation) error {
	if invocation.ToolName != "bash" {
		return nil
	}

	command, _ := stringArg(invocation.ParsedArgs, "command")
	command = strings.ToLower(command)
	disallowed := []string{
		"rm -rf /",
		"mkfs",
		"shutdown",
		"reboot",
		"poweroff",
		":(){:|:&};:",
	}

	for _, fragment := range disallowed {
		if strings.Contains(command, fragment) {
			return fmt.Errorf("bash command blocked by safety policy: %q", fragment)
		}
	}
	return nil
}

func (ShellSafetyInterceptor) After(_ context.Context, _ Invocation, _ *Result, _ error) {}

func SafeJoin(root string, candidate string) (string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace %q: %w", root, err)
	}

	if strings.TrimSpace(candidate) == "" {
		candidate = "."
	}

	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(rootAbs, candidate)
	}

	target := filepath.Clean(candidate)
	rel, err := filepath.Rel(rootAbs, target)
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace %q", target, rootAbs)
	}
	return target, nil
}

func stringArg(arguments map[string]any, key string) (string, bool) {
	if arguments == nil {
		return "", false
	}
	value, ok := arguments[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}
