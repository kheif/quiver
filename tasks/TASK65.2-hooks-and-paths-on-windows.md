# TASK65.2: Hooks and paths on Windows

**Phase:** Phase 8: Test phase

**Part of:** [TASK65](TASK65-windows.md): Windows

**Status:** Todo

**Depends on:** TASK65.1, TASK63.8

1. Quote hook commands for the shell each assistant runs them in on Windows (TASK52). `assistant.ShellQuote` quotes for a POSIX shell only; add the Windows form.
2. Paths with spaces and non-ASCII user names, in the Quiver home, the binary's path and the project path.
3. The `%APPDATA%` locations of every assistant config.
4. CRLF line endings in skill files: parsing still works.
5. The weight file of the local search model (TASK62.6): the loader reads or maps it on Windows (the spike read it, 191 MB), and the tests with the tiny model pass there.
6. The hook and MCP commands that the bundle writers of TASK63.4 produce use the same quoting, and the test that TASK63.4 marked for this task is switched on.

**Done when:** tests cover each point and pass on the Windows runner.
