// Package tools 提供统一工具注册、拦截与内置工具实现。
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/minorcell/mini-claude-code/internal/provider"
)

// State 描述一次工具执行时的上下文状态。
type State struct {
	WorkingDir string
	SessionID  string
}

// Invocation 表示一次已经解析好的工具调用请求。
type Invocation struct {
	ToolName   string
	CallID     string
	WorkingDir string
	SessionID  string
	Arguments  json.RawMessage
	ParsedArgs map[string]any
}

// Result 表示工具执行完成后的统一结果。
type Result struct {
	ToolName string         `json:"tool_name"`
	OK       bool           `json:"ok"`
	Output   string         `json:"output"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Render 将工具结果渲染为适合回写给模型的 JSON 文本。
func (r Result) Render() string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"output":"%s"}`, err.Error())
	}
	return string(data)
}

// Tool 定义内置工具需要实现的统一接口。
type Tool interface {
	// Definition 返回暴露给模型的工具描述。
	Definition() provider.ToolDefinition
	// Execute 执行一次工具调用。
	Execute(ctx context.Context, invocation Invocation) (Result, error)
}

// Interceptor 定义工具执行前后的拦截扩展点。
type Interceptor interface {
	// Before 在工具执行前触发，可用于校验、审计或阻断调用。
	Before(ctx context.Context, invocation *Invocation) error
	// After 在工具执行后触发，可用于记录结果或补充副作用。
	After(ctx context.Context, invocation Invocation, result *Result, err error)
}
