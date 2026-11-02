# TASK67.1: Benchmark runner

**Phase:** Phase 8: Test phase

**Part of:** [TASK67](TASK67-assistant-behaviour-benchmark.md): Assistant behaviour benchmark

**Status:** Todo

**Depends on:** TASK57

1. **Decide first:** Go (one toolchain, reuses Quiver's code) or Python (what the analyzers use now).
2. One tool in `bench/` that prepares the fixture home (as `spike/b01/gui-setup.sh` does), runs a prompt set through an adapter (TASK67.2), and writes one JSON result per run: assistant and version, model, date, Quiver version, and per prompt the Quiver tool calls in order, the skill markers, the always-on marker and any error.
3. It replaces the runners and analyzers in `spike/b01`; `docs/spike-*.md` point to it.

**Done when:** the runner reproduces the Codex numbers in docs/spike-mcp-clients.md within one prompt.
