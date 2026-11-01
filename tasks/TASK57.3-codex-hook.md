# TASK57.3: Codex hook

**Phase:** Phase 6: Session-start delivery

**Part of:** [TASK57](TASK57-quiver-install.md): `quiver install`

**Status:** Todo

**Depends on:** TASK54.2, TASK55, TASK57.1

1. **Where.** Codex's `hooks.json` (`~/.codex/hooks.json`, or `$CODEX_HOME/hooks.json`). The Codex CLI and app share it.
2. **What.** A `SessionStart` entry whose command is `<quiver> hook session-start codex`, `<quiver>` being the binary itself (`appCommand()`), since the Codex app has no shell `PATH`.
3. **How.** The JSON editing of TASK57.2, with the same rules for other hooks, backups and recognition.
4. **Trust.** Apply the trust step found in TASK54.2. If a new hook cannot be trusted without the user, `install` says what the user must do and `doctor` reports the hook as not trusted.
5. **Tests.** The fake codex (`assistanttest.InstallCodex`) covers the `codex` calls in `internal/cli`; a test like `TestRealCodex` in `internal/assistant/codex_test.go` checks the real codex in a throwaway `CODEX_HOME` when it is installed.

**Done when:** the tests pass, and a new Codex session follows an always-on fixture skill with no `get_skill` call for it.
