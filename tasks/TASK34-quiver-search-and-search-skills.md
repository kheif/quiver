# TASK34 — `quiver search` and `search_skills`

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK12, TASK18

1. Do a case-insensitive match on ID, name and description across the whole library.
2. Rank results: exact name match, then name prefix, then description match.
3. CLI: `quiver search <text>`, with `--json`.
4. MCP: `search_skills` (`query`) returns the matches and whether each one is assigned here. The description tells the assistant it may suggest an unassigned skill to the user but should not assume it applies.

**Done when:** Ranking tests pass, and the MCP tool marks assigned and unassigned skills correctly.
