package cli

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/registry"
	"github.com/kheif/quiver/internal/server"
	"github.com/spf13/cobra"
)

// hookLogFile is the hook's log inside the Quiver home: one line per run.
const hookLogFile = "logs/hook.log"

// maxHookLog is the size at which the hook log is moved to hook.log.1.
const maxHookLog = 4 << 20

// hookEvent is a hook command: the subcommand and the assistants' name for
// its event.
type hookEvent struct {
	use, name string
}

var (
	sessionStart  = hookEvent{"session-start", "SessionStart"}
	subagentStart = hookEvent{"subagent-start", "SubagentStart"}
)

// partStagger is how much longer each part waits than the one before it, for
// assistants that run their hook commands in parallel (docs/spike-hooks.md).
// The first staggerSteps parts wait that long each; later parts wait
// partStaggerLate each, and no part waits more than maxStagger, so that a
// text of hundreds of parts still arrives inside the hook's timeout. Parts
// say which one they are, so the order only has to be mostly right.
const (
	partStagger     = 50 * time.Millisecond
	staggerSteps    = 20
	partStaggerLate = 10 * time.Millisecond
	maxStagger      = 3 * time.Second
)

// stagger is how long part n waits before it prints.
func stagger(part int) time.Duration {
	early := min(part-1, staggerSteps)
	d := time.Duration(early)*partStagger + time.Duration(max(part-1-staggerSteps, 0))*partStaggerLate
	return min(d, maxStagger)
}

// stdinWait is how long a hook waits for the assistant's JSON on stdin. An
// assistant that never closes it must not hold the session start.
const stdinWait = 2 * time.Second

// hookFormat is how an assistant wants the always-on block (docs/spike-hooks.md).
type hookFormat struct {
	output func(event, text string) any
	// subagents is whether the assistant has a sub-agent event whose
	// output reaches the sub-agent.
	subagents bool
	// parallel is whether the assistant runs the commands of a hook event at
	// the same time and adds their output in the order they finish.
	parallel bool
}

// nested is Claude Code's form, which Codex and VS Code's Local harness read.
func nested(event, text string) map[string]any {
	return map[string]any{"hookSpecificOutput": map[string]any{"hookEventName": event, "additionalContext": text}}
}

var hookFormats = map[string]hookFormat{
	claudeCode: {func(e, t string) any { return nested(e, t) }, true, true},
	codexID:    {func(e, t string) any { return nested(e, t) }, true, true},
	// VS Code's Local harness reads the nested form, its Copilot harness
	// only a top-level additionalContext.
	"vscode": {func(e, t string) any {
		out := nested(e, t)
		out["additionalContext"] = t
		return out
	}, true, true},
	"cursor": {func(_, t string) any { return map[string]any{"additional_context": t} }, false, false},
	// Antigravity runs SessionStart for every sub-agent too.
	"antigravity": {func(_, t string) any {
		return map[string]any{"injectSteps": []any{map[string]any{"systemMessage": map[string]any{"systemMessage": t}}}}
	}, false, false},
}

// hookInput is the part of a hook's stdin that names the session's folder.
// Each assistant names it differently.
type hookInput struct {
	SessionID      string   `json:"session_id"`
	ConversationID string   `json:"conversation_id"` // Cursor
	Cwd            string   `json:"cwd"`
	WorkspaceRoots []string `json:"workspace_roots"` // Cursor
	WorkspacePaths []string `json:"workspacePaths"`  // Antigravity
}

