// Package e2e runs the compiled quiver binary the way users and assistants
// do: CLI commands as subprocesses, and an MCP client talking to
// "quiver mcp" over stdio. Each test gets its own HOME and Quiver home.
package e2e

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// bin is the quiver binary built by TestMain.
var bin string

func TestMain(m *testing.M) {
	os.Exit(buildAndRun(m))
}

func buildAndRun(m *testing.M) int {
	dir, err := os.MkdirTemp("", "quiver-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()
	bin = filepath.Join(dir, "quiver")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "../cmd/quiver")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build quiver: %v\n%s", err, out)
		return 1
	}
	return m.Run()
}

// env is one user's machine: a HOME, a Quiver home inside it, and a PATH
// with git but not the real claude command.
type env struct {
	t     *testing.T
	home  string
	qhome string
	path  string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	home := t.TempDir()
	return &env{t: t, home: home, qhome: filepath.Join(home, ".quiver"), path: filepath.Dir(git)}
}

// environ is the environment of every subprocess: the parent's, without
// what could reach the real user's files.
func (e *env) environ() []string {
	var out []string
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(key) {
		case "HOME", "USERPROFILE", "PATH", "XDG_CONFIG_HOME", "QUIVER_HOME", "QUIVER_PROJECT", "VISUAL", "EDITOR":
			continue
		}
		if strings.HasPrefix(key, "GIT_") {
			continue
		}
		out = append(out, kv)
	}
	return append(out,
		"HOME="+e.home,
		"USERPROFILE="+e.home,
		"XDG_CONFIG_HOME="+filepath.Join(e.home, ".config"),
		"QUIVER_HOME="+e.qhome,
		"PATH="+e.path,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
}

// run runs quiver in dir with stdin and returns its combined output.
func (e *env) run(dir, stdin string, args ...string) (string, error) {
	e.t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = e.environ()
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// quiver runs quiver in dir and fails the test if it fails.
func (e *env) quiver(dir string, args ...string) string {
	e.t.Helper()
	out, err := e.run(dir, "", args...)
	if err != nil {
		e.t.Fatalf("quiver %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

// git runs git in dir and fails the test if it fails.
func (e *env) git(dir string, args ...string) {
	e.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = e.environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		e.t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// mcp starts "quiver mcp" in dir and connects a client to it.
func (e *env) mcp(dir string) *mcp.ClientSession {
	e.t.Helper()
	cmd := exec.Command(bin, "mcp")
	cmd.Dir = dir
	cmd.Env = e.environ()
	cmd.Stderr = io.Discard
	client := mcp.NewClient(&mcp.Implementation{Name: "e2e", Version: "0"}, nil)
	cs, err := client.Connect(context.Background(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// call calls a tool and returns its text. A tool error fails the test.
func (e *env) call(cs *mcp.ClientSession, tool string, args map[string]any) string {
	e.t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		e.t.Fatalf("%s: %v", tool, err)
	}
	var out strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out.WriteString(tc.Text)
		}
	}
	if res.IsError {
		e.t.Fatalf("%s failed: %s", tool, out.String())
	}
	return out.String()
}

// dir makes an empty folder outside HOME.
func (e *env) dir(name string) string {
	e.t.Helper()
	d := filepath.Join(e.t.TempDir(), name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		e.t.Fatal(err)
	}
	return d
}

// write writes files (slash paths) under dir.
func (e *env) write(dir string, files map[string]string) {
	e.t.Helper()
	for p, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			e.t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			e.t.Fatal(err)
		}
	}
}

// copyDir copies the folder src to dst, like cp -R.
func (e *env) copyDir(src, dst string) {
	e.t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		to := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(to, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(to, data, 0o644)
	})
	if err != nil {
		e.t.Fatal(err)
	}
}

func skillMD(name, description, body string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\n---\n" + body + "\n"
}

func mustContain(t *testing.T, what, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("%s: missing %q in:\n%s", what, w, got)
		}
	}
}

func mustNotContain(t *testing.T, what, got string, unwanted ...string) {
	t.Helper()
	for _, w := range unwanted {
		if strings.Contains(got, w) {
			t.Errorf("%s: unexpected %q in:\n%s", what, w, got)
		}
	}
}
