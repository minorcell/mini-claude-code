// Package core 实现会话管理与 agent 执行闭环。
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/minorcell/mini-claude-code/internal/provider"
	"github.com/minorcell/mini-claude-code/internal/tools"
)

// AgentConfig 定义 agent 运行时使用的核心参数。
type AgentConfig struct {
	Model       string
	MaxTokens   int
	Temperature float64
	MaxSteps    int
	WorkingDir  string
}

// EventKind 描述一次 turn 中事件的类别。
type EventKind string

const (
	// EventAssistant 表示模型生成的助手文本。
	EventAssistant EventKind = "assistant"
	// EventTool 表示模型触发并执行了一次工具调用。
	EventTool EventKind = "tool"
)

// Event 记录一次 turn 中可供上层界面消费的事件。
type Event struct {
	Kind       EventKind
	Content    string
	ToolName   string
	ToolInput  string
	ToolOutput string
	IsError    bool
}

// ProgressEventKind 描述一次 turn 的增量执行事件。
type ProgressEventKind string

const (
	// ProgressEventStepStarted 表示 agent 进入了新的一步。
	ProgressEventStepStarted ProgressEventKind = "step_started"
	// ProgressEventStepCompleted 表示当前步骤已经拿到模型响应和 usage。
	ProgressEventStepCompleted ProgressEventKind = "step_completed"
	// ProgressEventAssistantMessage 表示模型在当前步骤产出了可展示文本。
	ProgressEventAssistantMessage ProgressEventKind = "assistant_message"
	// ProgressEventToolStarted 表示模型开始执行一次工具调用。
	ProgressEventToolStarted ProgressEventKind = "tool_started"
	// ProgressEventToolFinished 表示一次工具调用已经执行完成。
	ProgressEventToolFinished ProgressEventKind = "tool_finished"
)

// ProgressEvent 用于向上层 UI 广播 turn 内部的逐步进展。
type ProgressEvent struct {
	Kind       ProgressEventKind
	Step       int
	MaxSteps   int
	CallID     string
	Content    string
	ToolName   string
	ToolInput  string
	ToolOutput string
	IsError    bool
	Usage      provider.Usage
	TotalUsage provider.Usage
}

// TurnObserver 在 turn 执行期间消费增量事件。
type TurnObserver func(ProgressEvent)

// TurnResult 表示一次用户输入执行完成后的聚合结果。
type TurnResult struct {
	Events []Event
	Usage  provider.Usage
	Steps  int
}

// Agent 负责驱动模型、会话与工具注册表形成完整闭环。
type Agent struct {
	client   provider.Client
	registry *tools.Registry
	config   AgentConfig
}

// NewAgent 创建一个新的 agent 实例。
func NewAgent(client provider.Client, registry *tools.Registry, cfg AgentConfig) *Agent {
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 24
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 1024
	}

	return &Agent{
		client:   client,
		registry: registry,
		config:   cfg,
	}
}

// RunTurn 执行一轮用户输入，并在需要时自动循环工具调用直到得到最终回复。
func (a *Agent) RunTurn(ctx context.Context, session *Session, userInput string) (TurnResult, error) {
	return a.runTurn(ctx, session, userInput, nil)
}

// RunTurnWithObserver 在 RunTurn 基础上额外发出逐步执行事件，适合交互界面实时展示。
func (a *Agent) RunTurnWithObserver(ctx context.Context, session *Session, userInput string, observer TurnObserver) (TurnResult, error) {
	return a.runTurn(ctx, session, userInput, observer)
}

func (a *Agent) runTurn(ctx context.Context, session *Session, userInput string, observer TurnObserver) (TurnResult, error) {
	if session == nil {
		return TurnResult{}, fmt.Errorf("session is required")
	}

	session.Add(provider.Message{
		Role:    provider.RoleUser,
		Content: userInput,
	})

	result := TurnResult{}
	for step := 0; step < a.config.MaxSteps; step++ {
		emitProgress(observer, ProgressEvent{
			Kind:     ProgressEventStepStarted,
			Step:     step + 1,
			MaxSteps: a.config.MaxSteps,
		})

		response, err := a.client.Complete(ctx, provider.Request{
			Model:       a.config.Model,
			Messages:    cloneMessages(session.Messages),
			Tools:       a.registry.Definitions(),
			MaxTokens:   a.config.MaxTokens,
			Temperature: a.config.Temperature,
		})
		if err != nil {
			return result, err
		}

		result.Usage.InputTokens += response.Usage.InputTokens
		result.Usage.OutputTokens += response.Usage.OutputTokens
		result.Steps++
		emitProgress(observer, ProgressEvent{
			Kind:       ProgressEventStepCompleted,
			Step:       step + 1,
			MaxSteps:   a.config.MaxSteps,
			Usage:      response.Usage,
			TotalUsage: result.Usage,
		})

		session.Add(response.Message)
		if text := strings.TrimSpace(response.Message.Content); text != "" {
			emitProgress(observer, ProgressEvent{
				Kind:     ProgressEventAssistantMessage,
				Step:     step + 1,
				MaxSteps: a.config.MaxSteps,
				Content:  text,
			})
			result.Events = append(result.Events, Event{
				Kind:    EventAssistant,
				Content: text,
			})
		}

		if len(response.Message.ToolCalls) == 0 {
			return result, nil
		}

		for _, call := range response.Message.ToolCalls {
			toolInput := stringifyArguments(call.Arguments)
			emitProgress(observer, ProgressEvent{
				Kind:      ProgressEventToolStarted,
				Step:      step + 1,
				MaxSteps:  a.config.MaxSteps,
				CallID:    call.ID,
				ToolName:  call.Name,
				ToolInput: toolInput,
			})

			execution, err := a.registry.Execute(ctx, call, tools.State{
				WorkingDir: a.config.WorkingDir,
				SessionID:  session.ID,
			})
			renderedOutput := execution.Render()

			toolMessage := provider.Message{
				Role:       provider.RoleTool,
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    renderedOutput,
			}
			session.Add(toolMessage)
			emitProgress(observer, ProgressEvent{
				Kind:       ProgressEventToolFinished,
				Step:       step + 1,
				MaxSteps:   a.config.MaxSteps,
				CallID:     call.ID,
				ToolName:   call.Name,
				ToolInput:  toolInput,
				ToolOutput: renderedOutput,
				IsError:    err != nil,
			})

			result.Events = append(result.Events, Event{
				Kind:       EventTool,
				ToolName:   call.Name,
				ToolInput:  toolInput,
				ToolOutput: renderedOutput,
				IsError:    err != nil,
			})
		}
	}

	return result, fmt.Errorf("agent exceeded max steps (%d)", a.config.MaxSteps)
}

// cloneMessages 复制消息切片，避免 provider 在调用链中意外修改会话历史。
func cloneMessages(messages []provider.Message) []provider.Message {
	cloned := make([]provider.Message, len(messages))
	copy(cloned, messages)
	return cloned
}

// stringifyArguments 将工具参数渲染为更适合展示的格式化 JSON。
func stringifyArguments(arguments json.RawMessage) string {
	if len(arguments) == 0 {
		return "{}"
	}

	var out bytes.Buffer
	if err := json.Indent(&out, arguments, "", "  "); err != nil {
		return string(arguments)
	}
	return out.String()
}

func emitProgress(observer TurnObserver, event ProgressEvent) {
	if observer == nil {
		return
	}
	observer(event)
}
