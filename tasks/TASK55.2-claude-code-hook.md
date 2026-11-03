# TASK55.2: Claude Code hook

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** ✅ Done

**Depends on:** TASK52.1, TASK53, TASK55.1

1. **Where.** The user's Claude Code settings file found in TASK52.1 (`~/.claude/settings.json`, or `$CLAUDE_CONFIG_DIR/settings.json`).
2. **What.** A `SessionStart` entry with the matcher `startup|clear|compact`, whose command is `<quiver> hook session-start claude-code`, and a `SubagentStart` entry with no matcher, whose command is `<quiver> hook subagent-start claude-code`. `<quiver>` is the path `command()` in `internal/cli/mcp.go` returns (Claude Code has the shell's `PATH`, so the npm launcher works), quoted with `assistant.ShellQuote`.
3. **How.** Generalize `internal/assistant/jsonconfig.go`: today `change` edits the `mcpServers` object; the hooks live in `hooks.SessionStart` and `hooks.SubagentStart`, arrays of groups that each hold a `hooks` array. Keep every other key, group and hook, the key order, the indentation and the file mode; save `<file>.quiver-backup` first; write atomically (`fsutil.WriteFileAtomic`); never write a file that is not valid JSON.
4. **Recognize Quiver's hooks** by their command: one whose arguments end with `hook session-start claude-code` or `hook subagent-start claude-code` and whose program is named `quiver` (or `quiver.exe`). `install` points it to the current binary (reported as updated), adds it if missing, and leaves it alone if it is current; `uninstall` removes only it, and the group if it is then empty.
5. Set each hook's `timeout` to 10 seconds. The default is 600, and a slow hook delays the first reply by its whole run time (TASK52.1).

**Done when:** tests in `internal/assistant` cover a missing file, a file with other settings and hooks, an update after a move, removal, and a file that is not valid JSON; tests in `internal/cli` cover install and uninstall; and a new Claude Code session and a sub-agent it starts both follow an always-on fixture skill from the hooks.

**Notes:**
- `change` in `internal/assistant/jsonconfig.go` now delegates to a new `editJSONFile(path, f)`, which does the read, backup, atomic write and JSON validation on any file's top-level object. The MCP path and the hook path share it.
- `internal/assistant/hooks.go` adds `ClaudeSettingsPath` (`$CLAUDE_CONFIG_DIR/settings.json`, else `~/.claude/settings.json`), `InstallClaudeHooks(bin)` and `UninstallClaudeHooks()`. They edit `hooks.SessionStart` (matcher `startup|clear|compact`) and `hooks.SubagentStart` (no matcher), each group holding `{type:"command", command:"<quoted bin> hook <event> claude-code", timeout:10}`. Quiver's hook is recognised by a command whose args end with `hook <subcommand> claude-code` and whose program basename is `quiver`/`quiver.exe`; other keys, groups and hooks, the key order, indentation and mode are kept. `uninstall` removes only Quiver's hook, then an emptied group and an emptied event.
- `installClaude` adds the hooks after the MCP registration; `mcpUninstall` removes them in the Claude Code branch. Both print a line only when something changed.
- Tests: `internal/assistant/hooks_test.go` covers a missing file, a file with other settings and hooks, an update after a move, removal, invalid JSON and a no-op re-install; `internal/cli` `TestInstallClaudeHook` covers install and uninstall through the CLI.
- Live (Claude Code 2.1.308): `quiver install claude-code` wrote the two hooks; a session's SessionStart hook fired and injected the `<quiver-always-on>` block, and the main reply ended with `[always-on]`. With the same hook block in a project `.claude/settings.json` (real login, `--setting-sources project`, so the real user config was untouched), a session and a sub-agent it started both got the block and ended their replies with `[always-on]` (SubagentStart fired, the sub-agent transcript carried the block). The isolated-`CLAUDE_CONFIG_DIR` run could not reach the model because the keychain login is not portable to a throwaway config dir.
