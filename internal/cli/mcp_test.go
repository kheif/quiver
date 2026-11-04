package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kheif/quiver/internal/assistant"
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

// appHome gives the test an empty home folder, so Cursor and Claude Desktop
// count as not installed. installed creates the folders of the assistants
// with those IDs and returns them by ID.
func appHome(t *testing.T, installed ...string) map[string]*assistant.JSONAssistant {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir reads USERPROFILE on Windows
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData"))
	all, err := assistant.JSONAssistants()
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]*assistant.JSONAssistant{}
	for _, a := range all {
		byID[a.ID] = a
		if slices.Contains(installed, a.ID) {
			if err := os.MkdirAll(a.Dir, 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	return byID
}

func TestMCPInstall(t *testing.T) {
	f := assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/homebrew/bin/quiver")
	out, err := runIn(t, "", "install")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Registered with Claude Code: /opt/homebrew/bin/quiver mcp") || !strings.Contains(out, "Start a new Claude Code session") {
		t.Errorf("output:\n%s", out)
	}
	if f.User() != "/opt/homebrew/bin/quiver|mcp" {
		t.Errorf("user %q", f.User())
	}

	out, err = runIn(t, "", "install")
	if err != nil || !strings.Contains(out, "Already registered") || strings.Contains(out, "new Claude Code session") {
		t.Errorf("second install: %v\n%s", err, out)
	}

	fakeExecutable(t, "/usr/local/bin/quiver")
	out, err = runIn(t, "", "install")
	if err != nil || !strings.Contains(out, "Updated the Claude Code registration to /usr/local/bin/quiver mcp") {
		t.Errorf("moved binary: %v\n%s", err, out)
	}

	out, err = runIn(t, "", "uninstall")
	if err != nil || !strings.Contains(out, "Removed from Claude Code.") || f.User() != "" {
		t.Errorf("uninstall: %v, user %q\n%s", err, f.User(), out)
	}
	out, err = runIn(t, "", "uninstall")
	if err != nil || !strings.Contains(out, "was not registered") {
		t.Errorf("second uninstall: %v\n%s", err, out)
	}
}

// The old quiver mcp install and quiver mcp uninstall still work and print a
// deprecation line naming the new command. They are removed in the next major
// version.
func TestMCPInstallDeprecated(t *testing.T) {
	f := assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/homebrew/bin/quiver")
	out, err := runIn(t, "", "mcp", "install")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `Command "install" is deprecated, use "quiver install"`) {
		t.Errorf("no deprecation line:\n%s", out)
	}
	if !strings.Contains(out, "Registered with Claude Code: /opt/homebrew/bin/quiver mcp") {
		t.Errorf("did not register:\n%s", out)
	}

	out, err = runIn(t, "", "mcp", "uninstall")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `Command "uninstall" is deprecated, use "quiver uninstall"`) {
		t.Errorf("no deprecation line:\n%s", out)
	}
	if !strings.Contains(out, "Removed from Claude Code.") || f.User() != "" {
		t.Errorf("did not remove: user %q\n%s", f.User(), out)
	}
}

// install adds Quiver's session-start hooks to Claude Code's settings file,
// and uninstall removes them.
func TestInstallClaudeHook(t *testing.T) {
	assistanttest.Install(t)
	appHome(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "install", "claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Added the session-start hook for always-on skills") {
		t.Errorf("no hook line:\n%s", out)
	}
	path, _ := assistant.ClaudeSettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	var m struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct {
				Command string
				Timeout int
			}
		}
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("settings JSON: %v\n%s", err, data)
	}
	ss := m.Hooks["SessionStart"]
	if len(ss) != 1 || ss[0].Matcher != "startup|clear|compact" ||
		ss[0].Hooks[0].Command != "/opt/bin/quiver hook session-start claude-code" || ss[0].Hooks[0].Timeout != 10 {
		t.Errorf("SessionStart: %+v", ss)
	}
	sub := m.Hooks["SubagentStart"]
	if len(sub) != 1 || sub[0].Matcher != "" ||
		sub[0].Hooks[0].Command != "/opt/bin/quiver hook subagent-start claude-code" {
		t.Errorf("SubagentStart: %+v", sub)
	}

	out, err = runIn(t, "", "uninstall", "claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Removed the session-start hook from Claude Code.") {
		t.Errorf("no hook removal line:\n%s", out)
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), "quiver hook") {
		t.Errorf("hook still in settings:\n%s", data)
	}
}

