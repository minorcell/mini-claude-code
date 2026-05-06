package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type AnthropicConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type AnthropicClient struct {
	apiKey     string
	baseURL    string
	defaultMod string
}

func NewAnthropicClient(cfg AnthropicConfig) (*AnthropicClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.anthropic.com/v1"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "claude-3-7-sonnet-latest"
	}

	return &AnthropicClient{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		defaultMod: cfg.Model,
	}, nil
}

func (c *AnthropicClient) Name() string {
	return "anthropic"
}

func (c *AnthropicClient) Complete(ctx context.Context, req Request) (Response, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.defaultMod
	}

	system, messages, err := toAnthropicMessages(req.Messages)
	if err != nil {
		return Response{}, err
	}

	payload := map[string]any{
		"model":       model,
		"system":      system,
		"messages":    messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}
	if len(req.Tools) > 0 {
		payload["tools"] = toAnthropicTools(req.Tools)
	}

	var response anthropicResponse
	err = postJSON(ctx, defaultHTTPClient(), c.baseURL+"/messages", map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	}, payload, &response)
	if err != nil {
		return Response{}, err
	}

	var textParts []string
	toolCalls := make([]ToolCall, 0)
	for _, block := range response.Content {
		switch block.Type {
		case "text":
			if block.Text != "" {
				textParts = append(textParts, block.Text)
			}
		case "tool_use":
			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
	}

	stopReason := StopReasonEndTurn
	if response.StopReason == "tool_use" {
		stopReason = StopReasonToolUse
	}

	return Response{
		Message: Message{
			Role:      RoleAssistant,
			Content:   strings.Join(textParts, "\n"),
			ToolCalls: toolCalls,
		},
		StopReason: stopReason,
		Usage: Usage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
		},
	}, nil
}

type anthropicResponse struct {
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func toAnthropicMessages(messages []Message) (string, []map[string]any, error) {
	systemParts := make([]string, 0)
	converted := make([]map[string]any, 0, len(messages))

	for _, message := range messages {
		switch message.Role {
		case RoleSystem:
			if message.Content != "" {
				systemParts = append(systemParts, message.Content)
			}
		case RoleUser:
			converted = append(converted, map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": message.Content,
					},
				},
			})
		case RoleAssistant:
			content := make([]map[string]any, 0, len(message.ToolCalls)+1)
			if message.Content != "" {
				content = append(content, map[string]any{
					"type": "text",
					"text": message.Content,
				})
			}

			for _, call := range message.ToolCalls {
				var input any
				if len(call.Arguments) > 0 {
					if err := json.Unmarshal(call.Arguments, &input); err != nil {
						return "", nil, fmt.Errorf("decode anthropic tool arguments for %q: %w", call.Name, err)
					}
				}
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    call.ID,
					"name":  call.Name,
					"input": input,
				})
			}

			if len(content) > 0 {
				converted = append(converted, map[string]any{
					"role":    "assistant",
					"content": content,
				})
			}
		case RoleTool:
			converted = append(converted, map[string]any{
				"role": "user",
				"content": []map[string]any{
					{
						"type":        "tool_result",
						"tool_use_id": message.ToolCallID,
						"content":     message.Content,
					},
				},
			})
		}
	}

	return strings.Join(systemParts, "\n\n"), converted, nil
}

func toAnthropicTools(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.InputSchema,
		})
	}
	return result
}
