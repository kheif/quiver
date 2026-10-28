package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain points HOME and QUIVER_HOME at a temp folder, so a test that
// forgets sourceEnv never touches the real ~/.quiver.
func TestMain(m *testing.M) {
	os.Exit(isolated(m))
}

func isolated(m *testing.M) int {
	// The child of TestMCPStdoutHasOnlyMessages gets its folders from its parent.
	if os.Getenv("QUIVER_MCP_CHILD") == "1" {
		return m.Run()
	}
	dir, err := os.MkdirTemp("", "quiver-cli-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()
	for k, v := range map[string]string{
		"HOME":           dir,
		"USERPROFILE":    dir,
		"QUIVER_HOME":    filepath.Join(dir, ".quiver"),
		"QUIVER_PROJECT": "",
	} {
		if err := os.Setenv(k, v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return m.Run()
}
