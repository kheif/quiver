# TASK52.1: Claude Code hooks

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK52](TASK52-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** ✅ Done

**Depends on:** TASK50

Facts to confirm against the current Claude Code docs before measuring:
- Hooks are set in settings files: `~/.claude/settings.json` (or `$CLAUDE_CONFIG_DIR/settings.json`) for the user, `.claude/settings.json` in a project, and a managed settings file for companies.
- The entry has this shape:
  ```json
  {"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "/path/to/quiver hook session-start claude-code"}]}]}}
  ```
- The hook gets JSON on stdin with `session_id`, `cwd`, `hook_event_name` and `source` (`startup`, `resume`, `clear` or `compact`). For `SessionStart`, its stdout is added to the context, or the JSON `{"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": "..."}}`.

Steps, in a throwaway workspace with the hook in its `.claude/settings.json`:
1. The hook prints `reply-style` in full. Run the 10 prompts with `claude -p` and count `[always-on]` and `get_skill` calls for `reply-style` (expected: none).
2. Compare plain stdout with `additionalContext`. Find the size limit: print 5, 10, 20 and 50 KB and check whether the end of the text still reaches the model.
3. Check each `source`: is the text there after `/compact`, after `/clear`, and in a resumed session?
4. Sub-agents: does a sub-agent started with the Task tool see the text? Is there an event that can add context to a sub-agent?
5. Managed settings: find the file's path on macOS, Linux and Windows, whether its hooks run next to the user's, and whether a user can turn them off.
6. Time: how long may the hook run before Claude Code gives up, and does a slow hook delay the first reply?

**Done when:** the results are in `docs/spike-hooks.md`.

**Notes:**
- Results in docs/spike-hooks.md (Claude Code 2.1.308, `spike/b01/claude-hooks-run.sh`). Delivery level: hook. Always-on 10 of 10, fitting skills 6 of 6, other skills 0 of 4, with plain stdout and with `additionalContext` alike, and no `get_skill` for `reply-style`. With the server's pointer also active, the model still loaded it in 6 of 10.
- Limit: 10,000 characters per hook command, counted in characters. Above it the model gets a file path and a 2 KB preview, and a rule after the preview was followed in 0 of 6.
- Sources: `clear` needs the hook, or the rule is gone. On `resume` the earlier text stays and a new firing adds a second copy. The matcher `startup|clear|compact` gives every context the text once.
- `SessionStart` does not reach sub-agents (0 of 6); `SubagentStart` with `additionalContext` does (6 of 6).
- A failing hook never stops a session, and its text is dropped. A slow hook delays the first reply by its whole run time; the default timeout is 600 seconds.
- Besides the facts listed above, stdin has `transcript_path`, interactive sessions add `model` and `scratchpad_dir`, and `source` can also be `fork` (documented).
- Not measured: the user's own settings file, the managed settings file, Windows, automatic compaction, and what the terminal shows for a failed hook.
