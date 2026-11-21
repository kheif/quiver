# TASK68: Assistant behaviour benchmark

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56, TASK62.1, TASK62.8, TASK63.7

Whether assistants load fitting skills and follow always-on skills was measured by hand, once per assistant (docs/spike-b01-results.md, docs/spike-mcp-clients.md, docs/spike-hooks.md). A new model or assistant version can change it without notice. The spike runners in `spike/b01` (`run-real.sh` and `analyze.py` for Claude Code, `cursor-run.sh`, `codex-run.sh`, `antigravity-run.sh`, `gemini-run.sh` and their analyzers) become one benchmark, run before every release and after an assistant update.

Each run calls paid models, so it runs by hand or on a schedule, not on every push. An assistant is supported by the benchmark when it has a headless mode and an adapter (TASK68.2); every other assistant is listed with the reason it is not measured, as TASK62 does for Claude Desktop.

**Done when:** every subtask is done, and a baseline run is committed.

**Subtasks:**
- [TASK68.1](TASK68.1-benchmark-runner.md): Benchmark runner
- [TASK68.2](TASK68.2-assistant-adapters.md): Assistant adapters
- [TASK68.3](TASK68.3-benchmark-cases.md): Benchmark cases
- [TASK68.4](TASK68.4-baselines-and-reports.md): Baselines and reports
