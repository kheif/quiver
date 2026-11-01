# TASK50 — `mcp install` for more assistants

**Phase:** Phase 5 — v2

**Status:** ✅ Done

**Depends on:** TASK37

TASK37 added Cursor and Claude Desktop and the shared JSON installer. `mcp install` and `mcp uninstall` take assistant names as arguments.

1. Find out how Codex, Gemini CLI, VS Code and others register stdio MCP servers (config file path and format, or a CLI command). Codex uses TOML.
2. Run the TASK37 spike for each: does it pass the MCP `instructions` to the model?
3. Add each one that does as a name for `mcp install <name>`.
4. Confirm that each assistant passes the project folder (roots or working directory) correctly. Record any assistant that needs `QUIVER_PROJECT`.

**Done when:** Install and uninstall work for each supported assistant, and each one sees the correct project's skills.

**Notes:**
- Spike results in docs/spike-mcp-clients.md (`spike/b01/codex-run.sh`, `spike/b01/antigravity-run.sh`). Codex 0.166 defers MCP tools, so its model gets `instructions` only after it searches for Quiver's tools: 0 of 6 fitting prompts loaded a skill unprompted, every one did when the prompt named Quiver. Antigravity CLI 1.4.12 writes `instructions` to a file the model rarely reads: 1 of 6. Both start the server in the working directory, so project skills work. Gemini CLI 0.66 could not be measured: its Google sign-in is refused. VS Code is not measured yet.
- Codex is set up with `codex mcp add`, which keeps `config.toml`'s comments; it replaces the whole entry, so `internal/assistant/codex.go` passes an existing `env` again. Antigravity is a JSON assistant (`~/.gemini/config/mcp_config.json`), installed when `~/.gemini/antigravity-cli` exists.
- Decided 2026-11-01: `mcp install` without names sets up every supported assistant that is installed, Claude Desktop included (`JSONAssistant.NoInstructions` became `OnRequest`). Codex, Claude Desktop and Antigravity get a note that they use Quiver's skills only when asked. `doctor` checks every installed assistant.
- The read tools (`list_skills`, `get_skill`, `get_skill_file`, `search_skills`) are marked `readOnlyHint`, so Codex runs them without asking.
- `internal/cli` tests drop the folders holding `codex` from `PATH` and use `assistanttest.InstallCodex`; `TestRealCodex` checks the fake against the real codex in a throwaway `CODEX_HOME` when it is installed.
