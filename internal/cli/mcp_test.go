package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant/assistanttest"
	"github.com/kheif/quiver/internal/project"
	"github.com/kheif/quiver/internal/setup"
)

// TestMCPStdoutHasOnlyMessages runs "quiver mcp" in a child process (this
// test binary, re-run with QUIVER_MCP_CHILD=1) and checks that every line it
// writes to stdout is a JSON-RPC message.
func TestMCPStdoutHasOnlyMessages(t *testing.T) {
	if os.Getenv("QUIVER_MCP_CHILD") == "1" {
		os.Args = []string{"quiver", "mcp"}
		os.Exit(Execute())
	}
	home := sourceEnv(t)
	dir := filepath.Join(t.TempDir(), "app")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := project.Register(dir)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMCPStdoutHasOnlyMessages$")
	cmd.Env = append(os.Environ(), "QUIVER_MCP_CHILD=1", "QUIVER_PROJECT="+dir)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// Keep stdin open until both replies are read: the server stops at EOF.
	for _, msg := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_skills","arguments":{}}}`,
	} {
		if _, err := stdin.Write([]byte(msg + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	var replies []string
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		var msg struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil || msg.JSONRPC != "2.0" {
			t.Errorf("stdout line is not a JSON-RPC message: %q", sc.Text())
		}
		if replies = append(replies, sc.Text()); len(replies) == 2 {
			_ = stdin.Close()
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("%v\nstderr:\n%s", err, stderr.String())
	}
	if len(replies) != 2 || !strings.Contains(replies[1], `"skills"`) {
		t.Errorf("replies: %q", replies)
	}
	if !strings.Contains(stderr.String(), "server started") || !strings.Contains(stderr.String(), "project="+p.ID) {
		t.Errorf("stderr:\n%s", stderr.String())
	}
	if data, err := os.ReadFile(filepath.Join(home, "logs", "mcp.log")); err != nil || !strings.Contains(string(data), "server started") {
		t.Errorf("log file: %v\n%s", err, data)
	}
}

// fakeExecutable makes mcp install see bin as the quiver binary.
func fakeExecutable(t *testing.T, bin string) {
	t.Helper()
	old := executable
	executable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { executable = old })
}

func TestMCPInstall(t *testing.T) {
	f := assistanttest.Install(t)
	fakeExecutable(t, "/opt/homebrew/bin/quiver")
	out, err := runIn(t, "", "mcp", "install")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Registered with Claude Code: /opt/homebrew/bin/quiver mcp") || !strings.Contains(out, "Start a new Claude Code session") {
		t.Errorf("output:\n%s", out)
	}
	if f.User() != "/opt/homebrew/bin/quiver|mcp" {
		t.Errorf("user %q", f.User())
	}

	out, err = runIn(t, "", "mcp", "install")
	if err != nil || !strings.Contains(out, "Already registered") || strings.Contains(out, "new Claude Code session") {
		t.Errorf("second install: %v\n%s", err, out)
	}

	fakeExecutable(t, "/usr/local/bin/quiver")
	out, err = runIn(t, "", "mcp", "install")
	if err != nil || !strings.Contains(out, "Updated the Claude Code registration to /usr/local/bin/quiver mcp") {
		t.Errorf("moved binary: %v\n%s", err, out)
	}

	out, err = runIn(t, "", "mcp", "uninstall")
	if err != nil || !strings.Contains(out, "Removed from Claude Code.") || f.User() != "" {
		t.Errorf("uninstall: %v, user %q\n%s", err, f.User(), out)
	}
	out, err = runIn(t, "", "mcp", "uninstall")
	if err != nil || !strings.Contains(out, "was not registered") {
		t.Errorf("second uninstall: %v\n%s", err, out)
	}
}

func TestMCPInstallHidden(t *testing.T) {
	f := assistanttest.Install(t)
	t.Chdir(t.TempDir())
	f.SetLocal(t, "/dev/quiver", "mcp")
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "install")
	if err != nil {
		t.Fatal(err)
	}
	if want := "warning: in this folder, a registration from Local config (private to you in this project) runs /dev/quiver mcp instead"; !strings.Contains(out, want) {
		t.Errorf("output:\n%s", out)
	}
}

func TestMCPInstallPrint(t *testing.T) {
	f := assistanttest.Install(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "install", "--print")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"claude mcp add --scope user quiver -- /opt/bin/quiver mcp", `"command": "/opt/bin/quiver"`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if len(f.Calls()) != 0 {
		t.Errorf("ran claude: %q", f.Calls())
	}
}

func TestMCPInstallNoClaude(t *testing.T) {
	sourceEnv(t)
	// Create the home while git is still on PATH.
	if _, err := setup.Init(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "install")
	if err != nil || !strings.Contains(out, "run:\n  claude mcp add --scope user quiver -- /opt/bin/quiver mcp") {
		t.Errorf("install: %v\n%s", err, out)
	}
	out, err = runIn(t, "", "mcp", "uninstall")
	if err != nil || !strings.Contains(out, "run:\n  claude mcp remove quiver --scope user") {
		t.Errorf("uninstall: %v\n%s", err, out)
	}
}

// Run through the npm launcher, mcp install registers the quiver command
// on PATH, not the binary inside node_modules.
func TestMCPInstallNpm(t *testing.T) {
	f := assistanttest.Install(t)
	fakeExecutable(t, "/usr/lib/node_modules/@kheif/quiver-linux-x64/bin/quiver")
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "quiver"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(npmEnv, "1")
	out, err := runIn(t, "", "mcp", "install")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(bin, "quiver") + "|mcp"; f.User() != want {
		t.Errorf("user %q, want %q", f.User(), want)
	}
	if !strings.Contains(out, "After switching Node versions") {
		t.Errorf("no npm note:\n%s", out)
	}
}
