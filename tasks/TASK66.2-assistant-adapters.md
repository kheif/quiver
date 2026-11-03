# TASK66.2: Assistant adapters

**Phase:** Phase 8: Test phase

**Part of:** [TASK66](TASK66-assistant-behaviour-benchmark.md): Assistant behaviour benchmark

**Status:** Todo

**Depends on:** TASK66.1

One adapter per assistant with a headless mode: Claude Code (`claude -p`), Codex (`codex exec --json`), Cursor (`cursor-agent -p`), Antigravity (`agy -p`), and Gemini CLI once its login works (its Google sign-in was refused on 2026-11-01).

Each adapter sets Quiver up only for the run and never changes the user's own config: a throwaway workspace with project config (Claude Code, Cursor, Antigravity's `.agents/` folder), per-run flags (Codex `-c mcp_servers...`), or a throwaway home where an assistant has no project config. Tool permissions are given per run, never with a flag that skips every permission.

**Done when:** each adapter runs the B01 prompt set and its result matches the last hand measurement.
