package commands_test

import (
	"testing"

	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/app"
)

func TestTUICommandRegistered(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)

	found := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "tui" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'tui' subcommand to be registered")
	}
}

func TestTUICommandHasCorrectUse(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)

	for _, cmd := range root.Commands() {
		if cmd.Name() == "tui" {
			if cmd.Use != "tui" {
				t.Errorf("expected Use 'tui', got %q", cmd.Use)
			}
			return
		}
	}
	t.Fatal("tui command not found")
}

func TestInteractiveFlagRegistered(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)

	flag := root.PersistentFlags().Lookup("interactive")
	if flag == nil {
		t.Error("expected --interactive flag on root command")
	}
}
