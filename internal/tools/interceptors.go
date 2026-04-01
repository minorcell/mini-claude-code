package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// WorkspaceInterceptor 用于阻止文件与命令路径逃逸出工作区。
type WorkspaceInterceptor struct {
	Root string
}

// Before 在执行前检查路径参数是否仍位于工作区内。
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

// After 是工作区拦截器的后置钩子，当前不做额外处理。
func (WorkspaceInterceptor) After(_ context.Context, _ Invocation, _ *Result, _ error) {}

// ShellSafetyInterceptor 用于拦截明显危险的 shell 指令片段。
type ShellSafetyInterceptor struct{}

// Before 在执行 bash 前检查是否命中危险命令片段。
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

// After 是 shell 安全拦截器的后置钩子，当前不做额外处理。
func (ShellSafetyInterceptor) After(_ context.Context, _ Invocation, _ *Result, _ error) {}

// SafeJoin 将候选路径限制在给定工作区内，并返回安全的绝对路径。
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

// stringArg 从已解析参数中安全读取字符串值。
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
