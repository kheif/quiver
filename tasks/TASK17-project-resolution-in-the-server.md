# TASK17 — Project resolution in the server

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** ✅ Done

**Depends on:** TASK16, TASK06, TASK07

The server must know which project it is serving. Resolve it once, at startup, because `instructions` (TASK18) are built before the first request:

1. `QUIVER_PROJECT` environment variable, if set.
2. The process working directory. Claude Code starts stdio servers in the project folder (confirmed in the B01 spike).

Then:

- Run `project.Find` on the result. It walks up to the nearest `.quiver.json`, so subfolders and monorepos work. Remember the project ID for the session.
- If there is no `.quiver.json`, serve only global skills. Do not create the file until the user creates or assigns a skill there. Remember the folder, so `create_skill` can register it.
- Do not use MCP roots. Under protocol version 2026-07-28 the server cannot send `roots/list` while serving a request (SEP-2322), and Claude Code never triggered the `initialized` handler in the spike. See `docs/spike-b01-results.md`.

**Notes:**
- `server.Resolve` returns a `Where`: `Dir` (the folder to register later), `From`, and `Project` (nil without `.quiver.json`).
- No tool reads `Where` yet, so its tests call `Resolve` directly. TASK18 should cover the served set per project through the harness.
- A missing `QUIVER_PROJECT` folder or a broken `.quiver.json` does not stop the server. It serves the global skills and only logs a warning, so the assistant never hears about it. TASK18 could mention it in `list_skills`.
- At startup the server also records the folder of a known project (`recordPath`), so a moved project gets its new path. It writes `registry.json` only when the paths change, and does not add projects the registry does not know.
- `Where` is fixed for the session. If `quiver add` registers the folder during a session, the server keeps serving only the global skills until the next session, unless TASK26 resolves again.
- Checked by hand: Claude Code starts the server in the project folder (`dir=` in `logs/mcp.log`).

**Done when:**
- Harness tests cover both sources and a start inside a subfolder.
- The resolved key is logged at startup.
