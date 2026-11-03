# TASK64.1: Fake assistants in Go

**Phase:** Phase 8: Test phase

**Part of:** [TASK64](TASK64-windows.md): Windows

**Status:** Todo

**Depends on:** TASK56

1. Write the fakes as one Go program (for example `internal/assistant/assistanttest/fakecmd`) that acts as `claude` or `codex` depending on the name it is run as, and keeps its state and call log in files next to it, as the scripts do now.
2. `Install` and `InstallCodex` build it once per test binary (in a temp folder, named `claude.exe` and `codex.exe` on Windows) and put it first on `PATH`.
3. Keep the `Fake` API (`SetUser`, `SetLocal`, `SetCodex`, `User`, `Calls`), so the tests do not change.
4. Remove the Windows skips that were only there for the fakes.

**Done when:** the six tests run and pass on the Windows runner.
