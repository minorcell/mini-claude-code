package core

import (
	"context"
	"testing"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/tools"
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
			},
			{
				Message: provider.Message{
					Role:    provider.RoleAssistant,
					Content: "done",
				},
				StopReason: provider.StopReasonEndTurn,
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
