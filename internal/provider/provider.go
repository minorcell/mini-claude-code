// Package provider 定义统一模型接口及不同协议适配器共享的数据结构。
package provider

import (
	"context"
	"encoding/json"
)

// Role 表示统一消息模型中的角色类型。
type Role string

const (
	// RoleSystem 表示系统提示消息。
	RoleSystem Role = "system"
	// RoleUser 表示用户输入消息。
	RoleUser Role = "user"
	// RoleAssistant 表示模型生成的助手消息。
	RoleAssistant Role = "assistant"
	// RoleTool 表示工具执行结果消息。
	RoleTool Role = "tool"
)

// StopReason 表示一次模型响应结束的原因。
type StopReason string

const (
	// StopReasonEndTurn 表示本轮输出自然结束。
	StopReasonEndTurn StopReason = "end_turn"
	// StopReasonToolUse 表示模型请求先执行工具再继续推理。
	StopReasonToolUse StopReason = "tool_use"
)

// Message 是跨 provider 复用的统一消息结构。
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall 描述模型发出的单次工具调用请求。
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolDefinition 描述暴露给模型使用的工具定义。
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// Request 表示发送给 provider 的统一请求。
type Request struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

// Response 表示 provider 返回的统一响应。
type Response struct {
	Message    Message    `json:"message"`
	StopReason StopReason `json:"stop_reason"`
	Usage      Usage      `json:"usage"`
}

// Usage 记录本次请求的 token 使用情况。
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Client 定义统一的模型调用接口。
type Client interface {
	// Name 返回当前客户端对应的 provider 类型标识。
	Name() string
	// Complete 发起一次统一请求，并返回归一化后的响应结果。
	Complete(ctx context.Context, req Request) (Response, error)
}
