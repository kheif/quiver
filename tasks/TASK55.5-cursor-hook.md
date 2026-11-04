# TASK55.5: Cursor hook

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** ✅ Done

**Depends on:** TASK52.4, TASK53, TASK55.1

1. **Where.** The user's `~/.cursor/hooks.json`, which the Cursor app and the CLI share; `{"version": 1, "hooks": {...}}`.
2. **What.** A `sessionStart` entry `{"command": "<quiver> hook session-start cursor", "timeout": 10}`, `<quiver>` being the binary itself (`appCommand()`) quoted with `assistant.ShellQuote`, since the app has no shell `PATH`. No sub-agent entry: Cursor's `subagentStart` cannot add text (TASK52.4).
3. **How.** The JSON editing of TASK55.2: keep every other hook and key, back up, write atomically, recognize Quiver's entry by its command, and remove only it on `uninstall`.
4. The MCP server's `instructions` stay the fallback (TASK54); with the hook, Cursor's level is hook.

**Done when:** tests cover a new file, a file with other hooks, an update after a move and removal; a Cursor CLI session follows an always-on fixture skill from the hook with no `get_skill` call for it; and a session in the Cursor app does too, checked by hand.
