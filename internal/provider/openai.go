package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type OpenAIClient struct {
	apiKey     string
	baseURL    string
	defaultMod string
}

func NewOpenAIClient(cfg OpenAIConfig) (*OpenAIClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("openai api key is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "gpt-4.1-mini"
	}

	return &OpenAIClient{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		defaultMod: cfg.Model,
	}, nil
}

func (c *OpenAIClient) Name() string {
	return "openai"
}

func (c *OpenAIClient) Complete(ctx context.Context, req Request) (Response, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.defaultMod
	}

	payload := map[string]any{
		"model":       model,
		"messages":    toOpenAIMessages(req.Messages),
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}
	if len(req.Tools) > 0 {
		payload["tools"] = toOpenAITools(req.Tools)
	}

	var response openAIResponse
	err := postJSON(ctx, defaultHTTPClient(), c.baseURL+"/chat/completions", map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}, payload, &response)
	if err != nil {
		return Response{}, err
	}

	if len(response.Choices) == 0 {
		return Response{}, fmt.Errorf("openai returned no choices")
	}

	choice := response.Choices[0]
	toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))
	for _, item := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        item.ID,
			Name:      item.Function.Name,
			Arguments: json.RawMessage(item.Function.Arguments),
		})
	}

	stopReason := StopReasonEndTurn
	if choice.FinishReason == "tool_calls" {
		stopReason = StopReasonToolUse
	}

	return Response{
		Message: Message{
			Role:      RoleAssistant,
			Content:   choice.Message.Content,
			ToolCalls: toolCalls,
		},
		StopReason: stopReason,
		Usage: Usage{
			InputTokens:  response.Usage.PromptTokens,
			OutputTokens: response.Usage.CompletionTokens,
		},
	}, nil
}

type openAIResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func toOpenAIMessages(messages []Message) []map[string]any {
	converted := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		item := map[string]any{
			"role": string(message.Role),
		}

		switch message.Role {
		case RoleAssistant:
			item["content"] = message.Content
			if len(message.ToolCalls) > 0 {
				toolCalls := make([]map[string]any, 0, len(message.ToolCalls))
				for _, call := range message.ToolCalls {
					toolCalls = append(toolCalls, map[string]any{
						"id":   call.ID,
						"type": "function",
						"function": map[string]any{
							"name":      call.Name,
							"arguments": string(call.Arguments),
						},
					})
				}
				item["tool_calls"] = toolCalls
			}
		case RoleTool:
			item["content"] = message.Content
			item["tool_call_id"] = message.ToolCallID
			if message.Name != "" {
				item["name"] = message.Name
			}
		default:
			item["content"] = message.Content
		}

		converted = append(converted, item)
	}
	return converted
}

func toOpenAITools(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			},
		})
	}
	return result
}
