# TASK57.2: Claude Code hook

**Phase:** Phase 6: Session-start delivery

**Part of:** [TASK57](TASK57-quiver-install.md): `quiver install`

**Status:** Todo

**Depends on:** TASK54.1, TASK55, TASK57.1

1. **Where.** The user's Claude Code settings file found in TASK54.1 (`~/.claude/settings.json`, or `$CLAUDE_CONFIG_DIR/settings.json`).
2. **What.** A `SessionStart` entry whose command is `<quiver> hook session-start claude-code`, `<quiver>` being the path `command()` in `internal/cli/mcp.go` returns (Claude Code has the shell's `PATH`, so the npm launcher works), quoted with `assistant.ShellQuote`.
3. **How.** Generalize `internal/assistant/jsonconfig.go`: today `change` edits the `mcpServers` object; the hook lives in `hooks.SessionStart`, an array of groups that each hold a `hooks` array. Keep every other key, group and hook, the key order, the indentation and the file mode; save `<file>.quiver-backup` first; write atomically (`fsutil.WriteFileAtomic`); never write a file that is not valid JSON.
4. **Recognize Quiver's hook** by its command: one whose arguments end with `hook session-start claude-code` and whose program is named `quiver` (or `quiver.exe`). `install` points it to the current binary (reported as updated), adds it if missing, and leaves it alone if it is current; `uninstall` removes only it, and the group if it is then empty.
5. Set the hook's timeout to the value found in TASK54.1.

**Done when:** tests in `internal/assistant` cover a missing file, a file with other settings and hooks, an update after a move, removal, and a file that is not valid JSON; tests in `internal/cli` cover install and uninstall; and a new Claude Code session follows an always-on fixture skill from the hook.
