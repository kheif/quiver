# TASK43 — Stable skill IDs

**Phase:** Phase 5 — v2

**Status:** ✅ Done

**Depends on:** TASK42

Backlog B05. A remote skill's ID is `<source>/<name>`, and `<source>/<plugin>/<name>` only when names collide. If an update adds a skill with the same name, the existing ID changes and its assignments point at nothing.

1. Store each remote skill's ID when it is first discovered (in the source's entry in `config.json` or in the state), and keep it across updates. Only new skills get the qualified form on a clash.
2. A skill that disappears from its source keeps its ID; `quiver update` and `doctor` report it as removed upstream, as today.
3. Update data-model.md.

**Done when:** a test updates a source so that a second `pdf` appears, and the existing `src/pdf` assignment still works.
