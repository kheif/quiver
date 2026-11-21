# TASK69.1: Fuzzing

**Phase:** Phase 8: Test phase

**Part of:** [TASK69](TASK69-robustness.md): Robustness

**Status:** Todo

**Depends on:** TASK56, TASK63.8

CI fuzzes `skill.Parse` for 30 seconds (`fuzz` job in `.github/workflows/ci.yml`). Add fuzz targets for:
- the JSON config editor (`parseObject` and `change` in `internal/assistant/jsonconfig.go`), and the hooks editing of TASK55.2: other content must come back unchanged;
- reading `registry.json`, the state file and `config.json`;
- the three-way merge in `internal/jsonmerge`;
- the hook's stdin (TASK53);
- the readers of the files that v1.6 and v1.7 add: `state/search.json`, `cache/embeddings.json`, `state/bundles.json` and the bundle manifest (TASK62.5, TASK62.6, TASK63.2, TASK63.3);
- the model loader of TASK62.6 on a tiny generated model (`model.safetensors` and `tokenizer.json`): no crash and no unbounded allocation.

Run each in the CI fuzz job.

**Done when:** every target runs in CI, and each crash found is fixed with a regression test.
