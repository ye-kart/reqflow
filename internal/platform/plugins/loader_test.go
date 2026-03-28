package plugins

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ye-kart/reqflow/pkg/plugin"
)

func TestLoadPlugins_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	loaded, err := LoadPlugins(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 loaded plugins, got %d", len(loaded))
	}
}

func TestLoadPlugins_NonExistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")

	loaded, err := LoadPlugins(dir)
	if err != nil {
		t.Fatalf("unexpected error loading from nonexistent dir: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 loaded plugins, got %d", len(loaded))
	}
}

func TestLoadPlugins_SkipsNonSoFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some non-.so files
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "plugin.go"), []byte("package main"), 0644)

	loaded, err := LoadPlugins(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 loaded plugins, got %d", len(loaded))
	}
}

func TestPluginInfo(t *testing.T) {
	info := PluginInfo{
		Name:    "test-plugin",
		Version: "1.0.0",
		Types:   []string{"protocol", "formatter"},
	}
	if info.Name != "test-plugin" {
		t.Errorf("expected name %q, got %q", "test-plugin", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected version %q, got %q", "1.0.0", info.Version)
	}
	if len(info.Types) != 2 {
		t.Errorf("expected 2 types, got %d", len(info.Types))
	}
}

func TestListRegistered(t *testing.T) {
	// Reset the plugin registry for a clean test.
	resetPluginRegistry()

	infos := ListRegistered()
	if len(infos) != 0 {
		t.Errorf("expected 0 registered plugins, got %d", len(infos))
	}
}

func TestGetPluginInfo_NotFound(t *testing.T) {
	resetPluginRegistry()

	_, ok := GetPluginInfo("nonexistent")
	if ok {
		t.Fatal("expected plugin info not to be found")
	}
}

func TestRegisterPluginInfo(t *testing.T) {
	resetPluginRegistry()

	info := PluginInfo{
		Name:    "my-plugin",
		Version: "2.0.0",
		Types:   []string{"auth"},
	}
	RegisterPluginInfo(info)

	got, ok := GetPluginInfo("my-plugin")
	if !ok {
		t.Fatal("expected plugin info to be found")
	}
	if got.Version != "2.0.0" {
		t.Errorf("expected version %q, got %q", "2.0.0", got.Version)
	}

	infos := ListRegistered()
	if len(infos) != 1 {
		t.Errorf("expected 1 registered plugin, got %d", len(infos))
	}
}

func TestRegisterCompileTimePlugin(t *testing.T) {
	resetPluginRegistry()
	// Reset the plugin package registry too.
	plugin.ResetForTesting()

	p := &fakePlugin{name: "builtin-proto", version: "0.1.0"}
	proto := &fakeProto{name: "custom-grpc"}

	RegisterCompileTimePlugin(p, WithProtocol(proto))

	// Check plugin info was registered.
	info, ok := GetPluginInfo("builtin-proto")
	if !ok {
		t.Fatal("expected plugin info to be registered")
	}
	if info.Version != "0.1.0" {
		t.Errorf("expected version %q, got %q", "0.1.0", info.Version)
	}

	// Check protocol was registered in the plugin registry.
	got, ok := plugin.GetProtocol("custom-grpc")
	if !ok {
		t.Fatal("expected protocol to be registered")
	}
	if got.Name() != "custom-grpc" {
		t.Errorf("expected name %q, got %q", "custom-grpc", got.Name())
	}
}

func TestRegisterCompileTimePlugin_WithAuthAndFormatter(t *testing.T) {
	resetPluginRegistry()
	plugin.ResetForTesting()

	p := &fakePlugin{name: "multi-plugin", version: "1.0.0"}
	auth := &fakeAuth{name: "custom-auth", authType: "hmac"}
	fmt := &fakeFmt{name: "yaml-fmt"}

	RegisterCompileTimePlugin(p,
		WithAuthProvider(auth),
		WithFormatter(fmt),
	)

	info, ok := GetPluginInfo("multi-plugin")
	if !ok {
		t.Fatal("expected plugin info to be registered")
	}
	if len(info.Types) != 2 {
		t.Errorf("expected 2 types, got %d: %v", len(info.Types), info.Types)
	}

	if _, ok := plugin.GetAuthProvider("custom-auth"); !ok {
		t.Error("expected auth provider to be registered")
	}
	if _, ok := plugin.GetFormatter("yaml-fmt"); !ok {
		t.Error("expected formatter to be registered")
	}
}

// --- Fakes ---

type fakePlugin struct {
	name    string
	version string
}

func (f *fakePlugin) Name() string                  { return f.name }
func (f *fakePlugin) Version() string                { return f.version }
func (f *fakePlugin) Init(config map[string]string) error { return nil }

type fakeProto struct {
	name string
}

func (f *fakeProto) Name() string { return f.name }
func (f *fakeProto) Execute(_ context.Context, _ map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

type fakeAuth struct {
	name     string
	authType string
}

func (f *fakeAuth) Name() string { return f.name }
func (f *fakeAuth) Type() string { return f.authType }
func (f *fakeAuth) Apply(headers map[string]string, _ map[string]string) (map[string]string, error) {
	return headers, nil
}

type fakeFmt struct {
	name string
}

func (f *fakeFmt) Name() string                              { return f.name }
func (f *fakeFmt) Format(_ map[string]interface{}) ([]byte, error) { return nil, nil }
