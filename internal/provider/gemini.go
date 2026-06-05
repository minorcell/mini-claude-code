package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type GeminiConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type GeminiClient struct {
	apiKey     string
	baseURL    string
	defaultMod string
}

func NewGeminiClient(cfg GeminiConfig) (*GeminiClient, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "gemini-2.0-flash"
	}

	return &GeminiClient{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		defaultMod: cfg.Model,
	}, nil
}

func (c *GeminiClient) Name() string {
	return "gemini"
}

func (c *GeminiClient) Complete(ctx context.Context, req Request) (Response, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.defaultMod
	}

	system, contents, err := toGeminiMessages(req.Messages)
	if err != nil {
		return Response{}, err
	}

	payload := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"temperature":     req.Temperature,
			"maxOutputTokens": req.MaxTokens,
		},
	}
	if system != "" {
		payload["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": system}},
		}
	}
	if len(req.Tools) > 0 {
		payload["tools"] = []map[string]any{
			{
				"functionDeclarations": toGeminiTools(req.Tools),
			},
		}
	}

	var response geminiResponse
	err = postJSON(ctx, defaultHTTPClient(), c.baseURL+"/models/"+model+":generateContent?key="+c.apiKey, nil, payload, &response)
	if err != nil {
		return Response{}, err
	}

	if len(response.Candidates) == 0 {
		return Response{}, fmt.Errorf("gemini returned no candidates")
	}

	parts := response.Candidates[0].Content.Parts
	var textParts []string
	var toolCalls []ToolCall
	for index, part := range parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
		if part.FunctionCall != nil {
			args, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return Response{}, fmt.Errorf("encode gemini function arguments for %q: %w", part.FunctionCall.Name, err)
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        fmt.Sprintf("gemini-call-%d", index+1),
				Name:      part.FunctionCall.Name,
				Arguments: args,
			})
		}
	}

	stopReason := StopReasonEndTurn
	if len(toolCalls) > 0 {
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
			InputTokens:  response.UsageMetadata.PromptTokenCount,
			OutputTokens: response.UsageMetadata.CandidatesTokenCount,
		},
	}, nil
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text         string `json:"text,omitempty"`
				FunctionCall *struct {
					Name string         `json:"name"`
					Args map[string]any `json:"args"`
				} `json:"functionCall,omitempty"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func toGeminiMessages(messages []Message) (string, []map[string]any, error) {
	var systemParts []string
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
				"parts": []map[string]any{
					{"text": message.Content},
				},
			})
		case RoleAssistant:
			parts := make([]map[string]any, 0, len(message.ToolCalls)+1)
			if message.Content != "" {
				parts = append(parts, map[string]any{"text": message.Content})
			}
			for _, call := range message.ToolCalls {
				var args map[string]any
				if len(call.Arguments) > 0 {
					if err := json.Unmarshal(call.Arguments, &args); err != nil {
						return "", nil, fmt.Errorf("decode gemini tool arguments for %q: %w", call.Name, err)
					}
				}
				parts = append(parts, map[string]any{
					"functionCall": map[string]any{
						"name": call.Name,
						"args": args,
					},
				})
			}
			if len(parts) > 0 {
				converted = append(converted, map[string]any{
					"role":  "model",
					"parts": parts,
				})
			}
		case RoleTool:
			converted = append(converted, map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{
						"functionResponse": map[string]any{
							"name": message.Name,
							"response": map[string]any{
								"name":    message.Name,
								"content": message.Content,
							},
						},
					},
				},
			})
		}
	}

	return strings.Join(systemParts, "\n\n"), converted, nil
}

func toGeminiTools(tools []ToolDefinition) []map[string]any {
	result := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"parameters":  tool.InputSchema,
		})
	}
	return result
}
