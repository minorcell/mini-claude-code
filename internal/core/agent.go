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

type AgentConfig struct {
	Model       string
	MaxTokens   int
	Temperature float64
	MaxSteps    int
	WorkingDir  string
}

type EventKind string

const (
	EventAssistant EventKind = "assistant"
	EventTool      EventKind = "tool"
)

type Event struct {
	Kind       EventKind
	Content    string
	ToolName   string
	ToolInput  string
	ToolOutput string
	IsError    bool
}

type ProgressEventKind string

const (
	ProgressEventStepStarted      ProgressEventKind = "step_started"
	ProgressEventStepCompleted    ProgressEventKind = "step_completed"
	ProgressEventAssistantMessage ProgressEventKind = "assistant_message"
	ProgressEventToolStarted      ProgressEventKind = "tool_started"
	ProgressEventToolFinished     ProgressEventKind = "tool_finished"
)

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

type TurnObserver func(ProgressEvent)

type TurnResult struct {
	Events []Event
	Usage  provider.Usage
	Steps  int
}

type Agent struct {
	client   provider.Client
	registry *tools.Registry
	config   AgentConfig
}

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

func (a *Agent) RunTurn(ctx context.Context, session *Session, userInput string) (TurnResult, error) {
	return a.runTurn(ctx, session, userInput, nil)
}

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
			Messages:    session.Messages,
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

func stringifyArguments(arguments json.RawMessage) string {
	if len(arguments) == 0 {
		return "{}"
	}

	var out bytes.Buffer
	if err := json.Indent(&out, arguments, "", "  "); err == nil {
		return out.String()
	}
	return string(arguments)
}

func emitProgress(observer TurnObserver, event ProgressEvent) {
	if observer == nil {
		return
	}
	observer(event)
}
