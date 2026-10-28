# TASK31 — README and help text

**Phase:** Phase 4 — v1 Release

**Status:** ✅ Done

**Depends on:** TASK30

1. Write a README with: what Quiver is, install, a 2-minute quickstart (`init` → `mcp install` → `new` → ask the assistant), the command reference, the MCP tool reference, the limits of always-on skills (see TASK19) and a security note about remote sources.
2. Give every cobra command a `Short`, a `Long` and an `Example`.
3. Add shell completion with `quiver completion zsh|bash|fish`. Skill, source and project arguments complete with real values.

**Implementation notes:**
- Completion functions are in `internal/cli/complete.go`. They read the home without the lock and never write; an error gives no suggestions. The root's auto-init is skipped for completion requests, so a tab press never creates `~/.quiver`.
- `add`, `where` and `edit` (local only) complete library IDs with their descriptions; `remove` completes what is assigned in the target (`--global`, `--project` or this folder); `update`, `source remove` and `list --source` complete sources; `projects rename|forget` complete IDs and unique names; `--project` and `import` complete folders. Commands without arguments suggest nothing instead of file names.
- `TestHelpText` fails when a command lacks `Short`, `Long` or `Example`.
- The README describes `go install` until the release adds Homebrew (TASK32). PROJECT.md said `quiver update` shows changes before applying; it lists them after pulling, so the README and PROJECT.md now say that.

**Done when:** `quiver help <cmd>` is useful for every command, and completion suggests skill IDs.
