package commands_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/ye-kart/reqflow/internal/adapters/cli/commands"
	"github.com/ye-kart/reqflow/internal/app"
)

func TestRunCommand_HasReportFlag(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)

	// Find the run command.
	var runCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "run" {
			runCmd = cmd
			break
		}
	}
	if runCmd == nil {
		t.Fatal("expected 'run' subcommand")
	}

	flag := runCmd.Flags().Lookup("report")
	if flag == nil {
		t.Error("expected --report flag on run command")
	}

	fileFlag := runCmd.Flags().Lookup("report-file")
	if fileFlag == nil {
		t.Error("expected --report-file flag on run command")
	}
}

func TestCollectionRunCommand_HasReportFlag(t *testing.T) {
	a := &app.App{}
	root := commands.NewRootCommand(a)

	// Find the collection command, then the run subcommand.
	var colCmd *cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "collection" {
			colCmd = cmd
			break
		}
	}
	if colCmd == nil {
		t.Fatal("expected 'collection' subcommand")
	}

	var runCmd *cobra.Command
	for _, cmd := range colCmd.Commands() {
		if cmd.Name() == "run" {
			runCmd = cmd
			break
		}
	}
	if runCmd == nil {
		t.Fatal("expected 'collection run' subcommand")
	}

	flag := runCmd.Flags().Lookup("report")
	if flag == nil {
		t.Error("expected --report flag on collection run command")
	}

	fileFlag := runCmd.Flags().Lookup("report-file")
	if fileFlag == nil {
		t.Error("expected --report-file flag on collection run command")
	}
}
