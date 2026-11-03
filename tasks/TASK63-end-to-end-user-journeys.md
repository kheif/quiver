# TASK63: End-to-end user journeys

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56, TASK30

The e2e suite (`e2e/`) builds the binary once (`buildAndRun` in `e2e/harness_test.go`) and covers six flows: new, add and remove; `create_skill`; import; source update; move, copy and split; `doctor --fix`. Its helpers run quiver and git in a throwaway home (`newEnv`, `run`, `quiver`, `git`), open an MCP session (`mcp`, `call`) and write files (`write`, `skillMD`).

The journeys below run whole user stories with the real binary, the fake assistants of `internal/assistant/assistanttest` on `PATH`, and local bare git repositories as remotes, without network. Each journey is one test in `e2e/journeys_test.go`; each step checks its output, not only its exit code.

**Done when:** every subtask is done and the journeys run on Linux, macOS and Windows.

**Subtasks:**
- [TASK63.1](TASK63.1-journey-setup-and-delivery.md): Journey: setup and delivery
- [TASK63.2](TASK63.2-journey-library-lifecycle.md): Journey: library lifecycle
- [TASK63.3](TASK63.3-journey-two-machines.md): Journey: two machines
- [TASK63.4](TASK63.4-journey-failures.md): Journey: failures
