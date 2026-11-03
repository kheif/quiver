package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kheif/quiver/internal/fsutil"
	"github.com/kheif/quiver/internal/server"
	"github.com/spf13/cobra"
)

// hookLogFile is the hook's log inside the Quiver home: one line per run.
const hookLogFile = "logs/hook.log"

// hookEvent is a hook command: the subcommand and the assistants' name for
// its event.
type hookEvent struct {
	use, name string
}

var (
	sessionStart  = hookEvent{"session-start", "SessionStart"}
	subagentStart = hookEvent{"subagent-start", "SubagentStart"}
)

// hookFormat is how an assistant wants the always-on block (docs/spike-hooks.md).
type hookFormat struct {
	output func(event, text string) any
	// subagents is whether the assistant has a sub-agent event whose
	// output reaches the sub-agent.
	subagents bool
}

// nested is Claude Code's form, which Codex and VS Code's Local harness read.
func nested(event, text string) map[string]any {
	return map[string]any{"hookSpecificOutput": map[string]any{"hookEventName": event, "additionalContext": text}}
}

var hookFormats = map[string]hookFormat{
	claudeCode: {func(e, t string) any { return nested(e, t) }, true},
	codexID:    {func(e, t string) any { return nested(e, t) }, true},
	// VS Code's Local harness reads the nested form, its Copilot harness
	// only a top-level additionalContext.
	"vscode": {func(e, t string) any {
		out := nested(e, t)
		out["additionalContext"] = t
		return out
	}, true},
	"cursor": {func(_, t string) any { return map[string]any{"additional_context": t} }, false},
	// Antigravity runs SessionStart for every sub-agent too.
	"antigravity": {func(_, t string) any {
		return map[string]any{"injectSteps": []any{map[string]any{"systemMessage": map[string]any{"systemMessage": t}}}}
	}, false},
}

// hookInput is the part of a hook's stdin that names the session's folder.
// Each assistant names it differently.
type hookInput struct {
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
always-on skills, global ones first, in the assistant's format. It never
fails a session: on an error it prints what it knows and logs to
~/.quiver/logs/hook.log.`,
		Example: `  # What Claude Code's SessionStart hook runs
  echo '{"cwd": "'"$PWD"'"}' | quiver hook session-start claude-code`,
		ValidArgsFunction: noArgs,
		Args:              cobra.NoArgs,
		// A hook must not create the Quiver home or fail because of it.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
	}
	for _, ev := range []hookEvent{sessionStart, subagentStart} {
		cmd.AddCommand(&cobra.Command{
			Use:   ev.use + " <assistant>",
			Short: fmt.Sprintf("Print the always-on skills for %s", ev.name),
			Long: fmt.Sprintf(`Print the always-on skills for the assistant's %s hook, in its
format. <assistant> is one of: %s. The hook's JSON comes on
stdin; its folder is the session's.`, ev.name, strings.Join(hookAssistants(), ", ")),
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
	if len(args) != 1 {
		fail("bad arguments", fmt.Errorf("want one assistant (%s), got %d arguments", strings.Join(hookAssistants(), ", "), len(args)))
		return
	}
	attrs = append(attrs, "assistant", args[0])
	format, ok := hookFormats[args[0]]
	if !ok {
		fail("unknown assistant", fmt.Errorf("%q; want one of %s", args[0], strings.Join(hookAssistants(), ", ")))
		return
	}
	if ev == subagentStart && !format.subagents {
		fail("no sub-agent hook", fmt.Errorf("%s has no sub-agent event that takes text", args[0]))
		return
	}

	var input hookInput
	if data, err := io.ReadAll(in); err != nil {
		fail("read stdin", err)
	} else if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &input); err != nil {
			fail("parse stdin", err)
		}
	}
	// Without a folder, the session's folder is ours: assistants run hooks
	// in it.
	w, err := resolveHook(input.folder())
	if err != nil {
		// The global skills are still known.
		fail("find the project", err)
	}
	attrs = append(attrs, "dir", w.Dir, "project", w.ProjectID())
	block, err := server.AlwaysOnBlock(w.ProjectID())
	if err != nil {
		fail("build the block", err)
		return
	}
	if block.Text != "" {
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(format.output(ev.name, block.Text)); err != nil {
			fail("write", err)
			return
		}
	}
	log.Info("hook", append(attrs, "skills", strings.Join(block.IDs, ","), "listed", strings.Join(block.Listed, ","), "ms", time.Since(start).Milliseconds())...)
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
	path := filepath.Join(home, filepath.FromSlash(hookLogFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return discard, func() {}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return discard, func() {}
	}
	return slog.New(slog.NewTextHandler(f, nil)), func() { _ = f.Close() }
}
