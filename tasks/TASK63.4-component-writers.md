# TASK63.4: Component writers

**Phase:** Phase 7c: Tool bundles (v1.7)

**Part of:** [TASK63](TASK63-tool-bundles.md): Tool bundles

**Status:** Todo

**Depends on:** TASK63.3

One function per component type that writes it for an assistant and a scope, records it in the ledger, and can reverse it. They reuse what exists; nothing here parses an assistant's config a second way.

1. **MCP.** The shared JSON installer of TASK37 (`internal/assistant/jsonconfig.go`), extended with a scope: the user's config, or the project's (`.mcp.json`, `.cursor/mcp.json`, VS Code's `.vscode/mcp.json`; the names come from TASK63.1). The entry's name is prefixed with the bundle's ID so it cannot clash with the user's.
2. **Hooks.** The hook installers of TASK55 for each assistant, with the same safety: other hooks stay, key order and indentation stay, the file is backed up, invalid JSON is never written, a wrapped or edited hook is recognized. Quiver's own hook is recognized by its command (`isQuiverHook` in `internal/assistant/hooks.go`); a bundle's hook is another command and is recognized only through the ledger, so Quiver's install and uninstall never touch it and a bundle's removal never touches Quiver's. Codex runs a hook only after it is trusted (`internal/assistant/codexhooks.go` trusts Quiver's own hooks through `codex app-server`): a bundle's hook is written without trust, `quiver tool status` reports "needs trust in Codex", the consent text says so, and Quiver never trusts a bundle's hook for the user.
3. **Instructions.** A marked block in the assistant's rules file (`CLAUDE.md`, `AGENTS.md`, a Cursor rule), with the markers and the pointer logic of TASK54. A file the user symlinked is written through the link only as TASK54 does.
4. **Files.** Files named by the manifest, written under the project root, with their folder added to the repository's exclude file (`git rev-parse --git-path info/exclude`, because in a worktree `.git` is a file). A path that leaves the project, through `..` or through a symlink, is refused. An existing file is never overwritten.
5. **Idempotent.** Writing twice changes nothing and records nothing twice. An entry of the same name that Quiver did not write is skipped with a warning, never replaced.
6. **Windows.** Hook and MCP commands are quoted with `assistant.ShellQuote` (`internal/assistant/claude.go`), which quotes for a POSIX shell only; TASK65.2 adds the Windows form, and until then the test is marked for TASK65.2.

**Done when:** tests, for every assistant `quiver install` supports and both scopes, cover write, write again, reverse, a clashing entry, a JSONC config, a symlinked rules file, a project outside a git repository, a path that leaves the project, a symlink that leaves it, a git worktree, a bundle hook next to Quiver's own hook (neither touches the other), and a bundle hook in Codex (reported as needing trust, never trusted).