// install adds Quiver's sessionStart hook to Cursor's hooks file; uninstall
// removes it.
func TestInstallCursorHook(t *testing.T) {
	appHome(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "install", "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Added the Cursor session-start hook for always-on skills") {
		t.Errorf("install output:\n%s", out)
	}
	path, _ := assistant.CursorHooksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("hooks file: %v", err)
	}
	var m struct {
		Version int
		Hooks   map[string][]struct {
			Command string
			Timeout int
		}
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("hooks JSON: %v\n%s", err, data)
	}
	if m.Version != 1 {
		t.Errorf("version %d, want 1", m.Version)
	}
	ss := m.Hooks["sessionStart"]
	if len(ss) != 1 || ss[0].Command != "/opt/bin/quiver hook session-start cursor" || ss[0].Timeout != 10 {
		t.Errorf("sessionStart: %+v", ss)
	}
	if _, has := m.Hooks["subagentStart"]; has {
		t.Errorf("Cursor has no sub-agent hook, but one was written: %v", m.Hooks["subagentStart"])
	}

	out, err = runIn(t, "", "uninstall", "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Removed the session-start hook from Cursor.") {
		t.Errorf("uninstall output:\n%s", out)
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), "quiver hook") {
		t.Errorf("hook still in file:\n%s", data)
	}
}

// install adds Quiver's "quiver" hook to Antigravity's hooks file; uninstall
// removes it.
func TestInstallAntigravityHook(t *testing.T) {
	appHome(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "install", "antigravity")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Added the Antigravity session-start hook for always-on skills") {
		t.Errorf("install output:\n%s", out)
	}
	path, _ := assistant.AntigravityHooksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("hooks file: %v", err)
	}
	var m map[string]map[string][]struct {
		Type    string
		Command string
		Timeout int
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("hooks JSON: %v\n%s", err, data)
	}
	ss := m["quiver"]["SessionStart"]
	if len(ss) != 1 || ss[0].Type != "command" || ss[0].Command != "/opt/bin/quiver hook session-start antigravity" || ss[0].Timeout != 10 {
		t.Errorf("quiver SessionStart: %+v", ss)
	}

	out, err = runIn(t, "", "uninstall", "antigravity")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Removed the session-start hook from Antigravity.") {
		t.Errorf("uninstall output:\n%s", out)
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), "quiver hook") {
		t.Errorf("hook still in file:\n%s", data)
	}
}

// install registers VS Code under the "servers" key with "type": "stdio",
// writes Quiver's whole hook file, and reports a hook; doctor is ok; uninstall
// removes both.
func TestInstallVSCode(t *testing.T) {
	assistanttest.Install(t) // stub claude, so doctor does not run the real one
	apps := appHome(t, "vscode")
	fakeExecutable(t, "/opt/bin/quiver")
	vsc := apps["vscode"]
	out, err := runIn(t, "", "install", "vscode")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Registered with VS Code: /opt/bin/quiver mcp") ||
		!strings.Contains(out, "Added the VS Code session-start hook for always-on skills") {
		t.Errorf("install output:\n%s", out)
	}
	// The MCP entry is under "servers" with "type": "stdio".
	var cfg struct {
		Servers map[string]struct {
			Command string
			Args    []string
			Type    string
		}
	}
	data, err := os.ReadFile(vsc.Path)
	if err != nil {
		t.Fatalf("mcp.json: %v", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("mcp.json JSON: %v\n%s", err, data)
	}
	if e := cfg.Servers["quiver"]; e.Command != "/opt/bin/quiver" || len(e.Args) != 1 || e.Args[0] != "mcp" || e.Type != "stdio" {
		t.Errorf("servers.quiver: %+v", cfg.Servers)
	}

	// The hook file has SessionStart and SubagentStart flat entries.
	path, _ := assistant.VSCodeHooksPath()
	hdata, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("hooks file: %v", err)
	}
	var hooks map[string][]struct {
		Type       string
		Command    string
		TimeoutSec int
	}
	if err := json.Unmarshal(hdata, &hooks); err != nil {
		t.Fatalf("hooks JSON: %v\n%s", err, hdata)
	}
	if ss := hooks["SessionStart"]; len(ss) != 1 || ss[0].Type != "command" ||
		ss[0].Command != "/opt/bin/quiver hook session-start vscode" || ss[0].TimeoutSec != 10 {
		t.Errorf("SessionStart: %+v", hooks["SessionStart"])
	}
	if sub := hooks["SubagentStart"]; len(sub) != 1 || sub[0].Command != "/opt/bin/quiver hook subagent-start vscode" {
		t.Errorf("SubagentStart: %+v", hooks["SubagentStart"])
	}

	// A second install changes nothing.
	out, err = runIn(t, "", "install", "vscode")
	if err != nil || strings.Contains(out, "Restart") || strings.Contains(out, "Added the VS Code") {
		t.Errorf("second install: %v\n%s", err, out)
	}

	out, _ = runIn(t, "", "doctor")
	wantLines(t, out, "ok     registered with VS Code", "ok     VS Code session-start hook")

	out, err = runIn(t, "", "uninstall", "vscode")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Removed from VS Code.") || !strings.Contains(out, "Removed the session-start hook from VS Code.") {
		t.Errorf("uninstall output:\n%s", out)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("hook file still there: %v", err)
	}
}