func (in hookInput) folder() string {
	for _, list := range [][]string{{in.Cwd}, in.WorkspaceRoots, in.WorkspacePaths} {
		if len(list) > 0 && list[0] != "" {
			return list[0]
		}
	}
	return ""
}

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Print the always-on skills for a new session (assistants run this)",
		Long: `Print the always-on skills of a session's folder for an assistant's hook.
Assistants run these commands from their hooks when a session or a
sub-agent starts; you do not need to run them yourself.

The command reads the hook's JSON on stdin, finds the project of the
session's folder (or of $QUIVER_PROJECT) and prints the text of its
always-on skills, global ones first, and then the id and description of its
other skills, in the assistant's format. An assistant passes only so much
text from one hook, so the text is split into parts: the hook of part N
prints part N, and quiver install gives the assistant as many hooks as the
text takes. It never fails a session: on an error it prints what it knows and
logs to ~/.quiver/logs/hook.log.`,
		Example: `  # What Claude Code's SessionStart hook runs
  echo '{"cwd": "'"$PWD"'"}' | quiver hook session-start claude-code`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		// A hook must not create the Quiver home or fail because of it.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
	}
	for _, ev := range []hookEvent{sessionStart, subagentStart} {
		cmd.AddCommand(&cobra.Command{
			Use:   ev.use + " <assistant> [part]",
			Short: fmt.Sprintf("Print the always-on skills for %s", ev.name),
			Long: fmt.Sprintf(`Print the always-on skills for the assistant's %s hook, in its
format. <assistant> is one of: %s. [part] is the number of the part to
print, 1 when it is left out. The hook's JSON comes on stdin; its folder is
the session's.`, ev.name, strings.Join(hookAssistants(), ", ")),
			Example: fmt.Sprintf(`  # What Claude Code's %s hook runs
  echo '{"cwd": "'"$PWD"'"}' | quiver hook %s claude-code`, ev.name, ev.use),
			ValidArgs: hookAssistants(),
			RunE: func(cmd *cobra.Command, args []string) error {
				runHook(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), ev, args)
				return nil
			},
		})
	}
	return cmd
}

func hookAssistants() []string {
	names := make([]string, 0, len(hookFormats))
	for name := range hookFormats {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// runHook prints the always-on block for a hook and logs one line. It never
// returns an error: a failing hook would leave the session without the
// skills, or stop it.
func runHook(in io.Reader, out, errOut io.Writer, ev hookEvent, args []string) {
	start := time.Now()
	log, closeLog := openHookLog()
	defer closeLog()
	attrs := []any{"event", ev.name}
	fail := func(msg string, err error) {
		log.Error(msg, append(attrs, "err", err, "ms", time.Since(start).Milliseconds())...)
		fmt.Fprintf(errOut, "quiver hook: %s: %v\n", msg, err)
	}
	defer func() {
		if r := recover(); r != nil {
			fail("panic", fmt.Errorf("%v", r))
		}
	}()
	if len(args) < 1 || len(args) > 2 {
		fail("bad arguments", fmt.Errorf("want one assistant (%s) and maybe a part number, got %d arguments", strings.Join(hookAssistants(), ", "), len(args)))
		return
	}
	attrs = append(attrs, "assistant", args[0])
	format, ok := hookFormats[args[0]]
	if !ok {
		fail("unknown assistant", fmt.Errorf("%q; want one of %s", args[0], strings.Join(hookAssistants(), ", ")))
		return
	}
	part := 1
	if len(args) == 2 {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			fail("bad part", fmt.Errorf("%q; want a number from 1", args[1]))
			return
		}
		part = n
	}
	attrs = append(attrs, "part", part)
	if ev == subagentStart && !format.subagents {
		fail("no sub-agent hook", fmt.Errorf("%s has no sub-agent event that takes text", args[0]))
		return
	}

	var input hookInput
	if data, err := readStdin(in, stdinWait); err != nil {
		fail("read stdin", err)
	} else if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &input); err != nil {
			fail("parse stdin", err)
		}
	}
	// Without a folder, the session's folder is ours: assistants run hooks
	// in it.
	var notes []string
	w, err := resolveHook(input.folder())
	if err != nil {
		// The global skills are still known; the notice says the rest is not.
		fail("find the project", err)
		notes = append(notes, "the project of this folder could not be found, so only the global skills are loaded: "+err.Error())
	}
	attrs = append(attrs, "dir", w.Dir, "project", w.ProjectID())
	notes = append(notes, copyNotes(w)...)
	limit := hookLimits[args[0]]
	// The parts of one session start share the text part 1 built, so that
	// each does not read the library again, and all say the same.
	cache := newPartCache(args[0], ev.name, w.ProjectID(), cmp.Or(input.SessionID, input.ConversationID), w.Dir)
	var text string
	var count int
	var problems []string
	cached, slept := false, false
	if part > 1 && cache != nil {
		if format.parallel {
			time.Sleep(stagger(part))
			slept = true
		}
		var meta partMeta
		if text, meta, cached = cache.read(part, cacheWait); cached {
			count, problems = meta.Count, meta.Problems
		}
	}
	var sc server.SessionContext
	if !cached {
		var err error
		sc, err = server.BuildSessionContext(w.ProjectID(), limit, notes...)
		if err != nil {
			// Without the skills the session starts anyway, and says so.
			fail("build the block", err)
			sc = server.FailedContext(limit, err)
		}
		count, problems = len(sc.Parts), sc.Problems
		if part <= count {
			text = sc.Parts[part-1]
		}
		if part == 1 && count > 1 && cache != nil {
			cache.write(sc)
		}
	}
	attrs = append(attrs, "parts", count)
	if cached {
		attrs = append(attrs, "cached", true)
	}
	if part == 1 {
		attrs = append(attrs, "skills", strings.Join(sc.IDs, ","))
	}
	if len(problems) > 0 {
		attrs = append(attrs, "problems", strings.Join(problems, "; "))
	}
	if part > count {
		// Nothing for this part: fewer parts than hook commands.
		log.Info("hook", append(attrs, "ms", time.Since(start).Milliseconds())...)
		return
	}
	attrs = append(attrs, "sha256", fmt.Sprintf("%x", sha256.Sum256([]byte(text)))[:12])
	// The last hook command of an assistant that has fewer than the text
	// needs says so, and gives the assistant the rest of the commands for the
	// next session.
	last := false
	if have, err := hookParts(args[0]); err == nil && have > 0 && part == have && count > have {
		last = true
		text += fmt.Sprintf("\nNote from Quiver: this assistant has %d Quiver hook commands, but the skills text takes %d, so the rest did not load in this session. Quiver sets up the missing commands for the next session; run quiver install if it does not.\n", have, count)
		attrs = append(attrs, "missing", count-have)
	}
	// Assistants that run the hook commands at the same time add their output
	// in the order they finish. A later part waits a little longer, so the
	// parts mostly arrive in order; each says which part it is, as well.
	if format.parallel && !slept {
		time.Sleep(stagger(part))
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(format.output(ev.name, text)); err != nil {
		fail("write", err)
		return
	}
	if last {
		if present, current, err := hookInstalled(args[0]); err == nil && present && current {
			if bin, err := hookBin(args[0]); err == nil {
				if _, err := installHooks(args[0], bin, count); err != nil {
					fail("add hook commands", err)
				}
			}
		}
	}
	log.Info("hook", append(attrs, "ms", time.Since(start).Milliseconds())...)
}

