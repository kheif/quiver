# TASK17 — Project resolution in the server

**Phase:** Phase 2 — MCP Server (first usable version)

**Depends on:** TASK16, TASK06, TASK07

The server must know which project it is serving. Resolve it once, at startup, because `instructions` (TASK18) are built before the first request:

1. `QUIVER_PROJECT` environment variable, if set.
2. The process working directory. Claude Code starts stdio servers in the project folder (confirmed in the B01 spike).

Then:

- Run `project.Find` on the result. It walks up to the nearest `.quiver.json`, so subfolders and monorepos work. Remember the project ID for the session.
- If there is no `.quiver.json`, serve only global skills. Do not create the file until the user creates or assigns a skill there. Remember the folder, so `create_skill` can register it.
- Do not use MCP roots. Under protocol version 2026-07-28 the server cannot send `roots/list` while serving a request (SEP-2322), and Claude Code never triggered the `initialized` handler in the spike. See `docs/spike-b01-results.md`.

**Done when:**
- Harness tests cover both sources and a start inside a subfolder.
- The resolved key is logged at startup.
