package cli

import (
	"bytes"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var out bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "quiver dev\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVerboseFlag(t *testing.T) {
	t.Cleanup(func() { verbose = false })
	cmd := newRootCmd()
	cmd.SetArgs([]string{"--verbose"})
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !verbose {
		t.Error("verbose not set")
	}
}
