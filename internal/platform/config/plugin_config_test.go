package config_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/ye-kart/reqflow/internal/platform/config"
)

func TestLoad_ParsesPluginConfig(t *testing.T) {
	globalDir := t.TempDir()

	content := `plugins:
  dir: /custom/plugins
  enabled:
    - my-custom-auth
    - xml-formatter
`
	if err := os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := config.Load(
		config.WithGlobalDir(globalDir),
		config.WithProjectDir(t.TempDir()),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Plugins.Dir != "/custom/plugins" {
		t.Errorf("expected plugin dir %q, got %q", "/custom/plugins", cfg.Plugins.Dir)
	}

	sort.Strings(cfg.Plugins.Enabled)
	if len(cfg.Plugins.Enabled) != 2 {
		t.Fatalf("expected 2 enabled plugins, got %d", len(cfg.Plugins.Enabled))
	}
	if cfg.Plugins.Enabled[0] != "my-custom-auth" {
		t.Errorf("expected first plugin %q, got %q", "my-custom-auth", cfg.Plugins.Enabled[0])
	}
	if cfg.Plugins.Enabled[1] != "xml-formatter" {
		t.Errorf("expected second plugin %q, got %q", "xml-formatter", cfg.Plugins.Enabled[1])
	}
}

func TestLoad_DefaultPluginConfig(t *testing.T) {
	cfg, err := config.Load(
		config.WithGlobalDir(t.TempDir()),
		config.WithProjectDir(t.TempDir()),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// When no plugin config is present, defaults should be used.
	if cfg.Plugins.Dir != "" {
		t.Errorf("expected empty plugin dir by default, got %q", cfg.Plugins.Dir)
	}
	if len(cfg.Plugins.Enabled) != 0 {
		t.Errorf("expected no enabled plugins by default, got %v", cfg.Plugins.Enabled)
	}
}

func TestLoad_ProjectPluginConfigOverridesGlobal(t *testing.T) {
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	globalContent := `plugins:
  dir: /global/plugins
  enabled:
    - plugin-a
`
	projectContent := `plugins:
  dir: /project/plugins
  enabled:
    - plugin-b
    - plugin-c
`
	os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(globalContent), 0644)
	os.WriteFile(filepath.Join(projectDir, ".reqflow.yaml"), []byte(projectContent), 0644)

	cfg, err := config.Load(
		config.WithGlobalDir(globalDir),
		config.WithProjectDir(projectDir),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Plugins.Dir != "/project/plugins" {
		t.Errorf("expected project plugin dir, got %q", cfg.Plugins.Dir)
	}
	if len(cfg.Plugins.Enabled) != 2 {
		t.Errorf("expected 2 enabled plugins from project, got %d", len(cfg.Plugins.Enabled))
	}
}
