package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/kheif/quiver/internal/assistant"
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

// writeSkillFile gives a local skill a description.
func writeSkillFile(t *testing.T, id, desc string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("QUIVER_HOME"), "local", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + id + "\ndescription: " + desc + "\n---\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
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
	if strings.Count(text, "## alpha") != 1 || strings.Contains(text, "## gone") {
		t.Errorf("alpha twice or the removed skill in:\n%s", text)
	}
	// The removed skill is still assigned: the session is told it is missing.
	if !strings.Contains(text, "<quiver-notice>") || !strings.Contains(text, "- gone: it is assigned, but not in the library\n") {
		t.Errorf("no notice about the missing skill in:\n%s", text)
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
			if !strings.Contains(text, "## beta\n") || !strings.Contains(text, "<quiver-always-on>") {
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

// Always-on skills past a hook's limit arrive whole in more parts: the hook
// of part N prints part N, and a part past the last prints nothing.
func TestHookParts(t *testing.T) {
	dir := assignEnv(t, "big1", "big2", "big3")
	// Characters, not bytes, count for Claude Code: each body is 4,000
	// characters and 8,000 bytes.
	for _, id := range []string{"big1", "big2", "big3"} {
		writeBody(t, id, "START-"+id+"\n"+strings.Repeat("ş", 3980)+"\nEND-"+id+"\n")
	}
	if _, err := runIn(t, "", "add", "big1", "big2", "big3", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	var all string
	for part, args := range [][]string{
		{"session-start", "claude-code"}, // part 1 has no number
		{"session-start", "claude-code", "2"},
	} {
		out, errOut := runHookCmd(t, cwdInput(dir), args...)
		if errOut != "" {
			t.Errorf("part %d: stderr %s", part+1, errOut)
		}
		_, text := nestedContext(t, out)
		if n := utf8.RuneCountInString(text); n == 0 || n > hookLimits[claudeCode].Max {
			t.Errorf("part %d has %d characters", part+1, n)
		}
		if want := fmt.Sprintf(`part="%d/2"`, part+1); !strings.Contains(text, want) {
			t.Errorf("part %d lacks %s:\n%.200s", part+1, want, text)
		}
		all += text
	}
	for _, id := range []string{"big1", "big2", "big3"} {
		if !strings.Contains(all, "START-"+id+"\n") || !strings.Contains(all, "END-"+id+"\n") {
			t.Errorf("%s is not whole in:\n%.300s", id, all)
		}
	}
	wantOrder(t, all, "## big1\n", "## big2\n", "## big3\n", "get_skill_file tool")
	if !strings.HasSuffix(strings.TrimSpace(all), "</quiver-always-on>") {
		t.Errorf("the last part does not close the section:\n%s", all[len(all)-200:])
	}
	// Past the last part: nothing, and no error.
	if out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code", "3"); out != "" || errOut != "" {
		t.Errorf("part 3: stdout %q, stderr %q", out, errOut)
	}
	if log := hookLog(t); !strings.Contains(log, "part=2") || !strings.Contains(log, "parts=2") || !strings.Contains(log, "skills=big1,big2,big3") {
		t.Errorf("log: %s", log)
	}
	// The sub-agent hook passes the same parts.
	out, _ := runHookCmd(t, cwdInput(dir), "subagent-start", "claude-code", "2")
	if event, text := nestedContext(t, out); event != "SubagentStart" || !strings.Contains(text, `part="2/2"`) {
		t.Errorf("subagent: %q %.100s", event, text)
	}
	// An assistant with a larger limit takes it in one.
	out, _ = runHookCmd(t, cwdInput(dir), "session-start", "vscode")
	var vs struct {
		AdditionalContext string `json:"additionalContext"`
	}
	if err := json.Unmarshal([]byte(out), &vs); err != nil || strings.Contains(vs.AdditionalContext, "part=") ||
		!strings.Contains(vs.AdditionalContext, "END-big3\n") {
		t.Errorf("vscode: %v %.200s", err, vs.AdditionalContext)
	}
}

// The other skills' ids and descriptions follow the always-on skills in the
// hook's text.
func TestHookCatalog(t *testing.T) {
	dir := hookEnv(t)
	writeSkillFile(t, "pdf", "Reads PDF files.")
	if _, err := runIn(t, "", "add", "pdf", "--global"); err != nil {
		t.Fatal(err)
	}
	out, _ := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	wantOrder(t, text, "</quiver-always-on>\n", "<quiver-skills>\n", "- pdf: Reads PDF files.\n", "</quiver-skills>\n")
	if strings.Contains(text, "- alpha:") {
		t.Errorf("an always-on skill is in the catalog:\n%s", text)
	}
}

// A hook that is not the last of its assistant's commands prints its part and
// nothing else. The last one tells when the text needs more commands than the
// assistant has, and adds them for the next session.
func TestHookShortfall(t *testing.T) {
	dir := assignEnv(t, "big1", "big2", "big3")
	for _, id := range []string{"big1", "big2", "big3"} {
		writeBody(t, id, "START-"+id+"\n"+strings.Repeat("x", 6000)+"\nEND-"+id+"\n")
	}
	if _, err := runIn(t, "", "add", "big1", "big2", "big3", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	bin := "/opt/bin/quiver"
	fakeExecutable(t, bin)
	if _, err := assistant.InstallClaudeHooks(bin, 1); err != nil {
		t.Fatal(err)
	}
	out, _ := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	if !strings.Contains(text, "Note from Quiver: this assistant has 1 Quiver hook commands, but the skills text takes 3") {
		t.Errorf("no note in:\n%s", text[len(text)-300:])
	}
	if n, err := assistant.ClaudeHookParts(); err != nil || n != 3 {
		t.Errorf("hook commands after the run: %d, %v\n%s", n, err, hookLog(t))
	}
	// With all three, no note.
	out, _ = runHookCmd(t, cwdInput(dir), "session-start", "claude-code", "3")
	if _, text := nestedContext(t, out); strings.Contains(text, "Note from Quiver") {
		t.Errorf("note in the last part:\n%s", text)
	}
}

func TestHookBadPart(t *testing.T) {
	dir := hookEnv(t)
	for _, part := range []string{"0", "x", "1.5"} {
		out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code", part)
		if out != "" || !strings.Contains(errOut, "bad part") {
			t.Errorf("part %s: stdout %q, stderr %q", part, out, errOut)
		}
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
	// A broken registry.json: nothing is known, the session still starts and is told.
	if err := os.WriteFile(filepath.Join(os.Getenv("QUIVER_HOME"), "registry.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	if !strings.Contains(errOut, "build the block") || !strings.Contains(text, "<quiver-notice>") || !strings.Contains(text, "Quiver could not read your skills") {
		t.Errorf("broken registry: stdout %q, stderr %q", out, errOut)
	}
	// Only part 1 says so.
	if out, _ := runHookCmd(t, cwdInput(dir), "session-start", "claude-code", "2"); out != "" {
		t.Errorf("part 2 of a failed build: %q", out)
	}
	if log := hookLog(t); !strings.Contains(log, "level=ERROR") || !strings.Contains(log, `msg="build the block"`) {
		t.Errorf("log: %s", log)
	}
}

// Without always-on skills the hook still passes the catalog; with no skill at
// all, it passes nothing.
func TestHookNothingAlwaysOn(t *testing.T) {
	dir := assignEnv(t, "pdf")
	if out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code"); out != "" || errOut != "" {
		t.Errorf("no skill served: stdout %q, stderr %q", out, errOut)
	}
	if _, err := runIn(t, "", "add", "pdf", "--global"); err != nil {
		t.Fatal(err)
	}
	out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	if errOut != "" || strings.Contains(text, "quiver-always-on") || !strings.Contains(text, "- pdf: Test skill.\n") {
		t.Errorf("catalog only: stderr %q, text:\n%s", errOut, text)
	}
}

func TestStagger(t *testing.T) {
	for part, want := range map[int]time.Duration{
		1:    0,
		2:    50 * time.Millisecond,
		21:   time.Second,
		22:   time.Second + 10*time.Millisecond,
		100:  time.Second + 79*10*time.Millisecond,
		230:  maxStagger,
		5000: maxStagger,
	} {
		if got := stagger(part); got != want {
			t.Errorf("stagger(%d) = %v, want %v", part, got, want)
		}
	}
}

// An assistant that never closes the hook's stdin must not hold the session.
func TestReadStdinGivesUp(t *testing.T) {
	pr, pw := io.Pipe()
	defer func() { _ = pw.Close() }()
	start := time.Now()
	if _, err := readStdin(pr, 50*time.Millisecond); err == nil {
		t.Error("want an error when no input comes")
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("took %v", time.Since(start))
	}
	got, err := readStdin(strings.NewReader(`{"cwd":"/x"}`), time.Second)
	if err != nil || string(got) != `{"cwd":"/x"}` {
		t.Errorf("got %q, %v", got, err)
	}
}

// A copy of a project folder has the same .quiver.json, so the hook gives it
// the original's skills; the session is told.
func TestHookCopiedProjectIsNamed(t *testing.T) {
	dir := hookEnv(t)
	// A session in the project records its folder.
	if _, err := runIn(t, dir, "status"); err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(t.TempDir(), "copy")
	if err := os.MkdirAll(clone, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".quiver.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clone, ".quiver.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	out, _ := runHookCmd(t, cwdInput(clone), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	if !strings.Contains(text, "<quiver-notice>") || !strings.Contains(text, "quiver projects split") {
		t.Errorf("no notice about the copy:\n%s", text)
	}
	// The original is not told.
	out, _ = runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	if _, text := nestedContext(t, out); strings.Contains(text, "quiver projects split") {
		t.Errorf("the original got the notice:\n%s", text)
	}
}

// A .quiver.json that is not a small regular file is refused, and the session
// still gets the global skills and a notice.
func TestHookBadProjectFileIsNamed(t *testing.T) {
	dir := hookEnv(t)
	if err := os.WriteFile(filepath.Join(dir, ".quiver.json"), []byte(strings.Repeat("x", 100000)), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut := runHookCmd(t, cwdInput(dir), "session-start", "claude-code")
	_, text := nestedContext(t, out)
	if !strings.Contains(errOut, "find the project") || !strings.Contains(text, "only the global skills are loaded") || !strings.Contains(text, "## zeta\n") {
		t.Errorf("stderr %q, text:\n%s", errOut, text)
	}
}

func sessionInput(dir, session string) string {
	b, _ := json.Marshal(map[string]string{"cwd": dir, "hook_event_name": "SessionStart", "session_id": session})
	return string(b)
}

// The parts of one session start say the same, even when a skill is edited
// between them: part 1 builds the text and the later parts take theirs from
// it. Another session does not get it.
func TestHookPartsShareOneSnapshot(t *testing.T) {
	dir := assignEnv(t, "biga", "bigb", "bigc")
	bigSkills(t, 3)
	if _, err := runIn(t, "", "add", "biga", "bigb", "bigc", "--global", "--always"); err != nil {
		t.Fatal(err)
	}
	part := func(session string, n int) string {
		t.Helper()
		out, errOut := runHookCmd(t, sessionInput(dir, session), "session-start", "claude-code", strconv.Itoa(n))
		if out == "" {
			t.Fatalf("session %s part %d: nothing (%s)", session, n, errOut)
		}
		_, text := nestedContext(t, out)
		return text
	}
	part("ref", 1)
	want := part("ref", 2)

	part("s1", 1)
	writeBody(t, "biga", "CHANGED\n")
	if got := part("s1", 2); got != want {
		t.Errorf("part 2 of the session differs from its part 1's snapshot:\n%s", got)
	}
	if log := hookLog(t); !strings.Contains(log, "cached=true") {
		t.Errorf("part 2 did not use the cache:\n%s", log)
	}
	// A session whose part 1 did not run builds its own, from the edited
	// library, where the text no longer has a part 2.
	if out, _ := runHookCmd(t, sessionInput(dir, "s2"), "session-start", "claude-code", "2"); out != "" {
		t.Errorf("a session without part 1 got a text: %s", out)
	}
	// A new session start rebuilds: part 1 sees the edit.
	if text := part("s1", 1); !strings.Contains(text, "CHANGED") {
		t.Errorf("part 1 is old:\n%s", text)
	}
}

// The hook does not create the Quiver home.
func TestHookLeavesNoHomeBehind(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent")
	t.Setenv("QUIVER_HOME", home)
	out, _ := runHookCmd(t, `{"cwd":"`+t.TempDir()+`"}`, "session-start", "claude-code")
	if out != "" {
		t.Errorf("stdout %q", out)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("the hook created %s: %v", home, err)
	}
}
