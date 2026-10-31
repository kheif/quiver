package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/kheif/quiver/internal/assistant/assistanttest"
	"github.com/kheif/quiver/internal/release"
	"github.com/kheif/quiver/internal/setup"
)

// noticeEnv makes this a release build 1.2.0 with stderr at a terminal,
// and serves tag as the latest release, or fails with status 500 when tag
// is empty. It returns the number of requests.
func noticeEnv(t *testing.T, tag string) *atomic.Int32 {
	t.Helper()
	sourceEnv(t)
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	// doctor runs claude; never the real one.
	assistanttest.Install(t)
	t.Setenv("NO_COLOR", "1")
	t.Setenv(noUpdateCheckEnv, "")
	t.Setenv(npmEnv, "")
	oldVersion, oldTerminal := version, terminal
	version = "1.2.0"
	terminal = func(io.Writer) (int, bool) { return 80, true }
	t.Cleanup(func() { version, terminal = oldVersion, oldTerminal })
	fakeExecutable(t, filepath.Join(os.Getenv("HOME"), ".local", "bin", "quiver"))

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		if tag == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name": "` + tag + `"}`))
	}))
	t.Cleanup(srv.Close)
	oldURL := release.LatestURL
	release.LatestURL = srv.URL
	t.Cleanup(func() { release.LatestURL = oldURL })
	return &hits
}

// executeArgs runs quiver like Execute and returns stdout, stderr and the
// exit code.
func executeArgs(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetIn(strings.NewReader(""))
	code = execute(root, args, &errOut)
	return out.String(), errOut.String(), code
}

const newerLine = "Quiver 1.3.0 is out (you have 1.2.0): curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh\n"

func TestNoticeNewer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows gets the releases page instead of the curl line")
	}
	hits := noticeEnv(t, "v1.3.0")
	out, errOut, code := executeArgs(t, "projects")
	if code != 0 || errOut != newerLine || !strings.HasPrefix(out, "No projects.") {
		t.Errorf("code %d\nstdout %q\nstderr %q", code, out, errOut)
	}
	// The next command uses the saved answer and does not ask again.
	if _, errOut, _ = executeArgs(t, "projects"); errOut != newerLine || hits.Load() != 1 {
		t.Errorf("second run: %d requests, stderr %q", hits.Load(), errOut)
	}
	// After an error, the line comes last.
	if _, errOut, code = executeArgs(t, "where", "nope"); code != 1 || !strings.HasPrefix(errOut, "error: ") || !strings.HasSuffix(errOut, newerLine) {
		t.Errorf("after an error: code %d, stderr %q", code, errOut)
	}
}

func TestNoticeSameOrFailed(t *testing.T) {
	for _, tag := range []string{"v1.2.0", "v1.1.0", ""} {
		hits := noticeEnv(t, tag)
		if _, errOut, code := executeArgs(t, "projects"); code != 0 || errOut != "" || hits.Load() != 1 {
			t.Errorf("latest %q: code %d, %d requests, stderr %q", tag, code, hits.Load(), errOut)
		}
	}
}

func TestNoticeNotShown(t *testing.T) {
	hits := noticeEnv(t, "v1.3.0")
	// JSON output: the check runs, but no line follows it.
	if _, errOut, _ := executeArgs(t, "projects", "--json"); errOut != "" || hits.Load() != 1 {
		t.Errorf("--json: %d requests, stderr %q", hits.Load(), errOut)
	}
	root := newRootCmd()
	for name, args := range map[string][]string{
		"mcp":        {"mcp"},
		"completion": {"__complete", "add", ""},
	} {
		if startVersionCheck(root, args, io.Discard) != nil {
			t.Errorf("%s: check started", name)
		}
	}
	if v := startVersionCheck(root, []string{"mcp", "install"}, io.Discard); v == nil {
		t.Error("mcp install: no check")
	} else {
		v.finish(nil, io.Discard)
	}

	t.Setenv(noUpdateCheckEnv, "1")
	if startVersionCheck(root, []string{"status"}, io.Discard) != nil {
		t.Errorf("%s=1: check started", noUpdateCheckEnv)
	}
	t.Setenv(noUpdateCheckEnv, "")
	terminal = func(io.Writer) (int, bool) { return 0, false }
	if startVersionCheck(root, []string{"status"}, io.Discard) != nil {
		t.Error("not a terminal: check started")
	}
	terminal = func(io.Writer) (int, bool) { return 80, true }
	version = "dev"
	if startVersionCheck(root, []string{"status"}, io.Discard) != nil {
		t.Error("dev build: check started")
	}
}

func TestDevBuild(t *testing.T) {
	for v, want := range map[string]bool{
		"dev":                                 true,
		"1.2.0":                               false,
		"1.2.0-rc.1":                          false,
		"1.1.1-0.20261030120000-abcdef123456": true,
		"0.0.0-20261030120000-abcdef123456":   true,
		"1.2.0+dirty":                         true,
		"1.1.1-0.20261030120000-abcdef123456+dirty": true,
	} {
		if got := devBuild(v); got != want {
			t.Errorf("devBuild(%q) = %v", v, got)
		}
	}
}

func TestUpgradeCommand(t *testing.T) {
	noticeEnv(t, "v1.3.0")
	if runtime.GOOS != "windows" {
		if got := upgradeCommand(); got != "curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh" {
			t.Errorf("~/.local/bin: %q", got)
		}
		fakeExecutable(t, "/opt/my tools/quiver")
		if got := upgradeCommand(); got != "curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | QUIVER_INSTALL_DIR='/opt/my tools' sh" {
			t.Errorf("other folder: %q", got)
		}
	}
	version = "dev" // go install: the version comes from the module
	if got := upgradeCommand(); got != "go install github.com/kheif/quiver/cmd/quiver@latest" {
		t.Errorf("go install: %q", got)
	}
	t.Setenv(npmEnv, "1")
	if got := upgradeCommand(); got != "npm install -g @kheif/quiver" {
		t.Errorf("npm: %q", got)
	}
}

func TestDoctorVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows gets the releases page instead of the curl line")
	}
	noticeEnv(t, "v1.3.0")
	out, _ := runIn(t, "", "doctor")
	if !strings.Contains(out, "\ninfo   "+newerLine) {
		t.Errorf("newer:\n%s", out)
	}
	noticeEnv(t, "v1.2.0")
	if out, _ = runIn(t, "", "doctor"); !strings.Contains(out, "\ninfo   Quiver 1.2.0 is the latest version\n") {
		t.Errorf("same:\n%s", out)
	}
	noticeEnv(t, "")
	if out, _ = runIn(t, "", "doctor"); !strings.Contains(out, "\ninfo   latest Quiver version unknown\n       latest release: 500 Internal Server Error\n") {
		t.Errorf("failed:\n%s", out)
	}
	t.Setenv(noUpdateCheckEnv, "1")
	if out, _ = runIn(t, "", "doctor"); !strings.Contains(out, "latest Quiver version (skipped: $"+noUpdateCheckEnv+" is set)") {
		t.Errorf("disabled:\n%s", out)
	}
}
