package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
)

type Registry struct {
	tools        map[string]Tool
	interceptors []Interceptor
}

func NewRegistry(interceptors ...Interceptor) *Registry {
	return &Registry{
		tools:        make(map[string]Tool),
		interceptors: interceptors,
	}
}

func (r *Registry) Register(tool Tool) error {
	definition := tool.Definition()
	if definition.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if _, exists := r.tools[definition.Name]; exists {
		return fmt.Errorf("tool %q already registered", definition.Name)
	}
	r.tools[definition.Name] = tool
	return nil
}

func (r *Registry) MustRegister(tool Tool) {
	if err := r.Register(tool); err != nil {
		panic(err)
	}
}

func (r *Registry) Definitions() []provider.ToolDefinition {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	definitions := make([]provider.ToolDefinition, 0, len(names))
	for _, name := range names {
		definitions = append(definitions, r.tools[name].Definition())
	}
	return definitions
}

func (r *Registry) Execute(ctx context.Context, call provider.ToolCall, state State) (Result, error) {
	tool, ok := r.tools[call.Name]
	if !ok {
		result := Result{
			ToolName: call.Name,
			OK:       false,
			Output:   fmt.Sprintf("unknown tool %q", call.Name),
		}
		return result, fmt.Errorf("unknown tool %q", call.Name)
	}

	invocation := Invocation{
		ToolName:   call.Name,
		CallID:     call.ID,
		WorkingDir: state.WorkingDir,
		SessionID:  state.SessionID,
		Arguments:  call.Arguments,
	}
	if len(call.Arguments) > 0 {
		_ = json.Unmarshal(call.Arguments, &invocation.ParsedArgs)
	}

	for _, interceptor := range r.interceptors {
		if err := interceptor.Before(ctx, &invocation); err != nil {
			result := Result{
				ToolName: call.Name,
				OK:       false,
				Output:   err.Error(),
			}
			for i := len(r.interceptors) - 1; i >= 0; i-- {
				r.interceptors[i].After(ctx, invocation, &result, err)
			}
			return result, err
		}
	}

	result, err := tool.Execute(ctx, invocation)
	result.ToolName = call.Name
	result.OK = err == nil
	if err != nil && result.Output == "" {
		result.Output = err.Error()
	}

	for i := len(r.interceptors) - 1; i >= 0; i-- {
		r.interceptors[i].After(ctx, invocation, &result, err)
	}

	return result, err
}
