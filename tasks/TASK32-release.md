# TASK32 — Release

**Phase:** Phase 4 — v1 Release

**Depends on:** TASK31

1. Set up GoReleaser for darwin/linux/windows on amd64 and arm64.
2. Publish GitHub Releases with checksums.
3. Add a Homebrew tap: `brew install <owner>/tap/quiver`.
4. After a Homebrew upgrade, `quiver mcp install` must still point to the right binary. Use the stable Homebrew path, not the versioned Cellar path.
5. Tag `v1.0.0`.

**Done when:** `brew install` on a clean machine works, and the quickstart runs.
