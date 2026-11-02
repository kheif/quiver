# TASK67.4: Baselines and reports

**Phase:** Phase 8: Test phase

**Part of:** [TASK67](TASK67-assistant-behaviour-benchmark.md): Assistant behaviour benchmark

**Status:** Todo

**Depends on:** TASK67.3

1. A report prints, per assistant and case, the rates (fitting skill loaded, always-on followed, wrong skill loaded) and compares them with the last committed baseline. It fails when a rate drops by more than a threshold set per case.
2. Results are kept as `docs/bench/<date>-<assistant>.json`, and a short table in `docs/bench/README.md`.
3. docs/releasing.md requires a run before a release.

**Done when:** a baseline for every supported assistant is committed.
