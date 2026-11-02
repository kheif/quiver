# TASK67: Assistant behaviour benchmark

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK57

Whether assistants load fitting skills and follow always-on skills was measured by hand, once per assistant (docs/spike-b01-results.md, docs/spike-mcp-clients.md, docs/spike-hooks.md). A new model or assistant version can change it without notice. The spike runners in `spike/b01` (`run-real.sh` and `analyze.py` for Claude Code, `cursor-run.sh`, `codex-run.sh`, `antigravity-run.sh`, `gemini-run.sh` and their analyzers) become one benchmark, run before every release and after an assistant update.

Each run calls paid models, so it runs by hand or on a schedule, not on every push.

**Done when:** every subtask is done, and a baseline run is committed.

**Subtasks:**
- [TASK67.1](TASK67.1-benchmark-runner.md): Benchmark runner
- [TASK67.2](TASK67.2-assistant-adapters.md): Assistant adapters
- [TASK67.3](TASK67.3-benchmark-cases.md): Benchmark cases
- [TASK67.4](TASK67.4-baselines-and-reports.md): Baselines and reports
