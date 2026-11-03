package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kheif/quiver/internal/server"
)

// hookEnv gives the Quiver home always-on skills with a body each: zeta and
// alpha globally, beta and alpha in a project, and gone, which is global
// always-on but no longer in the library. It returns the project folder.
func hookEnv(t *testing.T) string {
	t.Helper()
	dir := assignEnv(t, "alpha", "beta", "zeta", "gone")
	for _, id := range []string{"alpha", "beta", "zeta", "gone"} {
		writeBody(t, id, "Body of "+id+".\n")
	}
	for _, args := range [][]string{
		{"add", "zeta", "alpha", "gone", "--global", "--always"},
		{"add", "beta", "alpha", "--project", dir, "--always"},
	} {
		if _, err := runIn(t, "", args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.RemoveAll(filepath.Join(os.Getenv("QUIVER_HOME"), "local", "gone")); err != nil {
		t.Fatal(err)
	}
	return dir
}

// writeBody gives a local skill a body after its front matter.
func writeBody(t *testing.T, id, body string) {
	t.Helper()
	content := "---\nname: " + id + "\ndescription: Test skill.\n---\n" + body
	path := filepath.Join(os.Getenv("QUIVER_HOME"), "local", id, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// runHookCmd runs quiver hook with args and stdin. The command never fails.
func runHookCmd(t *testing.T, stdin string, args ...string) (stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{"hook"}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook %v: %v", args, err)
	}
	return out.String(), errOut.String()
}

// nestedContext returns hookSpecificOutput's event name and text.
func nestedContext(t *testing.T, out string) (event, text string) {
	t.Helper()
	var got struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("%v in %q", err, out)
	}
	return got.HookSpecificOutput.HookEventName, got.HookSpecificOutput.AdditionalContext
}

func cwdInput(dir string) string {
	b, _ := json.Marshal(map[string]string{"cwd": dir, "hook_event_name": "SessionStart"})
	return string(b)
}

// hookLog returns the hook's log.
func hookLog(t *testing.T) string {
	t.Helper()
	return string(readTestFile(t, filepath.Join(os.Getenv("QUIVER_HOME"), filepath.FromSlash(hookLogFile))))
}

// wantOrder fails unless each want is in text, in this order.
func wantOrder(t *testing.T, text string, want ...string) {
	t.Helper()
	at := -1
	for _, w := range want {
		i := strings.Index(text, w)
		if i <= at {
			t.Fatalf("%q missing or out of order in:\n%s", w, text)
		}
		at = i
	}
}

func TestHookSessionStart(t *testing.T) {
	dir := hookEnv(t)
	out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	if errOut != "" {
		t.Errorf("stderr: %s", errOut)
	}
	event, text := nestedContext(t, out)
	if event != "SessionStart" {
		t.Errorf("event %q", event)
	}
	wantOrder(t, text,
		"<quiver-always-on>\nAlways-on skills from Quiver. Follow each one for the whole conversation.\n",
		"\n## alpha\nBody of alpha.\n\n---\n",
		"\n## zeta\nBody of zeta.\n",
		"\n## beta\nBody of beta.\n",
		"Extra files of these skills are loaded with the get_skill_file tool of the quiver MCP server.\n</quiver-always-on>\n")
	if strings.Count(text, "## alpha") != 1 || strings.Contains(text, "gone") {
		t.Errorf("alpha twice or the removed skill in:\n%s", text)
	}
	if !strings.HasSuffix(text, "</quiver-always-on>\n") {
		t.Errorf("block does not end with its marker:\n%s", text)
	}
	if log := hookLog(t); !strings.Contains(log, "skills=alpha,zeta,beta") || !strings.Contains(log, "assistant=claude-code") {
		t.Errorf("log: %s", log)
	}
}

func TestHookFolder(t *testing.T) {
	dir := hookEnv(t)
	outside := t.TempDir()
	for name, tc := range map[string]struct {
		stdin, chdir, env string
		want              []string
	}{
		"outside a project": {stdin: cwdInput(outside), want: []string{"alpha", "zeta"}},
		"QUIVER_PROJECT":    {stdin: cwdInput(outside), env: dir, want: []string{"alpha", "zeta", "beta"}},
		"no input":          {chdir: dir, want: []string{"alpha", "zeta", "beta"}},
		"input not JSON":    {stdin: "not json", chdir: outside, want: []string{"alpha", "zeta"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(server.ProjectEnv, tc.env)
			if tc.chdir != "" {
				t.Chdir(tc.chdir)
			}
			out, _ := runHookCmd(t, tc.stdin, "session-start", "codex")
			_, text := nestedContext(t, out)
			var got []string
			for _, line := range strings.Split(text, "\n") {
				if id, ok := strings.CutPrefix(line, "## "); ok {
					got = append(got, id)
				}
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("skills %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHookFormats(t *testing.T) {
	dir := hookEnv(t)
	b, _ := json.Marshal(map[string]any{"workspace_roots": []string{dir}})
	cursorIn := string(b)
	b, _ = json.Marshal(map[string]any{"workspacePaths": []string{dir}, "conversationId": "c1"})
	agyIn := string(b)
	for name, tc := range map[string]struct {
		args  []string
		stdin string
		text  func(map[string]any) any
	}{
		"codex": {[]string{"session-start", "codex"}, cwdInput(dir), func(m map[string]any) any {
			return m["hookSpecificOutput"].(map[string]any)["additionalContext"]
		}},
		"vscode top level": {[]string{"session-start", "vscode"}, cwdInput(dir), func(m map[string]any) any { return m["additionalContext"] }},
		"vscode nested": {[]string{"session-start", "vscode"}, cwdInput(dir), func(m map[string]any) any {
			return m["hookSpecificOutput"].(map[string]any)["additionalContext"]
		}},
		"cursor": {[]string{"session-start", "cursor"}, cursorIn, func(m map[string]any) any { return m["additional_context"] }},
		"antigravity": {[]string{"session-start", "antigravity"}, agyIn, func(m map[string]any) any {
			step := m["injectSteps"].([]any)[0].(map[string]any)
			return step["systemMessage"].(map[string]any)["systemMessage"]
		}},
	} {
		t.Run(name, func(t *testing.T) {
			out, _ := runHookCmd(t, tc.stdin, tc.args...)
			var m map[string]any
			if err := json.Unmarshal([]byte(out), &m); err != nil {
				t.Fatalf("%v in %q", err, out)
			}
			text, _ := tc.text(m).(string)
			// The folder came from each assistant's own field: beta is the project's.
			if !strings.Contains(text, "## beta\n") || !strings.HasPrefix(text, "<quiver-always-on>") {
				t.Errorf("text:\n%s", text)
			}
		})
	}
}

func TestHookSubagentStart(t *testing.T) {
	dir := hookEnv(t)
	out, _ := runHookCmd(t, cwdInput(dir), "subagent-start", "claude-code")
	event, text := nestedContext(t, out)
	if event != "SubagentStart" || !strings.Contains(text, "## beta\n") {
		t.Errorf("event %q, text:\n%s", event, text)
	}
	// Cursor's sub-agent event cannot take text, and a hook never fails.
	out, errOut := runHookCmd(t, cwdInput(dir), "subagent-start", "cursor")
	if out != "" || !strings.Contains(errOut, "no sub-agent hook") {
		t.Errorf("cursor: stdout %q, stderr %q", out, errOut)
	}
}

func TestHookSizeLimit(t *testing.T) {
	dir := assignEnv(t, "big1", "big2", "big3")
	// Characters, not bytes, count: each body is 4,000 characters and
	// 8,000 bytes.
	for _, id := range []string{"big1", "big2", "big3"} {
		writeBody(t, id, strings.Repeat("ş", 3999)+"\n")
	}
	if _, err := runIn(t, "", "add", "big1", "big2", "big3", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	out, _ := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	if n := utf8.RuneCountInString(text); n > server.MaxAlwaysOn {
		t.Errorf("block has %d characters", n)
	}
	wantOrder(t, text, "## big1\n", "## big2\n",
		"These always-on skills did not fit here: big3. Before your first reply, load each one with the get_skill tool of the quiver MCP server",
		"</quiver-always-on>\n")
	if strings.Contains(text, "## big3") {
		t.Error("big3 is in the block")
	}
	if log := hookLog(t); !strings.Contains(log, "skills=big1,big2 listed=big3") {
		t.Errorf("log: %s", log)
	}
}

func TestHookFailures(t *testing.T) {
	dir := hookEnv(t)
	for name, args := range map[string][]string{
		"unknown assistant": {"session-start", "nope"},
		"no assistant":      {"session-start"},
	} {
		out, errOut := runHookCmd(t, cwdInput(dir), args...)
		if out != "" || errOut == "" {
			t.Errorf("%s: stdout %q, stderr %q", name, out, errOut)
		}
	}
	// A broken registry.json: nothing is known, the session still starts.
	if err := os.WriteFile(filepath.Join(os.Getenv("QUIVER_HOME"), "registry.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	if out != "" || !strings.Contains(errOut, "build the block") {
		t.Errorf("broken registry: stdout %q, stderr %q", out, errOut)
	}
	if log := hookLog(t); !strings.Contains(log, "level=ERROR") || !strings.Contains(log, `msg="build the block"`) {
		t.Errorf("log: %s", log)
	}
}

func TestHookNothingAlwaysOn(t *testing.T) {
	dir := assignEnv(t, "pdf")
	if _, err := runIn(t, "", "add", "pdf", "--global"); err != nil {
		t.Fatal(err)
	}
	if out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code"); out != "" || errOut != "" {
		t.Errorf("stdout %q, stderr %q", out, errOut)
	}
}
