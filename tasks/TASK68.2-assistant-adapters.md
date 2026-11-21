# TASK68.2: Assistant adapters

**Phase:** Phase 8: Test phase

**Part of:** [TASK68](TASK68-assistant-behaviour-benchmark.md): Assistant behaviour benchmark

**Status:** Todo

**Depends on:** TASK68.1, TASK62.1

One adapter per assistant with a headless mode: Claude Code (`claude -p`), Codex (`codex exec --json`), Cursor (`cursor-agent -p`), Antigravity (`agy -p`), VS Code Copilot if it can be driven by a script (TASK62.1 checks that), and Gemini CLI once its login works (its Google sign-in was refused on 2026-11-01). An assistant whose adapter cannot be built is listed as not measured, with the reason. The adapters also run the cells of TASK62.1 for every assistant other than Claude Code, with the same thresholds, and the result is added to docs/spike-discovery-results.md.

Each adapter sets Quiver up only for the run and never changes the user's own config: a throwaway workspace with project config (Claude Code, Cursor, Antigravity's `.agents/` folder), per-run flags (Codex `-c mcp_servers...`), or a throwaway home where an assistant has no project config. Tool permissions are given per run, never with a flag that skips every permission.

**Done when:** each adapter runs the B01 prompt set and its result matches the last hand measurement.
