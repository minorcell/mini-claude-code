package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestNormalizeProviderTypeAliases(t *testing.T) {
	cfg := Default()
	cfg.Provider.Type = "openai_compatible"
	cfg.normalize()

	if cfg.Provider.Type != "openai-compatible" {
		t.Fatalf("expected normalized provider type %q, got %q", "openai-compatible", cfg.Provider.Type)
	}
}

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

func TestLoadProviderStructFromYAML(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`
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
