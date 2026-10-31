package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kheif/quiver/internal/project"
)

// TestMain points HOME and QUIVER_HOME at a temp folder, so a test that
// forgets sourceEnv never touches the real ~/.quiver or ~/.claude.
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
		// Claude Code's folder is then $HOME/.claude.
		"CLAUDE_CONFIG_DIR": "",
	} {
		if err := os.Setenv(k, v); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	// A test that assigns to the project of the current folder, the
	// package folder, would register the repository: Register walks up to
	// its .git. The release then fails on the dirty tree.
	leak := filepath.Join("..", "..", project.FileName)
	_, before := os.Stat(leak)
	// A tool a test runs, such as the real claude, may write into the
	// current folder too.
	entries := names(".")
	code := m.Run()
	if _, after := os.Stat(leak); before != nil && after == nil {
		fmt.Fprintf(os.Stderr, "a test wrote %s; run commands from a test folder (t.Chdir)\n", leak)
		_ = os.Remove(leak)
		code = 1
	}
	for name := range names(".") {
		if !entries[name] {
			fmt.Fprintf(os.Stderr, "a test wrote %s into the package folder; run commands from a test folder (t.Chdir) and stub assistants (assistanttest.Install)\n", name)
			_ = os.RemoveAll(name)
			code = 1
		}
	}
	return code
}

// names returns the names of the entries in dir.
func names(dir string) map[string]bool {
	out := map[string]bool{}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		out[e.Name()] = true
	}
	return out
}
