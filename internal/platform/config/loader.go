package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ye-kart/reqflow/internal/domain"
	"gopkg.in/yaml.v3"
)

// filePluginConfig is the YAML-serializable representation of plugin config.
type filePluginConfig struct {
	Dir     *string  `yaml:"dir,omitempty"`
	Enabled []string `yaml:"enabled,omitempty"`
}

// fileConfig is the YAML-serializable representation of a config file.
// Pointer fields allow distinguishing "not set" from zero values.
type fileConfig struct {
	DefaultOutput  *string           `yaml:"default_output,omitempty"`
	DefaultTimeout *string           `yaml:"default_timeout,omitempty"`
	NoColor        *bool             `yaml:"no_color,omitempty"`
	DefaultHeaders map[string]string `yaml:"default_headers,omitempty"`
	Plugins        *filePluginConfig `yaml:"plugins,omitempty"`
}

// loadOptions configures where Load looks for config files.
type loadOptions struct {
	globalDir  string
	projectDir string
}

// LoadOption is a functional option for Load.
type LoadOption func(*loadOptions)

// WithGlobalDir sets the directory containing the global config.yaml.
func WithGlobalDir(dir string) LoadOption {
	return func(o *loadOptions) {
		o.globalDir = dir
	}
}

// WithProjectDir sets the directory containing the project .reqflow.yaml.
func WithProjectDir(dir string) LoadOption {
	return func(o *loadOptions) {
		o.projectDir = dir
	}
}

// defaultGlobalDir returns the default global config directory (~/.reqflow).
func defaultGlobalDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".reqflow")
}

// defaultProjectDir returns the current working directory.
func defaultProjectDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

// ResolveGlobalDir returns the global config directory from the given options,
// or the default if none is configured.
func ResolveGlobalDir(opts []LoadOption) string {
	o := &loadOptions{globalDir: defaultGlobalDir()}
	for _, opt := range opts {
		opt(o)
	}
	return o.globalDir
}

// ResolveProjectDir returns the project config directory from the given options,
// or the default if none is configured.
func ResolveProjectDir(opts []LoadOption) string {
	o := &loadOptions{projectDir: defaultProjectDir()}
	for _, opt := range opts {
		opt(o)
	}
	return o.projectDir
}

// Load loads the merged configuration from global and project-local config files.
// Precedence: project-local > global > defaults.
func Load(opts ...LoadOption) (*domain.AppConfig, error) {
	o := &loadOptions{
		globalDir:  defaultGlobalDir(),
		projectDir: defaultProjectDir(),
	}
	for _, opt := range opts {
		opt(o)
	}

	// Start with defaults.
	cfg := &domain.AppConfig{
		Timeout: 30 * time.Second,
		Output:  domain.OutputPretty,
		NoColor: false,
	}

	// Load global config.
	globalPath := filepath.Join(o.globalDir, "config.yaml")
	globalFile, err := loadFileConfig(globalPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading global config: %w", err)
	}
	if globalFile != nil {
		applyFileConfig(cfg, globalFile)
	}

	// Load project-local config (overrides global).
	projectPath := filepath.Join(o.projectDir, ".reqflow.yaml")
	projectFile, err := loadFileConfig(projectPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	if projectFile != nil {
		applyFileConfig(cfg, projectFile)
	}

	return cfg, nil
}

// LoadFile loads a single config file and returns an AppConfig.
func LoadFile(path string) (*domain.AppConfig, error) {
	fc, err := loadFileConfig(path)
	if err != nil {
		return nil, err
	}

	cfg := &domain.AppConfig{}
	applyFileConfig(cfg, fc)
	return cfg, nil
}

// SaveFile writes an AppConfig to the given YAML file path.
func SaveFile(path string, cfg *domain.AppConfig) error {
	fc := &fileConfig{}

	if cfg.Output != "" {
		s := string(cfg.Output)
		fc.DefaultOutput = &s
	}
	if cfg.Timeout != 0 {
		s := cfg.Timeout.String()
		fc.DefaultTimeout = &s
	}
	if cfg.NoColor {
		fc.NoColor = &cfg.NoColor
	}
	if len(cfg.DefaultHeaders) > 0 {
		fc.DefaultHeaders = make(map[string]string)
		for _, h := range cfg.DefaultHeaders {
			fc.DefaultHeaders[h.Key] = h.Value
		}
	}

	data, err := yaml.Marshal(fc)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// loadFileConfig reads and parses a YAML config file.
// Returns (nil, nil-like) via os.ErrNotExist if file does not exist.
func loadFileConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return &fc, nil
}

// applyFileConfig merges a fileConfig into an existing AppConfig.
// Only fields that are explicitly set in the file override the existing values.
func applyFileConfig(cfg *domain.AppConfig, fc *fileConfig) {
	if fc.DefaultOutput != nil {
		cfg.Output = domain.OutputFormat(*fc.DefaultOutput)
	}
	if fc.DefaultTimeout != nil {
		if d, err := time.ParseDuration(*fc.DefaultTimeout); err == nil {
			cfg.Timeout = d
		}
	}
	if fc.NoColor != nil {
		cfg.NoColor = *fc.NoColor
	}
	if fc.DefaultHeaders != nil {
		// Merge headers: build a map from existing, overlay with new.
		headerMap := make(map[string]string)
		for _, h := range cfg.DefaultHeaders {
			headerMap[h.Key] = h.Value
		}
		for k, v := range fc.DefaultHeaders {
			headerMap[k] = v
		}
		cfg.DefaultHeaders = make([]domain.Header, 0, len(headerMap))
		for k, v := range headerMap {
			cfg.DefaultHeaders = append(cfg.DefaultHeaders, domain.Header{Key: k, Value: v})
		}
	}
	if fc.Plugins != nil {
		if fc.Plugins.Dir != nil {
			cfg.Plugins.Dir = *fc.Plugins.Dir
		}
		if fc.Plugins.Enabled != nil {
			cfg.Plugins.Enabled = fc.Plugins.Enabled
		}
	}
}
