# TASK64.2: Hooks and paths on Windows

**Phase:** Phase 8: Test phase

**Part of:** [TASK64](TASK64-windows.md): Windows

**Status:** Todo

**Depends on:** TASK64.1

1. Quote hook commands for the shell each assistant runs them in on Windows (TASK52). `assistant.ShellQuote` quotes for a POSIX shell only; add the Windows form.
2. Paths with spaces and non-ASCII user names, in the Quiver home, the binary's path and the project path.
3. The `%APPDATA%` locations of every assistant config.
4. CRLF line endings in skill files: parsing still works.

**Done when:** tests cover each point and pass on the Windows runner.
