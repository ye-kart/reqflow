package commands_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/app"
	"github.com/ye-kart/reqflow/internal/platform/plugins"
	"github.com/ye-kart/reqflow/pkg/plugin"
)

func TestPluginList_NoPlugins(t *testing.T) {
	plugin.ResetForTesting()
	plugins.ResetForTesting()

	a := &app.App{}
	root := commands.NewRootCommand(a,
		commands.WithGlobalConfigDir(t.TempDir()),
		commands.WithProjectConfigDir(t.TempDir()),
	)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"plugin", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No plugins registered") {
		t.Errorf("expected 'No plugins registered' message, got:\n%s", output)
	}
}

func TestPluginList_WithRegisteredPlugins(t *testing.T) {
	plugin.ResetForTesting()
	plugins.ResetForTesting()

	plugins.RegisterPluginInfo(plugins.PluginInfo{
		Name:    "my-auth",
		Version: "1.0.0",
		Types:   []string{"auth"},
		Source:  "compile-time",
	})
	plugins.RegisterPluginInfo(plugins.PluginInfo{
		Name:    "xml-fmt",
		Version: "0.2.0",
		Types:   []string{"formatter"},
		Source:  "compile-time",
	})

	a := &app.App{}
	root := commands.NewRootCommand(a,
		commands.WithGlobalConfigDir(t.TempDir()),
		commands.WithProjectConfigDir(t.TempDir()),
	)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"plugin", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-auth") {
		t.Errorf("expected output to contain 'my-auth', got:\n%s", output)
	}
	if !strings.Contains(output, "xml-fmt") {
		t.Errorf("expected output to contain 'xml-fmt', got:\n%s", output)
	}
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected output to contain version '1.0.0', got:\n%s", output)
	}
}

func TestPluginInfo_ShowsDetails(t *testing.T) {
	plugin.ResetForTesting()
	plugins.ResetForTesting()

	plugins.RegisterPluginInfo(plugins.PluginInfo{
		Name:    "my-auth",
		Version: "1.0.0",
		Types:   []string{"auth", "protocol"},
		Source:  "compile-time",
	})

	a := &app.App{}
	root := commands.NewRootCommand(a,
		commands.WithGlobalConfigDir(t.TempDir()),
		commands.WithProjectConfigDir(t.TempDir()),
	)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"plugin", "info", "my-auth"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "my-auth") {
		t.Errorf("expected output to contain 'my-auth', got:\n%s", output)
	}
	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected output to contain version, got:\n%s", output)
	}
	if !strings.Contains(output, "auth") {
		t.Errorf("expected output to contain type 'auth', got:\n%s", output)
	}
	if !strings.Contains(output, "compile-time") {
		t.Errorf("expected output to contain source 'compile-time', got:\n%s", output)
	}
}

func TestPluginInfo_NotFound(t *testing.T) {
	plugin.ResetForTesting()
	plugins.ResetForTesting()

	a := &app.App{}
	root := commands.NewRootCommand(a,
		commands.WithGlobalConfigDir(t.TempDir()),
		commands.WithProjectConfigDir(t.TempDir()),
	)

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"plugin", "info", "nonexistent"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent plugin")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}
