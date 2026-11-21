# TASK63.1: Bundle spike

**Phase:** Phase 7c: Tool bundles (v1.7)

**Part of:** [TASK63](TASK63-tool-bundles.md): Tool bundles

**Status:** Todo

**Depends on:** TASK62.10

Measure before building, as docs/spike-b01-results.md did for the MCP design. The spike starts after the release of v1.6.0 (TASK62.10) and needs none of its code: it carries throwaway prototypes under `spike/bundles/` (not product code, with its own `go.mod` like `spike/b01`, so CI and the linter do not compile it). Every run uses a throwaway home and a scratch project (a changed `HOME`, `XDG_*`, `UV_TOOL_DIR` and the like), never the real home, as in the earlier spikes.

1. **What the tools write.** For Graphify (one pinned version) and claude-mem (one pinned version), run each documented install path for Claude Code and for one assistant without hooks. Snapshot every file under the throwaway home and the scratch project before and after: path, JSON key or marked block, content. Record per component what was written, whether a second run changes anything, whether the tool's own uninstall restores the snapshot, and what survives it (data, caches, a running process).
2. **Declared or command.** For each component, whether Quiver's own writers (`internal/assistant`, the hook installers) produce the same entry from a declaration, and what only the tool's command can do (installing the program, a post-install step); and whether each assistant accepts an MCP entry name that carries the bundle's ID (a `/` or another character in a server name), which fixes the naming rule of TASK63.4.
3. **Plugin format.** What a Claude Code plugin directory holds (manifest, MCP servers, hooks) according to its current documentation, dated; whether a manifest can be imported from it unchanged, and what it lacks for the other assistants.
4. **Status.** What each tool offers that a status command could show (a version flag, a log, a port), with no metrics collected by Quiver.
5. **What goes wrong.** A tool that updates itself; two tools hooking the same event; an assistant whose settings are under an administrator (user hooks refused); a tool whose own installer writes a skill that Quiver also serves; a runtime that is missing.

The result goes to docs/spike-bundles-results.md with the commands and the raw snapshots' differences. It decides: the list of component types; where a manifest lives (in the tool's own repository, in a directory Quiver keeps, or both); which parts of each first bundle are declared and which are commands; whether the plugin format is imported; and the policy for a missing runtime.

**Done when:** the file exists and answers each numbered question with measured output, and TASK63.2 to TASK63.7 are corrected to match it before any of them starts. If a tool cannot be removed cleanly from a declaration and its own uninstall, the phase is planned again before anything ships.
