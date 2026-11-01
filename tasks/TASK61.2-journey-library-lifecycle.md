# TASK61.2: Journey: library lifecycle

**Phase:** Phase 7: Test phase

**Part of:** [TASK61](TASK61-end-to-end-user-journeys.md): End-to-end user journeys

**Status:** Todo

**Depends on:** TASK59, TASK30

1. `source add`, then a new commit in the remote that changes an always-on skill and a script.
2. `quiver update`: the preview marks the always-on skill and flags the script; answering no changes nothing, answering yes applies.
3. `pin` the skill, change the remote again, `update` leaves it; `unpin`, `update` takes the change.
4. `fork` the skill, `edit` the copy; the copy replaces the original wherever it was assigned or always-on.
5. `source remove` is refused while a skill is assigned, then works with `--force`.

After each step, `quiver status`, the hook's output and the MCP server's `get_skill` agree.

**Done when:** the journey passes on all three systems.
