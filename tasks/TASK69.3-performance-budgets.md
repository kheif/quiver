# TASK69.3: Performance budgets

**Phase:** Phase 8: Test phase

**Part of:** [TASK69](TASK69-robustness.md): Robustness

**Status:** Todo

**Depends on:** TASK56, TASK63.8

1. A generator in `testdata` (or a test helper) builds a library of 1000 skills, 50 of them always-on in various projects and 500 of them Searchable, and 200 projects.
2. Measure on it: the hook's output with the cue (TASK62.8 leaves the budget of the hook to this task), MCP server start up to `instructions`, `quiver status` and `quiver list`, a `search_skills` call with the `words` engine and with the `local` engine (the query vector from the mapped file, TASK62.6), and the build of the index for 1000 skills.
3. Set budgets from the measurements, with room for slower machines. A starting point: the hook under 100 ms, the MCP server start under 300 ms. The hook's budget matters most: it delays every session start.
4. Write the budgets in decisions.md; a test fails when one is exceeded by more than 50%, so CI noise does not fail it.

**Done when:** the budgets are recorded and tested.