// install adds Quiver's hooks to Codex's hooks file and trusts them; uninstall
// removes them.
func TestInstallCodexHook(t *testing.T) {
	assistanttest.InstallCodex(t)
	appHome(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "install", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Added the Codex session-start hook for always-on skills") ||
		!strings.Contains(out, "Trusted the Quiver hooks in Codex.") {
		t.Errorf("install output:\n%s", out)
	}
	path, _ := assistant.CodexHooksPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("hooks file: %v", err)
	}
	var m struct {
		Hooks map[string][]struct {
			Matcher string
			Hooks   []struct {
				Command                string
				Timeout                int
				AdditionalContextLimit int
				StatusMessage          string
			}
		}
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("hooks JSON: %v\n%s", err, data)
	}
	h := m.Hooks["SessionStart"][0]
	if h.Matcher != "startup|clear|compact" || h.Hooks[0].Command != "/opt/bin/quiver hook session-start codex" ||
		h.Hooks[0].Timeout != 10 || h.Hooks[0].AdditionalContextLimit != 0 || h.Hooks[0].StatusMessage != "Quiver always-on skills" {
		t.Errorf("SessionStart: %+v", h)
	}
	if sub := m.Hooks["SubagentStart"][0]; sub.Matcher != "" || sub.Hooks[0].Command != "/opt/bin/quiver hook subagent-start codex" {
		t.Errorf("SubagentStart: %+v", sub)
	}

	out, err = runIn(t, "", "uninstall", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Removed the session-start hook from Codex.") {
		t.Errorf("uninstall output:\n%s", out)
	}
	data, _ = os.ReadFile(path)
	if strings.Contains(string(data), "quiver hook") {
		t.Errorf("hook still in file:\n%s", data)
	}
}

