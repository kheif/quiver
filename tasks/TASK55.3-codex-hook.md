# TASK55.3: Codex hook

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK55](TASK55-quiver-install.md): `quiver install`

**Status:** Todo

**Depends on:** TASK52.2, TASK53, TASK55.1

1. **Where.** Codex's `hooks.json` (`~/.codex/hooks.json`, or `$CODEX_HOME/hooks.json`). The Codex CLI and app share it.
2. **What.** A `SessionStart` entry with the matcher `startup|clear|compact`, whose command is `<quiver> hook session-start codex`, and a `SubagentStart` entry with no matcher, whose command is `<quiver> hook subagent-start codex`. `<quiver>` is the binary itself (`appCommand()`), since the Codex app has no shell `PATH`, quoted with `assistant.ShellQuote`. A sub-agent that starts with its parent's history, Codex's default, then has the text twice; one that starts without it gets the text only from `SubagentStart` (TASK52.2).
3. **Fields.** Set each hook's `timeout` to 10 seconds; `additionalContextLimit` to 0, so Codex does not cut the block (TASK53 keeps it within its limit); and `statusMessage` to `Quiver always-on skills`, the name Codex's review list shows.
4. **How.** The JSON editing of TASK55.2, with the same rules for other hooks, backups and recognition.
5. **Trust.** Codex runs a new or changed hook only once it is trusted, and `codex exec` skips it without a message (TASK52.2). After writing the file, trust Quiver's own hooks, and only those, through `codex app-server`: `hooks/list` for their keys and current hashes, then `config/batchWrite` of `hooks.state.<key>.trusted_hash`, the two requests Codex's own review dialog sends. `install` says that it trusted them. Every change to a hook's fields or command changes its hash, so trust again whenever `install` updates the hook. If the app server fails, print the step for the user: start `codex`, open "Review hooks" and trust the Quiver hooks. `doctor` reads the status through `hooks/list` and reports a Quiver hook that is new or changed, for example after the binary moved.
6. **Tests.** The fake codex (`assistanttest.InstallCodex`) covers the `codex` calls in `internal/cli`, the app server included; a test like `TestRealCodex` in `internal/assistant/codex_test.go` checks the real codex in a throwaway `CODEX_HOME` when it is installed, including that `codex exec` runs the hooks right after `install`.

**Done when:** the tests pass, and right after `install`, with no review dialog, a new Codex session and a sub-agent it starts with `fork_turns` `none` both follow an always-on fixture skill with no `get_skill` call for it.
