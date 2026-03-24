package core

import (
	"context"
	"testing"

	"github.com/minorcell/mini-claude-code/projects/mini-opencode-go/provider"
	"github.com/minorcell/mini-claude-code/projects/mini-opencode-go/tools"
)

type mockProvider struct {
	requests  []provider.Request
	responses []provider.Response
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Complete(_ context.Context, req provider.Request) (provider.Response, error) {
	m.requests = append(m.requests, req)
	if len(m.responses) == 0 {
		return provider.Response{}, nil
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return response, nil
}

type fakeTool struct{}

func (fakeTool) Definition() provider.ToolDefinition {
	return provider.ToolDefinition{
		Name:        "echo",
		Description: "Echo test tool",
		InputSchema: map[string]any{
			"type": "object",
		},
	}
}

func (fakeTool) Execute(_ context.Context, _ tools.Invocation) (tools.Result, error) {
	return tools.Result{
		Output: "pong",
	}, nil
}

// TestRunTurnExecutesToolLoop 验证 agent 能完成一轮工具调用闭环。
func TestRunTurnExecutesToolLoop(t *testing.T) {
	registry := tools.NewRegistry()
	registry.MustRegister(fakeTool{})

	client := &mockProvider{
		responses: []provider.Response{
			{
				Message: provider.Message{
					Role: provider.RoleAssistant,
					ToolCalls: []provider.ToolCall{
						{
							ID:        "call-1",
							Name:      "echo",
							Arguments: []byte(`{"value":"ping"}`),
						},
					},
				},
				StopReason: provider.StopReasonToolUse,
				Usage: provider.Usage{
					InputTokens:  11,
					OutputTokens: 7,
				},
			},
			{
				Message: provider.Message{
					Role:    provider.RoleAssistant,
					Content: "done",
				},
				StopReason: provider.StopReasonEndTurn,
				Usage: provider.Usage{
					InputTokens:  13,
					OutputTokens: 5,
				},
			},
		},
	}

	agent := NewAgent(client, registry, AgentConfig{
		Model:      "test-model",
		WorkingDir: t.TempDir(),
		MaxSteps:   4,
	})

	session := NewSession("system")
	result, err := agent.RunTurn(context.Background(), session, "hello")
	if err != nil {
		t.Fatalf("RunTurn() error = %v", err)
	}

	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result.Events))
	}
	if result.Events[0].Kind != EventTool {
		t.Fatalf("expected first event to be tool, got %q", result.Events[0].Kind)
	}
	if result.Events[1].Content != "done" {
		t.Fatalf("expected final assistant content %q, got %q", "done", result.Events[1].Content)
	}
	if len(client.requests) != 2 {
		t.Fatalf("expected 2 provider requests, got %d", len(client.requests))
	}
	if len(client.requests[0].Tools) != 1 {
		t.Fatalf("expected tool definition in first request")
	}
	if got := session.Messages[len(session.Messages)-1].Content; got != "done" {
		t.Fatalf("expected final session message %q, got %q", "done", got)
	}
}

// TestRunTurnWithObserverEmitsProgress 验证增量观察者会收到逐步执行事件。
func TestRunTurnWithObserverEmitsProgress(t *testing.T) {
	registry := tools.NewRegistry()
	registry.MustRegister(fakeTool{})

	client := &mockProvider{
		responses: []provider.Response{
			{
				Message: provider.Message{
					Role: provider.RoleAssistant,
					ToolCalls: []provider.ToolCall{
						{
							ID:        "call-1",
							Name:      "echo",
							Arguments: []byte(`{"value":"ping"}`),
						},
					},
				},
				StopReason: provider.StopReasonToolUse,
				Usage: provider.Usage{
					InputTokens:  11,
					OutputTokens: 7,
				},
			},
			{
				Message: provider.Message{
					Role:    provider.RoleAssistant,
					Content: "done",
				},
				StopReason: provider.StopReasonEndTurn,
				Usage: provider.Usage{
					InputTokens:  13,
					OutputTokens: 5,
				},
			},
		},
	}

	agent := NewAgent(client, registry, AgentConfig{
		Model:      "test-model",
		WorkingDir: t.TempDir(),
		MaxSteps:   4,
	})

	var kinds []ProgressEventKind
	var stepCompleted []ProgressEvent
	_, err := agent.RunTurnWithObserver(context.Background(), NewSession("system"), "hello", func(event ProgressEvent) {
		kinds = append(kinds, event.Kind)
		if event.Kind == ProgressEventStepCompleted {
			stepCompleted = append(stepCompleted, event)
		}
	})
	if err != nil {
		t.Fatalf("RunTurnWithObserver() error = %v", err)
	}

	expected := []ProgressEventKind{
		ProgressEventStepStarted,
		ProgressEventStepCompleted,
		ProgressEventToolStarted,
		ProgressEventToolFinished,
		ProgressEventStepStarted,
		ProgressEventStepCompleted,
		ProgressEventAssistantMessage,
	}
	if len(kinds) != len(expected) {
		t.Fatalf("expected %d progress events, got %d", len(expected), len(kinds))
	}
	for i := range expected {
		if kinds[i] != expected[i] {
			t.Fatalf("expected event %d to be %q, got %q", i, expected[i], kinds[i])
		}
	}
	if len(stepCompleted) != 2 {
		t.Fatalf("expected 2 step-completed events, got %d", len(stepCompleted))
	}
	if stepCompleted[0].Usage.InputTokens != 11 || stepCompleted[0].Usage.OutputTokens != 7 {
		t.Fatalf("unexpected step 1 usage: %+v", stepCompleted[0].Usage)
	}
	if stepCompleted[0].TotalUsage.InputTokens != 11 || stepCompleted[0].TotalUsage.OutputTokens != 7 {
		t.Fatalf("unexpected step 1 total usage: %+v", stepCompleted[0].TotalUsage)
	}
	if stepCompleted[1].Usage.InputTokens != 13 || stepCompleted[1].Usage.OutputTokens != 5 {
		t.Fatalf("unexpected step 2 usage: %+v", stepCompleted[1].Usage)
	}
	if stepCompleted[1].TotalUsage.InputTokens != 24 || stepCompleted[1].TotalUsage.OutputTokens != 12 {
		t.Fatalf("unexpected step 2 total usage: %+v", stepCompleted[1].TotalUsage)
	}
}