// The install report and doctor name the always-on skills that do not load by
// themselves in an assistant without a hook.
func TestDeliveryReportNamesAlwaysOn(t *testing.T) {
	f := listEnv(t) // pdf is a global always-on skill; HOME is a temp folder
	assistanttest.Install(t)
	fakeExecutable(t, "/opt/bin/quiver")
	desktop, err := assistant.ClaudeDesktop()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(desktop.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(f.app)

	out, err := runIn(t, "", "install", "claude-code", "claude-desktop")
	if err != nil {
		t.Fatal(err)
	}
	wantLines(t, out,
		"  Claude Code: always-on skills load from a session-start hook.",
		"  Claude Desktop: no session-start hook. Quiver's skills are loaded when the model decides to.",
		"    These always-on skills do not load by themselves there: pdf.",
	)
	// The hook assistant's line is not followed by the "do not load" note.
	if i := strings.Index(out, "Claude Code: always-on"); i >= 0 {
		rest := out[i:]
		if before := strings.Index(rest, "do not load by themselves"); before >= 0 && before < strings.Index(rest, "Claude Desktop") {
			t.Errorf("Claude Code got the always-on note:\n%s", out)
		}
	}

	out, _ = runIn(t, "", "doctor")
	wantLines(t, out, "Claude Desktop has no session-start hook, so these always-on skills do not load by themselves there: pdf")
}

func TestMCPInstallHidden(t *testing.T) {
	f := assistanttest.Install(t)
	appHome(t)
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
	appHome(t)
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
	appHome(t)
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
	apps := appHome(t, "cursor")
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
	// Desktop apps have no node on PATH, so they get the binary itself.
	if reg, err := apps["cursor"].Get(); err != nil || reg == nil || reg.Command != "/usr/lib/node_modules/@kheif/quiver-linux-x64/bin/quiver" {
		t.Errorf("cursor: %+v %v", reg, err)
	}
	out, _ = runIn(t, "", "doctor")
	wantLines(t, out, "ok     registered with Claude Code", "ok     registered with Cursor")
}

// Without arguments, mcp install sets up every installed assistant and leaves
// the others alone.
func TestMCPInstallEveryAssistant(t *testing.T) {
	f := assistanttest.Install(t)
	apps := appHome(t, "cursor", "claude-desktop", "antigravity", "vscode")
	cx := assistanttest.InstallCodex(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "install")
	if err != nil {
		t.Fatal(err)
	}
	cursor, desktop, agy, vsc := apps["cursor"], apps["claude-desktop"], apps["antigravity"], apps["vscode"]
	wantLines(t, out,
		"Registered with Claude Code: /opt/bin/quiver mcp",
		"Registered with Codex: /opt/bin/quiver mcp\nStart a new Codex session to use it.",
		"Registered with Cursor: /opt/bin/quiver mcp ("+cursor.Path+")\nRestart Cursor to use it.\n",
		"Registered with Claude Desktop: /opt/bin/quiver mcp ("+desktop.Path+")",
		"Claude Desktop does not start Quiver in a project folder, so it gets only the global skills.",
		"Registered with Antigravity: /opt/bin/quiver mcp ("+agy.Path+")\nRestart Antigravity to use it.",
		"Registered with VS Code: /opt/bin/quiver mcp ("+vsc.Path+")\nRestart VS Code to use it.",
		// The delivery report, one line per assistant.
		"How always-on skills reach each assistant:",
		"  Claude Code: always-on skills load from a session-start hook.",
		"  Codex: always-on skills load from a session-start hook.",
		"  Cursor: always-on skills load from a session-start hook.",
		"  Claude Desktop: no session-start hook. Quiver's skills are loaded when the model decides to.",
		"  Antigravity: always-on skills load from a session-start hook.",
		"  VS Code: always-on skills load from a session-start hook.",
	)
	if f.User() != "/opt/bin/quiver|mcp" {
		t.Errorf("claude user %q", f.User())
	}
	if cx.User() != "/opt/bin/quiver|mcp|" {
		t.Errorf("codex user %q", cx.User())
	}
	for _, a := range []*assistant.JSONAssistant{cursor, desktop, agy, vsc} {
		if reg, err := a.Get(); err != nil || reg == nil || reg.Command != "/opt/bin/quiver" {
			t.Errorf("%s: %+v %v", a.ID, reg, err)
		}
	}

	out, _ = runIn(t, "", "doctor")
	wantLines(t, out, "ok     registered with Claude Code", "ok     registered with Codex", "ok     registered with Cursor", "ok     registered with Claude Desktop", "ok     registered with Antigravity", "ok     registered with VS Code", "ok     VS Code session-start hook")

	out, err = runIn(t, "", "mcp", "install")
	if err != nil || strings.Contains(out, "Restart") || strings.Contains(out, "Start a new") {
		t.Errorf("second install: %v\n%s", err, out)
	}
	wantLines(t, out, "Already registered with Cursor: /opt/bin/quiver mcp", "Already registered with Codex: /opt/bin/quiver mcp")

	fakeExecutable(t, "/usr/local/bin/quiver")
	out, _ = runIn(t, "", "doctor")
	wantLines(t, out,
		"warn   Cursor runs another quiver\n       it runs /opt/bin/quiver mcp, but this is /usr/local/bin/quiver; run: quiver install cursor",
		"warn   Codex runs another quiver\n       it runs /opt/bin/quiver mcp, but this is /usr/local/bin/quiver; run: quiver install codex",
	)

	out, err = runIn(t, "", "mcp", "uninstall")
	if err != nil {
		t.Fatal(err)
	}
	wantLines(t, out, "Removed from Claude Code.", "Removed from Codex.", "Removed from Cursor.", "Removed from Claude Desktop.", "Removed from Antigravity.", "Removed from VS Code.")
	out, _ = runIn(t, "", "doctor")
	wantLines(t, out,
		"warn   Codex is installed, but Quiver is not registered with it; run: quiver install codex",
		"warn   Cursor is installed, but Quiver is not registered with it; run: quiver install cursor",
	)
	out, err = runIn(t, "", "mcp", "uninstall", "cursor", "codex")
	if err != nil || !strings.Contains(out, "Quiver was not registered with Cursor.") || !strings.Contains(out, "Quiver was not registered with Codex.") {
		t.Errorf("second uninstall: %v\n%s", err, out)
	}
}

// Without the codex command, mcp install codex prints the command to run.
func TestMCPInstallNoCodex(t *testing.T) {
	appHome(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "install", "codex")
	if err != nil {
		t.Fatal(err)
	}
	wantLines(t, out, "The codex command was not found. To register Quiver with Codex, run:\n  codex mcp add quiver -- /opt/bin/quiver mcp")
	out, err = runIn(t, "", "mcp", "uninstall", "codex")
	if err != nil {
		t.Fatal(err)
	}
	wantLines(t, out, "The codex command was not found. To remove Quiver from Codex, run:\n  codex mcp remove quiver")
}

func TestMCPInstallNamed(t *testing.T) {
	f := assistanttest.Install(t)
	apps := appHome(t) // naming works when the assistant's folder is missing too
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "install", "cursor")
	if err != nil || !strings.Contains(out, "Registered with Cursor") || strings.Contains(out, "Claude") {
		t.Errorf("install: %v\n%s", err, out)
	}
	if reg, err := apps["cursor"].Get(); err != nil || reg == nil {
		t.Errorf("cursor: %+v %v", reg, err)
	}
	if len(f.Calls()) != 0 || apps["claude-desktop"].Installed() {
		t.Errorf("set up other assistants: %q", f.Calls())
	}
	if _, err := runIn(t, "", "mcp", "install", "cursor", "vim"); err == nil || !strings.Contains(err.Error(), "use one of: claude-code, codex, cursor, claude-desktop, antigravity") {
		t.Errorf("unknown assistant: %v", err)
	}

	// Several names, given twice, set each one up once.
	out, err = runIn(t, "", "mcp", "install", "claude-code", "claude-desktop", "claude-code")
	if err != nil || strings.Count(out, "Registered with Claude Code") != 1 || !strings.Contains(out, "Registered with Claude Desktop") || strings.Contains(out, "Cursor") {
		t.Errorf("install several: %v\n%s", err, out)
	}
	out, err = runIn(t, "", "mcp", "uninstall", "cursor", "claude-desktop")
	if err != nil || !strings.Contains(out, "Removed from Cursor.") || !strings.Contains(out, "Removed from Claude Desktop.") || strings.Contains(out, "Claude Code") {
		t.Errorf("uninstall several: %v\n%s", err, out)
	}
	if f.User() != "/opt/bin/quiver|mcp" {
		t.Errorf("claude user %q", f.User())
	}
}

