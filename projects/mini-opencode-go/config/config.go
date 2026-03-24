// Package config 负责加载、规范化并提供运行配置。
package config

import (
	_ "embed"
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
)

//go:embed prompt.md
var promptContent string

// SystemPrompt 返回内置的系统提示词。
func SystemPrompt() string {
	return promptContent
}

// Config 描述 mini-claude-code 的运行配置。
type Config struct {
	Provider    ProviderConfig `yaml:"provider"`
	MaxTokens   int            `yaml:"max_tokens"`
	MaxSteps    int            `yaml:"max_steps"`
	Temperature float64        `yaml:"temperature"`
	Workspace   string         `yaml:"workspace"`
}

// ProviderConfig 描述单个模型接入点。
//
// Name 用于展示接入名称，Type 用于选择底层 API 协议适配器。
type ProviderConfig struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	URL       string `yaml:"url"`
	EnvAPIKey string `yaml:"env_api_key"`
	ModelID   string `yaml:"model_id"`
}

// Default 返回可直接启动程序的默认配置。
func Default() Config {
	return Config{
		Provider: ProviderConfig{
			Name:      "openai",
			Type:      defaultProviderType,
			URL:       "https://api.openai.com/v1",
			EnvAPIKey: "OPENAI_API_KEY",
			ModelID:   "gpt-4.1-mini",
		},
		MaxTokens:   1024,
		MaxSteps:    24,
		Temperature: 0.2,
	}
}

// DefaultPath 返回默认配置文件路径。
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, ".mini-claude-code", "config.yaml"), nil
}

// Load 从指定路径加载配置；当文件不存在时会回退到默认配置。
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
			if err := initConfigFile(path, &cfg); err != nil {
				return Config{}, err
			}
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

// EffectiveModel 返回当前生效的模型 ID。
func (c Config) EffectiveModel() string {
	return strings.TrimSpace(c.Provider.ModelID)
}

// EffectiveProviderType 返回规范化后的 provider 类型。
func (c Config) EffectiveProviderType() string {
	return normalizeProviderType(c.Provider.Type)
}

// ProviderAPIKey 根据 env_api_key 指向的环境变量读取真实 API Key。
func (c Config) ProviderAPIKey() string {
	envName := strings.TrimSpace(c.Provider.EnvAPIKey)
	if envName == "" {
		return ""
	}
	return strings.TrimSpace(os.Getenv(envName))
}

// EffectiveWorkspace 返回当前生效的工作区绝对路径。
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

// initConfigFile 在配置文件不存在时创建目录并写入默认配置。
func initConfigFile(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}

	return nil
}

// applyEnv 预留给环境变量合并逻辑扩展，目前默认配置已足够覆盖启动路径。
func (c *Config) applyEnv() {}

func (c *Config) normalize() {
	if strings.TrimSpace(c.Provider.Name) == "" {
		c.Provider.Name = defaultProviderName
	}
	c.Provider.Name = strings.TrimSpace(c.Provider.Name)
	c.Provider.Type = normalizeProviderType(c.Provider.Type)
	c.Workspace = os.ExpandEnv(strings.TrimSpace(c.Workspace))
	c.Provider.URL = normalizeBaseURL(os.ExpandEnv(c.Provider.URL), defaultProviderURL(c.Provider.Type))
	c.Provider.EnvAPIKey = strings.TrimSpace(c.Provider.EnvAPIKey)
	c.Provider.ModelID = os.ExpandEnv(strings.TrimSpace(c.Provider.ModelID))

	if c.MaxTokens <= 0 {
		c.MaxTokens = Default().MaxTokens
	}
	if c.MaxSteps <= 0 {
		c.MaxSteps = Default().MaxSteps
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

// normalizeProviderType 将用户输入的 provider.type 统一折叠为内部标准值。
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

// defaultProviderURL 返回不同 provider 类型的默认 API 地址。
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

// defaultEnvAPIKey 返回不同 provider 类型推荐使用的默认环境变量名。
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

// defaultModelID 返回不同 provider 类型的默认模型 ID。
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
