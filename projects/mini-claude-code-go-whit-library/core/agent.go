package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/tools"
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
		cfg.MaxSteps = 8
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
	if session == nil {
		return TurnResult{}, fmt.Errorf("session is required")
	}

	session.Add(provider.Message{
		Role:    provider.RoleUser,
		Content: userInput,
	})

	result := TurnResult{}
	for step := 0; step < a.config.MaxSteps; step++ {
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

		session.Add(response.Message)
		if text := strings.TrimSpace(response.Message.Content); text != "" {
			result.Events = append(result.Events, Event{
				Kind:    EventAssistant,
				Content: text,
			})
		}

		if len(response.Message.ToolCalls) == 0 {
			return result, nil
		}

		for _, call := range response.Message.ToolCalls {
			execution, err := a.registry.Execute(ctx, call, tools.State{
				WorkingDir: a.config.WorkingDir,
				SessionID:  session.ID,
			})

			toolMessage := provider.Message{
				Role:       provider.RoleTool,
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    execution.Render(),
			}
			session.Add(toolMessage)

			result.Events = append(result.Events, Event{
				Kind:       EventTool,
				ToolName:   call.Name,
				ToolInput:  stringifyArguments(call.Arguments),
				ToolOutput: execution.Render(),
				IsError:    err != nil,
			})
		}
	}

	return result, fmt.Errorf("agent exceeded max steps (%d)", a.config.MaxSteps)
}

func cloneMessages(messages []provider.Message) []provider.Message {
	cloned := make([]provider.Message, len(messages))
	copy(cloned, messages)
	return cloned
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