// readStdin reads the hook's input, and gives up after wait: a hook that
// waited for an assistant that does not close its stdin would lose its text
// to the hook's timeout.
func readStdin(in io.Reader, wait time.Duration) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(io.LimitReader(in, 1<<20))
		done <- result{data, err}
	}()
	select {
	case r := <-done:
		return r.data, r.err
	case <-time.After(wait):
		return nil, fmt.Errorf("no input after %s", wait)
	}
}

// copyNotes warns when the session's folder shares its project ID with
// another folder: a copied or forked repository keeps the .quiver.json of
// the original and so gets the original's skills.
func copyNotes(w server.Where) []string {
	if w.Project == nil {
		return nil
	}
	r, err := registry.Load()
	if err != nil {
		return nil
	}
	others := r.OtherFolders(w.Project.ID, w.Project.Root)
	if len(others) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%s has the project ID %s, which %s also has. If this folder is a copy, it gets the skills of the original; run quiver projects split in it", w.Project.Root, w.Project.ID, strings.Join(others, ", "))}
}

func resolveHook(dir string) (server.Where, error) {
	if dir == "" {
		return server.Resolve()
	}
	return server.ResolveIn(dir, "hook input")
}

// openHookLog returns a logger that appends to hookLogFile and a function
// that closes the file. When the file cannot be opened, the logger
// discards: the hook's output must not depend on it.
func openHookLog() (*slog.Logger, func()) {
	discard := slog.New(slog.NewTextHandler(io.Discard, nil))
	home, err := fsutil.QuiverHome()
	if err != nil {
		return discard, func() {}
	}
	// A hook never creates the Quiver home: on a machine that does not use
	// Quiver, it leaves nothing behind.
	if info, err := os.Stat(home); err != nil || !info.IsDir() {
		return discard, func() {}
	}
	path := filepath.Join(home, filepath.FromSlash(hookLogFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return discard, func() {}
	}
	// The log keeps one earlier file, so it does not grow without end.
	if info, err := os.Stat(path); err == nil && info.Size() > maxHookLog {
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return discard, func() {}
	}
	return slog.New(slog.NewTextHandler(f, nil)), func() { _ = f.Close() }
}
