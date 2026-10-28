# TASK32 — Release

**Phase:** Phase 4 — v1 Release

**Status:** Todo

**Depends on:** TASK31

1. Set up GoReleaser for darwin/linux/windows on amd64 and arm64.
2. Publish GitHub Releases with checksums.
3. Add a Homebrew tap: `brew install <owner>/tap/quiver`.
4. After a Homebrew upgrade, `quiver mcp install` must still point to the right binary. Use the stable Homebrew path, not the versioned Cellar path.
5. Decide backlog B07 (approval for global `create_skill`) before tagging. A global skill's description reaches the instructions in every folder. The smallest fix: leave unapproved global skills out of the catalog.
6. Tag `v1.0.0`.

**Done when:** `brew install` on a clean machine works, and the quickstart runs.
