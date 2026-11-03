# TASK67.1: Fuzzing

**Phase:** Phase 8: Test phase

**Part of:** [TASK67](TASK67-robustness.md): Robustness

**Status:** Todo

**Depends on:** TASK56

CI fuzzes `skill.Parse` for 30 seconds (`fuzz` job in `.github/workflows/ci.yml`). Add fuzz targets for:
- the JSON config editor (`parseObject` and `change` in `internal/assistant/jsonconfig.go`), and the hooks editing of TASK55.2: other content must come back unchanged;
- reading `registry.json`, the state file and `config.json`;
- the three-way merge in `internal/jsonmerge`;
- the hook's stdin (TASK53).

Run each in the CI fuzz job.

**Done when:** every target runs in CI, and each crash found is fixed with a regression test.
