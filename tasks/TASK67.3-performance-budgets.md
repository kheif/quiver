# TASK67.3: Performance budgets

**Phase:** Phase 8: Test phase

**Part of:** [TASK67](TASK67-robustness.md): Robustness

**Status:** Todo

**Depends on:** TASK56

1. A generator in `testdata` (or a test helper) builds a library of 1000 skills, 50 of them always-on in various projects, and 200 projects.
2. Measure on it: the hook's output, MCP server start up to `instructions`, `quiver status` and `quiver list`.
3. Set budgets from the measurements, with room for slower machines. A starting point: the hook under 100 ms, the MCP server start under 300 ms. The hook's budget matters most: it delays every session start.
4. Write the budgets in decisions.md; a test fails when one is exceeded by more than 50%, so CI noise does not fail it.

**Done when:** the budgets are recorded and tested.
