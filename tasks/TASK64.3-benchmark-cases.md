# TASK64.3: Benchmark cases

**Phase:** Phase 7: Test phase

**Part of:** [TASK64](TASK64-assistant-behaviour-benchmark.md): Assistant behaviour benchmark

**Status:** Todo

**Depends on:** TASK64.2

Prompt sets, each with the expected result:
- fitting and non-fitting prompts (B01);
- always-on through the hook, and through the MCP fallback with the hook removed;
- a project skill inside and outside its project;
- a catalog longer than the `instructions` limit (`FILLER` in `run-real.sh`);
- after compaction, where the assistant supports it;
- a sub-agent task.

**Done when:** every case runs on every adapter that supports it.
