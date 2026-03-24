package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultProviderName = "openai"
	defaultProviderType = "openai"
	defaultSystemPrompt = `You are mini-claude-code, a concise software engineering assistant.

Use tools when they materially improve correctness. Prefer reading existing files before editing them. Keep answers direct and grounded in tool output.

When using tools:
- Explain the next concrete action briefly.
- Avoid unnecessary repetition.
- If a tool fails, inspect the error and adapt.`
)

type Config struct {
	Provider     ProviderConfig `yaml:"provider"`
	SystemPrompt string         `yaml:"system_prompt"`
	MaxTokens    int            `yaml:"max_tokens"`
	Temperature  float64        `yaml:"temperature"`
	Workspace    string         `yaml:"workspace"`
}

type ProviderConfig struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	URL       string `yaml:"url"`
	EnvAPIKey string `yaml:"env_api_key"`
	ModelID   string `yaml:"model_id"`
}

func Default() Config {
	return Config{
		Provider: ProviderConfig{
			Name:      "openai",
			Type:      defaultProviderType,
			URL:       "https://api.openai.com/v1",
			EnvAPIKey: "OPENAI_API_KEY",
			ModelID:   "gpt-4.1-mini",
		},
		SystemPrompt: defaultSystemPrompt,
		MaxTokens:    1024,
		Temperature:  0.2,
	}
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, ".mini-claude-code", "config.yaml"), nil
}

func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Config{}, err
		}
	}

	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg.applyEnv()
			cfg.normalize()
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.applyEnv()
	cfg.normalize()
	return cfg, nil
}

func (c Config) EffectiveModel() string {
	return strings.TrimSpace(c.Provider.ModelID)
}

func (c Config) EffectiveProviderType() string {
	return normalizeProviderType(c.Provider.Type)
}

func (c Config) ProviderAPIKey() string {
	envName := strings.TrimSpace(c.Provider.EnvAPIKey)
	if envName == "" {
		return ""
	}
	return strings.TrimSpace(os.Getenv(envName))
}

func (c Config) EffectiveWorkspace(cwd string) (string, error) {
	workspace := strings.TrimSpace(c.Workspace)
	if workspace == "" {
		workspace = cwd
	}

	expanded, err := expandHome(workspace)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolve workspace %q: %w", workspace, err)
	}
	return abs, nil
}

func (c *Config) applyEnv() {}

func (c *Config) normalize() {
	if strings.TrimSpace(c.Provider.Name) == "" {
		c.Provider.Name = defaultProviderName
	}
	c.Provider.Name = strings.TrimSpace(c.Provider.Name)
	c.Provider.Type = normalizeProviderType(c.Provider.Type)
	c.SystemPrompt = os.ExpandEnv(c.SystemPrompt)
	c.Workspace = os.ExpandEnv(strings.TrimSpace(c.Workspace))
	c.Provider.URL = normalizeBaseURL(os.ExpandEnv(c.Provider.URL), defaultProviderURL(c.Provider.Type))
	c.Provider.EnvAPIKey = strings.TrimSpace(c.Provider.EnvAPIKey)
	c.Provider.ModelID = os.ExpandEnv(strings.TrimSpace(c.Provider.ModelID))

	if c.MaxTokens <= 0 {
		c.MaxTokens = Default().MaxTokens
	}

	if strings.TrimSpace(c.SystemPrompt) == "" {
		c.SystemPrompt = defaultSystemPrompt
	}

	if strings.TrimSpace(c.Provider.EnvAPIKey) == "" {
		c.Provider.EnvAPIKey = defaultEnvAPIKey(c.Provider.Type)
	}
	if strings.TrimSpace(c.Provider.ModelID) == "" {
		c.Provider.ModelID = defaultModelID(c.Provider.Type)
	}
}

func normalizeBaseURL(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func normalizeProviderType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "openai":
		return defaultProviderType
	case "openai-compatible", "openai_compatible", "openaicompatible", "compatible":
		return "openai-compatible"
	case "anthropic":
		return "anthropic"
	case "gemini":
		return "gemini"
	default:
		return normalized
	}
}

func defaultProviderURL(providerType string) string {
	switch normalizeProviderType(providerType) {
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta"
	case "openai-compatible":
		return ""
	default:
		return "https://api.openai.com/v1"
	}
}

func defaultEnvAPIKey(providerType string) string {
	switch normalizeProviderType(providerType) {
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "openai-compatible":
		return ""
	default:
		return "OPENAI_API_KEY"
	}
}

func defaultModelID(providerType string) string {
	switch normalizeProviderType(providerType) {
	case "anthropic":
		return "claude-3-7-sonnet-latest"
	case "gemini":
		return "gemini-2.0-flash"
	case "openai-compatible":
		return ""
	default:
		return "gpt-4.1-mini"
	}
}
