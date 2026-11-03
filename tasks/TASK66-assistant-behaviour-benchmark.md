# TASK66: Assistant behaviour benchmark

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56

Whether assistants load fitting skills and follow always-on skills was measured by hand, once per assistant (docs/spike-b01-results.md, docs/spike-mcp-clients.md, docs/spike-hooks.md). A new model or assistant version can change it without notice. The spike runners in `spike/b01` (`run-real.sh` and `analyze.py` for Claude Code, `cursor-run.sh`, `codex-run.sh`, `antigravity-run.sh`, `gemini-run.sh` and their analyzers) become one benchmark, run before every release and after an assistant update.

Each run calls paid models, so it runs by hand or on a schedule, not on every push.

**Done when:** every subtask is done, and a baseline run is committed.

**Subtasks:**
- [TASK66.1](TASK66.1-benchmark-runner.md): Benchmark runner
- [TASK66.2](TASK66.2-assistant-adapters.md): Assistant adapters
- [TASK66.3](TASK66.3-benchmark-cases.md): Benchmark cases
- [TASK66.4](TASK66.4-baselines-and-reports.md): Baselines and reports
