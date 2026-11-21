# TASK66.2: Journey: library lifecycle

**Phase:** Phase 8: Test phase

**Part of:** [TASK66](TASK66-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK56, TASK30, TASK63.8, TASK65.1, TASK65.2

1. `source add`, then a new commit in the remote that changes an always-on skill and a script.
2. `quiver update`: the preview marks the always-on skill and flags the script; answering no changes nothing, answering yes applies.
3. `pin` the skill, change the remote again, `update` leaves it; `unpin`, `update` takes the change.
4. `fork` the skill, `edit` the copy; the copy replaces the original wherever it was assigned or always-on.
5. `source remove` is refused while a skill is assigned, then works with `--force`.
6. `add --searchable` on a skill: `quiver search` and `search_skills` find it, and `remove` takes it out of every level of the scope (TASK62.3).
7. A bundle: `quiver tool add` of the fake tool of TASK63.7, `tool status`, `tool update` after the manifest of the fake tool changed, and `tool remove`; afterwards the config files are as before and the ledger is empty.

After each step, `quiver status`, the hook's output and the MCP server's `get_skill` agree.

**Done when:** the journey passes on all three systems.
