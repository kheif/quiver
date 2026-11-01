# TASK54.1: Claude Code hooks

**Phase:** Phase 6: Session-start delivery

**Part of:** [TASK54](TASK54-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** Todo

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
