package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadUsesProviderEnvAPIKey 验证 env_api_key 能正确解析真实 API Key。
func TestLoadUsesProviderEnvAPIKey(t *testing.T) {
	t.Setenv("MY_GATEWAY_KEY", "secret-value")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	cfg.Provider.EnvAPIKey = "MY_GATEWAY_KEY"
	if got := cfg.ProviderAPIKey(); got != "secret-value" {
		t.Fatalf("expected api key %q, got %q", "secret-value", got)
	}
}

// TestNormalizeProviderTypeAliases 验证 provider.type 的别名会被统一折叠。
func TestNormalizeProviderTypeAliases(t *testing.T) {
	cfg := Default()
	cfg.Provider.Type = "openai_compatible"
	cfg.normalize()

	if cfg.Provider.Type != "openai-compatible" {
		t.Fatalf("expected normalized provider type %q, got %q", "openai-compatible", cfg.Provider.Type)
	}
}

// TestEffectiveWorkspaceExpandsHome 验证工作区路径支持波浪号展开。
func TestEffectiveWorkspaceExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}

	cfg := Default()
	cfg.Workspace = "~/workspace"

	workspace, err := cfg.EffectiveWorkspace("/tmp")
	if err != nil {
		t.Fatalf("EffectiveWorkspace() error = %v", err)
	}

	expected := filepath.Join(home, "workspace")
	if workspace != expected {
		t.Fatalf("expected %q, got %q", expected, workspace)
	}
}

// TestLoadProviderStructFromYAML 验证新的 provider 结构可以从 YAML 正常加载。
func TestLoadProviderStructFromYAML(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`
max_steps: 40
provider:
  name: DeepSeek
  type: openai-compatible
  url: https://gateway.example.com/v1
  env_api_key: GATEWAY_KEY
  model_id: deepseek-chat
`)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Provider.Name != "DeepSeek" {
		t.Fatalf("expected provider name DeepSeek, got %q", cfg.Provider.Name)
	}
	if cfg.MaxSteps != 40 {
		t.Fatalf("expected max_steps 40, got %d", cfg.MaxSteps)
	}
	if cfg.Provider.Type != "openai-compatible" {
		t.Fatalf("expected provider type openai-compatible, got %q", cfg.Provider.Type)
	}
	if cfg.Provider.URL != "https://gateway.example.com/v1" {
		t.Fatalf("unexpected provider url %q", cfg.Provider.URL)
	}
	if cfg.Provider.EnvAPIKey != "GATEWAY_KEY" {
		t.Fatalf("unexpected env_api_key %q", cfg.Provider.EnvAPIKey)
	}
	if cfg.Provider.ModelID != "deepseek-chat" {
		t.Fatalf("unexpected model_id %q", cfg.Provider.ModelID)
	}
}

// TestNormalizeMaxStepsFallsBackToDefault 验证非法 max_steps 会回退到默认值。
func TestNormalizeMaxStepsFallsBackToDefault(t *testing.T) {
	cfg := Default()
	cfg.MaxSteps = 0
	cfg.normalize()

	if cfg.MaxSteps != Default().MaxSteps {
		t.Fatalf("expected default max_steps %d, got %d", Default().MaxSteps, cfg.MaxSteps)
	}
}
