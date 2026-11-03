# TASK64: Windows

**Phase:** Phase 8: Test phase

**Status:** Todo

**Depends on:** TASK56

CI runs the tests on Windows (without `-race`, which needs cgo). Six tests skip there because the fake `claude` and `codex` in `internal/assistant/assistanttest/fake.go` are shell scripts. The hook commands of TASK55 add new Windows cases: quoting, paths and the shell each assistant uses. The Release workflow installs a release with `install.ps1` and npm on Windows, but only when a tag is pushed.

**Done when:** every subtask is done, and no test skips on Windows for lack of a fake.

**Subtasks:**
- [TASK64.1](TASK64.1-fake-assistants-in-go.md): Fake assistants in Go
- [TASK64.2](TASK64.2-hooks-and-paths-on-windows.md): Hooks and paths on Windows
- [TASK64.3](TASK64.3-installed-binary-runs-the-hook.md): Installed binary runs the hook
