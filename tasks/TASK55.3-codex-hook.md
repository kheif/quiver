# TASK55.3: Codex hook

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** ✅ Done

**Depends on:** TASK52.2, TASK53, TASK55.1

1. **Where.** Codex's `hooks.json` (`~/.codex/hooks.json`, or `$CODEX_HOME/hooks.json`). The Codex CLI and app share it.
2. **What.** A `SessionStart` entry with the matcher `startup|clear|compact`, whose command is `<quiver> hook session-start codex`, and a `SubagentStart` entry with no matcher, whose command is `<quiver> hook subagent-start codex`. `<quiver>` is the binary itself (`appCommand()`), since the Codex app has no shell `PATH`, quoted with `assistant.ShellQuote`. A sub-agent that starts with its parent's history, Codex's default, then has the text twice; one that starts without it gets the text only from `SubagentStart` (TASK52.2).
3. **Fields.** Set each hook's `timeout` to 10 seconds; `additionalContextLimit` to 0, so Codex does not cut the block (TASK53 keeps it within its limit); and `statusMessage` to `Quiver always-on skills`, the name Codex's review list shows.
4. **How.** The JSON editing of TASK55.2, with the same rules for other hooks, backups and recognition.
5. **Trust.** Codex runs a new or changed hook only once it is trusted, and `codex exec` skips it without a message (TASK52.2). After writing the file, trust Quiver's own hooks, and only those, through `codex app-server`: `hooks/list` for their keys and current hashes, then `config/batchWrite` of `hooks.state.<key>.trusted_hash`, the two requests Codex's own review dialog sends. `install` says that it trusted them. Every change to a hook's fields or command changes its hash, so trust again whenever `install` updates the hook. If the app server fails, print the step for the user: start `codex`, open "Review hooks" and trust the Quiver hooks. `doctor` reads the status through `hooks/list` and reports a Quiver hook that is new or changed, for example after the binary moved.
6. **Tests.** The fake codex (`assistanttest.InstallCodex`) covers the `codex` calls in `internal/cli`, the app server included; a test like `TestRealCodex` in `internal/assistant/codex_test.go` checks the real codex in a throwaway `CODEX_HOME` when it is installed, including that `codex exec` runs the hooks right after `install`.

**Done when:** the tests pass, and right after `install`, with no review dialog, a new Codex session and a sub-agent it starts with `fork_turns` `none` both follow an always-on fixture skill with no `get_skill` call for it.

**Notes:**
- `internal/assistant/hooks.go` was generalized: a `hookTarget{path, id, extra}` writes the same `SessionStart`/`SubagentStart` group shape for any assistant. Codex's target (`CodexHooksPath` → `$CODEX_HOME/hooks.json` or `~/.codex/hooks.json`) adds `additionalContextLimit: 0` and `statusMessage: "Quiver always-on skills"` to each entry; `timeout` is 10. `InstallCodexHooks`/`UninstallCodexHooks` reuse the shared install/uninstall.
- `internal/assistant/codexhooks.go` trusts the hooks through `codex app-server`: `initialize`, `initialized`, `hooks/list {cwds}`, then `config/batchWrite` of `hooks.state.<key>.trusted_hash` for Quiver's own hooks that are `untrusted` or `modified` (recognised by their command). `CodexTrustHooks` returns how many it trusted; `CodexHooksNeedTrust` lists the ones still new or changed, for `doctor`.
- `installCodex` writes the hooks and then trusts them; if the app server fails it prints the manual "Review hooks" step and does not fail. `mcpUninstall` removes them. `doctor` warns when Quiver's Codex hooks are new or changed (for example after the binary moved).
- Tests: `assistanttest.InstallCodex` now ships a fake `codex app-server` (a small Python helper) that answers `hooks/list` and `config/batchWrite`, so `internal/cli` `TestInstallCodexHook` covers install and uninstall. `TestRealCodexTrust` in `internal/assistant/codex_test.go` drives the real `codex app-server` in a throwaway `CODEX_HOME` (copying the login): install → 2 untrusted → trust → 0 need, then a move → 2 modified → trust → 0 need.
- Live (codex-cli 0.166.0): after `quiver install codex` (MCP registered, hooks written, both trusted, `hooks.state` in `config.toml`), one `codex exec` with a `fork_turns "none"` sub-agent — the main session's `SessionStart` and the sub-agent's `SubagentStart` hooks both fired, both rollouts carried the `<quiver-always-on>` block, both replies ended with `[always-on]`, and there was no `get_skill` function call (the only `get_skill` text was the server's instructions pointer in a developer message).
