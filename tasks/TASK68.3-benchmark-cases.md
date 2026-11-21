# TASK68.3: Benchmark cases

**Phase:** Phase 8: Test phase

**Part of:** [TASK68](TASK68-assistant-behaviour-benchmark.md): Assistant behaviour benchmark

**Status:** Todo

**Depends on:** TASK68.2, TASK62.8, TASK63.7

Prompt sets, each with the expected result:
- fitting and non-fitting prompts (B01);
- always-on through the hook, and through the MCP fallback with the hook removed;
- a project skill inside and outside its project;
- a catalog longer than the `instructions` limit (`FILLER` in `run-real.sh`);
- after compaction, where the assistant supports it;
- a sub-agent task;
- the cases above again with the bundle of a tool whose hooks or instructions change behaviour installed (TASK63.7), to show that always-on skills and the search cue still work next to it;
- a task whose topic only a Searchable skill covers, with no skill listed (the assistant searches, loads the fitting skill and applies it, TASK62.8), and a plain question about the project (the assistant does not search).

**Done when:** every case runs on every adapter that supports it.
