# TASK55.6: Antigravity hook

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** ✅ Done

**Depends on:** TASK52.4, TASK53, TASK55.1

1. **Where.** `~/.gemini/config/hooks.json`, the global customization root that every Antigravity product reads (TASK52.4).
2. **What.** A hook named `quiver`: `{"quiver": {"SessionStart": [{"type": "command", "command": "<quiver> hook session-start antigravity", "timeout": 10}]}}`, `<quiver>` being the binary itself (`appCommand()`) quoted with `assistant.ShellQuote`. `SessionStart` also runs for every sub-agent, so no second entry is needed.
3. **How.** Quiver owns the `quiver` key and keeps every other named hook and key; back up, write atomically, and remove only that key on `uninstall`.
4. **Doctor.** Reports the hook as missing, outdated or pointing to another binary.

**Done when:** tests cover a new file, a file with other hooks, an update and removal, and an `agy` session follows an always-on fixture skill from the hook with no `get_skill` call for it.
