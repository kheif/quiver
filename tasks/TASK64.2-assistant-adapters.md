# TASK64.2: Assistant adapters

**Phase:** Phase 7: Test phase

**Part of:** [TASK64](TASK64-assistant-behaviour-benchmark.md): Assistant behaviour benchmark

**Status:** Todo

**Depends on:** TASK64.1

One adapter per assistant with a headless mode: Claude Code (`claude -p`), Codex (`codex exec --json`), Cursor (`cursor-agent -p`), Antigravity (`agy -p`), and Gemini CLI once its login works (its Google sign-in was refused on 2026-11-01).

Each adapter sets Quiver up only for the run and never changes the user's own config: a throwaway workspace with project config (Claude Code, Cursor), per-run flags (Codex `-c mcp_servers...`), or, where neither exists (Antigravity reads only `~/.gemini/config/mcp_config.json`), an entry added and removed by the run, as `antigravity-run.sh` does. Tool permissions are given per run, never with a flag that skips every permission.

**Done when:** each adapter runs the B01 prompt set and its result matches the last hand measurement.
