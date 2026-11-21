# TASK65: Windows

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56, TASK63.8

CI runs the tests on Windows (without `-race`, which needs cgo). Every test that calls `assistanttest.Install` or `InstallCodex` skips there (39 calls in 11 test files) because the fake `claude` and `codex` in `internal/assistant/assistanttest/fake.go` are shell scripts. The hook commands of TASK55 add new Windows cases: quoting, paths and the shell each assistant uses. The Release workflow installs a release with `install.ps1` and npm on Windows, but only when a tag is pushed. The local search model of TASK62.6 maps its weight file with a Unix call and reads it where that is not possible (TASK65.2 checks the read on Windows), and the bundle writers of TASK63.4 quote hook and MCP commands with the helper of TASK65.2.

**Done when:** every subtask is done, and no test skips on Windows for lack of a fake.

**Subtasks:**
- [TASK65.1](TASK65.1-fake-assistants-in-go.md): Fake assistants in Go
- [TASK65.2](TASK65.2-hooks-and-paths-on-windows.md): Hooks and paths on Windows
- [TASK65.3](TASK65.3-installed-binary-runs-the-hook.md): Installed binary runs the hook
