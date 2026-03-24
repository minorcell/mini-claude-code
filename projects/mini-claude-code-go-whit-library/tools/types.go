package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

type State struct {
	WorkingDir string
	SessionID  string
}

type Invocation struct {
	ToolName   string
	CallID     string
	WorkingDir string
	SessionID  string
	Arguments  json.RawMessage
	ParsedArgs map[string]any
}

type Result struct {
	ToolName string         `json:"tool_name"`
	OK       bool           `json:"ok"`
	Output   string         `json:"output"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (r Result) Render() string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"output":"%s"}`, err.Error())
	}
	return string(data)
}

type Tool interface {
	Definition() provider.ToolDefinition
	Execute(ctx context.Context, invocation Invocation) (Result, error)
}

type Interceptor interface {
	Before(ctx context.Context, invocation *Invocation) error
	After(ctx context.Context, invocation Invocation, result *Result, err error)
}
