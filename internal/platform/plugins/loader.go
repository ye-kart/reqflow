// Package plugins provides plugin loading and management for reqflow.
// It supports both compile-time registration via init() and runtime loading
// of Go plugin (.so) files from a configured directory.
package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goplugin "plugin"
	"sync"

	"github.com/ye-kart/reqflow/pkg/plugin"
)

// PluginInfo describes a registered plugin.
type PluginInfo struct {
	Name    string
	Version string
	Types   []string // e.g., "protocol", "auth", "formatter"
	Source  string   // "compile-time" or "shared-object"
}

var (
	mu          sync.RWMutex
	pluginInfos = make(map[string]PluginInfo)
)

// RegisterPluginInfo records metadata about a plugin.
func RegisterPluginInfo(info PluginInfo) {
	mu.Lock()
	defer mu.Unlock()
	pluginInfos[info.Name] = info
}

// GetPluginInfo returns the info for a registered plugin.
func GetPluginInfo(name string) (PluginInfo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	info, ok := pluginInfos[name]
	return info, ok
}

// ListRegistered returns info for all registered plugins.
func ListRegistered() []PluginInfo {
	mu.RLock()
	defer mu.RUnlock()
	infos := make([]PluginInfo, 0, len(pluginInfos))
	for _, info := range pluginInfos {
		infos = append(infos, info)
	}
	return infos
}

// resetPluginRegistry clears plugin info (for testing).
func resetPluginRegistry() {
	mu.Lock()
	defer mu.Unlock()
	pluginInfos = make(map[string]PluginInfo)
}

// CompileTimeOption configures what a compile-time plugin provides.
type CompileTimeOption func(*compileTimeConfig)

type compileTimeConfig struct {
	protocols  []plugin.Protocol
	auths      []plugin.AuthProvider
	formatters []plugin.OutputFormatter
}

// WithProtocol adds a protocol extension to the compile-time registration.
func WithProtocol(p plugin.Protocol) CompileTimeOption {
	return func(c *compileTimeConfig) {
		c.protocols = append(c.protocols, p)
	}
}

// WithAuthProvider adds an auth provider extension to the compile-time registration.
func WithAuthProvider(a plugin.AuthProvider) CompileTimeOption {
	return func(c *compileTimeConfig) {
		c.auths = append(c.auths, a)
	}
}

// WithFormatter adds a formatter extension to the compile-time registration.
func WithFormatter(f plugin.OutputFormatter) CompileTimeOption {
	return func(c *compileTimeConfig) {
		c.formatters = append(c.formatters, f)
	}
}

// RegisterCompileTimePlugin registers a plugin and its extensions at compile time.
// This is intended to be called from init() functions.
func RegisterCompileTimePlugin(p plugin.Plugin, opts ...CompileTimeOption) {
	cfg := &compileTimeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var types []string

	for _, proto := range cfg.protocols {
		plugin.RegisterProtocol(proto)
		types = append(types, "protocol")
	}
	for _, auth := range cfg.auths {
		plugin.RegisterAuthProvider(auth)
		types = append(types, "auth")
	}
	for _, fmt := range cfg.formatters {
		plugin.RegisterFormatter(fmt)
		types = append(types, "formatter")
	}

	RegisterPluginInfo(PluginInfo{
		Name:    p.Name(),
		Version: p.Version(),
		Types:   types,
		Source:  "compile-time",
	})
}

// LoadPlugins scans the given directory for .so plugin files and loads them.
// Each plugin .so must export a "ReqflowPlugin" symbol of type plugin.Plugin.
// Returns info about successfully loaded plugins. Non-existent directory is not
// an error (returns empty slice).
func LoadPlugins(dir string) ([]PluginInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugin directory: %w", err)
	}

	var loaded []PluginInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".so" {
			continue
		}

		info, err := loadSharedPlugin(filepath.Join(dir, entry.Name()))
		if err != nil {
			return loaded, fmt.Errorf("loading plugin %s: %w", entry.Name(), err)
		}
		loaded = append(loaded, info)
	}
	return loaded, nil
}

// loadSharedPlugin loads a single .so plugin file.
func loadSharedPlugin(path string) (PluginInfo, error) {
	p, err := goplugin.Open(path)
	if err != nil {
		return PluginInfo{}, fmt.Errorf("opening plugin: %w", err)
	}

	sym, err := p.Lookup("ReqflowPlugin")
	if err != nil {
		return PluginInfo{}, fmt.Errorf("plugin missing ReqflowPlugin symbol: %w", err)
	}

	plug, ok := sym.(plugin.Plugin)
	if !ok {
		return PluginInfo{}, fmt.Errorf("ReqflowPlugin does not implement plugin.Plugin")
	}

	if err := plug.Init(nil); err != nil {
		return PluginInfo{}, fmt.Errorf("initializing plugin: %w", err)
	}

	var types []string

	// Check for protocol support.
	if proto, ok := sym.(plugin.Protocol); ok {
		plugin.RegisterProtocol(proto)
		types = append(types, "protocol")
	}
	// Check for auth provider support.
	if auth, ok := sym.(plugin.AuthProvider); ok {
		plugin.RegisterAuthProvider(auth)
		types = append(types, "auth")
	}
	// Check for formatter support.
	if fmt, ok := sym.(plugin.OutputFormatter); ok {
		plugin.RegisterFormatter(fmt)
		types = append(types, "formatter")
	}

	info := PluginInfo{
		Name:    plug.Name(),
		Version: plug.Version(),
		Types:   types,
		Source:  "shared-object",
	}
	RegisterPluginInfo(info)

	return info, nil
}
