# TASK68.2: Parallel processes

**Phase:** Phase 8: Test phase

**Part of:** [TASK68](TASK68-robustness.md): Robustness

**Status:** Todo

**Depends on:** TASK57

The MCP server, CLI commands and the hook run at the same time: two sessions start while `quiver update` writes the registry. Writers take the lock (`fsutil.WithLock`, a flock on `~/.quiver/.lock`) and replace files atomically (`fsutil.WriteFileAtomic`); readers such as the hook take no lock.

1. Tests start real quiver processes in parallel: a writer loop (`add`, `remove`, `update`) and reader loops (`hook session-start`, `status`, an MCP session calling `list_skills`).
2. Check that no reader ever fails to parse a file or sees a mix of two versions, and that no write is lost.

**Done when:** the tests pass with `-race` on Linux and macOS and without it on Windows.
