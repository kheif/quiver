# TASK36 — `quiver show`, `quiver search`, every skill in the catalog

**Phase:** Phase 4c — v1.2

**Status:** ✅ Done

**Depends on:** TASK33

1. **`quiver show <id>`** prints a skill: ID, where it comes from (local, or source and commit), where it is assigned, always-on, its files, then the `SKILL.md` text. `--json`. Completion of IDs.
2. **`quiver search <text>`** searches the whole library, also unassigned skills: case-insensitive match on ID, name and description. Rank: exact name, name prefix, name contains, description contains. Show whether each match is assigned here. `--json`. Put the search in `internal/library`, so `search_skills` (Phase 5) and the TUI use the same function.
3. **Every skill in the catalog, with its description.** Claude Code cuts the server's `instructions` at about 4096 characters (`maxInstructions` is 3800), and hook output at about 10,000. Today `catalog` in `internal/server/catalog.go` then drops descriptions from the end (about 20 skills keep one) and leaves skills out after about 150. A tool result has no such limit, and the always-on pointer showed that the assistant calls a tool before its first reply when told to (10 of 10, and after `/compact`). Decided 2026-10-29:
   - When the catalog fits, it stays in `instructions` as today.
   - When it does not fit, `instructions` hold no list, only a pointer like the always-on one: before your first reply, call `list_skills` of this server; it lists the skills of this folder, and when one fits a task, load it with `get_skill`. The always-on pointer stays first.
   - `list_skills` returns every served skill with its full description (project skills first, then global), and marks always-on ones.
   - Remove the shortening and dropping in `catalog`; keep the `maxDescription` cut only as a guard against huge descriptions.
   - Measure it like the always-on spike (docs/spike-b01-results.md): with a catalog over the budget, a fitting prompt in a new session calls `list_skills` and then loads the skill, in at least 9 of 10 runs, also after `/compact`. Record the result there.
   - Too many skills for one tool result (hundreds) is for later: group them by area, and list an area on request.

**Done when:** tests cover show for a local and a remote skill, search ranking, the catalog in `instructions` when it fits and the `list_skills` pointer when it does not, and `list_skills` with 200 skills.

**Notes:**
- `catalog` lists every served skill or none: a list over the budget becomes `listPointer`. `maxDescription` is 1024, the Agent Skills limit.
- `list_skills` adds `always_on` and sorts project skills first; `loadServed` keeps each part sorted by ID.
- `library.Search` ranks an exact ID like an exact name, and an ID that contains the text like a name that contains it.
- `quiver search` shows `global` or `project` under HERE; `quiver show` gives the HEAD of `local/` or of the source as the commit.
- Measured with 44 skills: 10 of 10 fitting prompts called `list_skills` and loaded the skill, 4 of 4 after `/compact` (docs/spike-b01-results.md, `make spike-catalog`).