// The --for flag of v1.2.0 still names an assistant, hidden and deprecated.
func TestMCPInstallFor(t *testing.T) {
	f := assistanttest.Install(t)
	apps := appHome(t)
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "install", "--for", "cursor")
	if err != nil || !strings.Contains(out, "Registered with Cursor") || strings.Contains(out, "Claude") {
		t.Errorf("install: %v\n%s", err, out)
	}
	wantLines(t, out, "Flag --for has been deprecated, name the assistant as an argument: quiver mcp install <assistant>")
	if len(f.Calls()) != 0 {
		t.Errorf("ran claude: %q", f.Calls())
	}

	// With arguments, --for counts as one more.
	out, err = runIn(t, "", "mcp", "uninstall", "claude-desktop", "--for", "cursor")
	if err != nil || !strings.Contains(out, "Removed from Cursor.") || !strings.Contains(out, "Quiver was not registered with Claude Desktop.") {
		t.Errorf("uninstall: %v\n%s", err, out)
	}
	wantLines(t, out, "Flag --for has been deprecated, name the assistant as an argument: quiver mcp uninstall <assistant>")
	if reg, err := apps["cursor"].Get(); err != nil || reg != nil {
		t.Errorf("cursor: %+v %v", reg, err)
	}

	for _, sub := range []string{"install", "uninstall"} {
		if out, _ := runIn(t, "", "mcp", sub, "--help"); strings.Contains(out, "--for") {
			t.Errorf("mcp %s --help shows --for:\n%s", sub, out)
		}
	}
}

// A broken config file is reported and left alone; the other assistants
// are still set up.
func TestMCPInstallBrokenJSON(t *testing.T) {
	f := assistanttest.Install(t)
	apps := appHome(t, "cursor")
	fakeExecutable(t, "/opt/bin/quiver")
	const broken = `{"mcpServers": {`
	if err := os.WriteFile(apps["cursor"].Path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runIn(t, "", "mcp", "install")
	if err == nil {
		t.Errorf("no error:\n%s", out)
	}
	wantLines(t, out, "Cursor: not registered: "+apps["cursor"].Path+" is not valid JSON, so it was left as it is")
	if f.User() != "/opt/bin/quiver|mcp" {
		t.Errorf("claude user %q", f.User())
	}
	if b, _ := os.ReadFile(apps["cursor"].Path); string(b) != broken {
		t.Errorf("file changed: %s", b)
	}
	out, _ = runIn(t, "", "doctor")
	wantLines(t, out, "error  Cursor MCP config")
}

func TestMCPConfig(t *testing.T) {
	fakeExecutable(t, "/opt/bin/quiver")
	out, err := runIn(t, "", "mcp", "config")
	if err != nil || out != assistant.ConfigSnippet("/opt/bin/quiver")+"\n" {
		t.Errorf("config: %v\n%s", err, out)
	}
}
