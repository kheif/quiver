package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/kheif/quiver/internal/assistant"
	"github.com/kheif/quiver/internal/release"
	"github.com/spf13/cobra"
)

// noUpdateCheckEnv turns the check for a new Quiver version off.
const noUpdateCheckEnv = "QUIVER_NO_UPDATE_CHECK"

// pseudoVersion matches the version Go gives a build from a commit
// between releases, such as 1.1.1-0.20261030120000-abcdef123456.
var pseudoVersion = regexp.MustCompile(`-(\d+\.)?(0\.)?\d{14}-[0-9a-f]{12}`)

// devBuild reports whether v is not a release: a build without a version,
// from a commit between releases or with local changes.
func devBuild(v string) bool {
	return v == "dev" || strings.Contains(v, "+") || pseudoVersion.MatchString(v)
}

// versionCheck asks GitHub for the latest release while a command runs.
type versionCheck struct {
	done  chan struct{}
	state release.State
}

// startVersionCheck starts the check for a command run with args, unless
// it is the MCP server, shell completion, a development build, turned off,
// or stderr is not a terminal. It returns nil then.
func startVersionCheck(root *cobra.Command, args []string, stderr io.Writer) *versionCheck {
	if os.Getenv(noUpdateCheckEnv) != "" || devBuild(currentVersion()) {
		return nil
	}
	if _, ok := terminal(stderr); !ok {
		return nil
	}
	cmd, _, err := root.Find(args)
	if err != nil {
		return nil
	}
	switch {
	case cmd.Name() == cobra.ShellCompRequestCmd, cmd.Name() == cobra.ShellCompNoDescRequestCmd:
		return nil
	case cmd.Name() == "mcp" && cmd.Parent() == root:
		// stdout is the protocol, and the server runs for hours.
		return nil
	}
	v := &versionCheck{done: make(chan struct{})}
	userAgent := "quiver/" + currentVersion()
	go func() {
		defer close(v.done)
		// A failed check is silent; the state found before is still used.
		v.state, _ = release.Check(context.Background(), time.Now(), userAgent)
	}()
	return v
}

// finish waits for the check, which release.Timeout limits, and prints a
// line on stderr when a newer version is out. ran is the command that ran;
// it prints nothing after JSON output.
func (v *versionCheck) finish(ran *cobra.Command, stderr io.Writer) {
	if v == nil {
		return
	}
	<-v.done
	if ran != nil {
		if f := ran.Flags().Lookup("json"); f != nil && f.Value.String() == "true" {
			return
		}
	}
	if line := newVersionLine(v.state.Latest); line != "" {
		fmt.Fprintln(stderr, dim(stderr, line))
	}
}

// newVersionLine says that version latest is out and how to install it;
// "" when it is not newer than this one.
func newVersionLine(latest string) string {
	current := currentVersion()
	if !release.Newer(latest, current) {
		return ""
	}
	return fmt.Sprintf("Quiver %s is out (you have %s): %s", latest, current, upgradeCommand())
}

// upgradeCommand is how to install the latest version the way this one
// was installed.
func upgradeCommand() string {
	switch {
	case os.Getenv(npmEnv) == "1":
		return "npm install -g @kheif/quiver"
	case version == "dev":
		// A release build has the version from ldflags; go install does not.
		return "go install github.com/kheif/quiver/cmd/quiver@latest"
	case runtime.GOOS == "windows":
		return "download it from https://github.com/kheif/quiver/releases/latest"
	}
	line := "curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh"
	bin, err := executable()
	if err != nil {
		return line
	}
	home, _ := os.UserHomeDir()
	if dir := filepath.Dir(bin); home != "" && dir != filepath.Join(home, ".local", "bin") {
		// install.sh puts the binary in ~/.local/bin unless told otherwise.
		line = "curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | QUIVER_INSTALL_DIR=" + assistant.ShellQuote(dir) + " sh"
	}
	return line
}
