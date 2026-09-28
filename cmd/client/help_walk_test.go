package main

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

// B2: `domains custom add` defined --target with shorthand -t, colliding with
// the persistent -t/--token flag. Cobra panics building the merged usage
// string. Walking the whole real tree catches this bug and any future
// shorthand clash the same way.
func TestHelpWalk_NoPanic(t *testing.T) {
	resetGlobals(t)
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if err := cmd.Help(); err != nil {
			t.Errorf("%s --help: %v", cmd.CommandPath(), err)
		}
		for _, c := range cmd.Commands() {
			walk(c)
		}
	}
	walk(root)
}
