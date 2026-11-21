# TASK63.7: First bundles

**Phase:** Phase 7c: Tool bundles (v1.7)

**Part of:** [TASK63](TASK63-tool-bundles.md): Tool bundles

**Status:** Todo

**Depends on:** TASK63.5

Two real tools prove that the manifest is general: Graphify, which has a program, an MCP server, hooks and project files, and claude-mem, which is a plugin with hooks, an MCP server, a background worker and data.

1. A manifest for each, written from the decisions of TASK63.1, pinned to the version that was measured there. Each names its `requires`, its `kind` (claude-mem: `memory`), whether it installs its own skill, and the data it leaves behind.
2. **Fake tools.** A test double for each, a small program that writes the same shape of files as the measured real one, so the end-to-end tests (TASK30) add, change, remove and update a bundle on every assistant with no network and no real install.
3. **Real runs.** Each bundle is added and removed with the real tool on a throwaway home on Claude Code, by hand; the commands, the snapshot before and after, and the result are recorded in docs/spike-bundles-results.md.
4. **Documentation.** A README section and a website guide, "Managing tools", with the consent text, the ledger, what removal leaves behind and how to write a manifest (the page carries the "Next release" aside and badge of `web/README.md` until the release).

**Done when:** the end-to-end tests pass on every assistant, both real runs are recorded with an empty ledger and unchanged config files after removal, and a third manifest written only from the documentation works.
