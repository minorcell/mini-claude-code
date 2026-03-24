package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/config"
	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/core"
	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/provider"
	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/tools"
	"github.com/minorcell/mini-claude-code/projects/mini-claude-code-go-whit-library/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath, err := config.DefaultPath()
	if err != nil {
		return err
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	workspace, err := cfg.EffectiveWorkspace(cwd)
	if err != nil {
		return err
	}

	client, modelName, err := buildClient(cfg)
	if err != nil {
		return err
	}

	registry := tools.DefaultRegistry(workspace)
	agent := core.NewAgent(client, registry, core.AgentConfig{
		Model:       modelName,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		MaxSteps:    8,
		WorkingDir:  workspace,
	})
	session := core.NewSession(cfg.SystemPrompt)

	return tui.Run(tui.App{
		Agent:        agent,
		Session:      session,
		ConfigPath:   configPath,
		ProviderName: cfg.Provider.Name,
		ProviderType: cfg.EffectiveProviderType(),
		ModelName:    modelName,
		Workspace:    workspace,
	})
}

func buildClient(cfg config.Config) (provider.Client, string, error) {
	providerType := cfg.EffectiveProviderType()
	modelName := cfg.EffectiveModel()
	apiKey := cfg.ProviderAPIKey()
	baseURL := cfg.Provider.URL

	switch providerType {
	case "openai", "openai-compatible":
		if providerType == "openai-compatible" && strings.TrimSpace(baseURL) == "" {
			return nil, "", fmt.Errorf("provider.url is required for provider.type=%q", providerType)
		}
		client, err := provider.NewOpenAIClient(provider.OpenAIConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   modelName,
		})
		return client, modelName, err
	case "anthropic":
		client, err := provider.NewAnthropicClient(provider.AnthropicConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   modelName,
		})
		return client, modelName, err
	case "gemini":
		client, err := provider.NewGeminiClient(provider.GeminiConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   modelName,
		})
		return client, modelName, err
	default:
		return nil, "", fmt.Errorf("unsupported provider.type %q", cfg.Provider.Type)
	}
}
