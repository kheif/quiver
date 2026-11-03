# TASK53: `quiver hook session-start`

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** Todo

**Depends on:** TASK52, TASK19

A command that a session-start hook runs: it prints the always-on skills of the folder the session starts in, in the format the assistant expects. A second command does the same when a sub-agent starts, since the session-start text does not reach sub-agents (TASK52.1). Installing the hooks is TASK55.

1. **Command.** `quiver hook session-start <assistant>` in a new `internal/cli/hook.go`. `<assistant>` is a name from `quiver install` (`claude-code`, `codex`, `vscode`, `cursor`, `antigravity`) and selects the input and output format found in TASK52: for `cursor` `{"additional_context": ...}`, for `antigravity` `{"injectSteps": [{"systemMessage": {"systemMessage": ...}}]}` with the input's `workspacePaths[0]` as the folder (TASK52.4). For `vscode` the output carries the text twice, as `additionalContext` and as `hookSpecificOutput.additionalContext`, since VS Code's two harnesses each read one of them (TASK52.3). It is listed in the help like `quiver mcp`, with a Long text that says assistants run it.
2. **Project.** Read the hook's JSON from stdin and take its `cwd`. Resolve the project as the MCP server does (`server.Resolve` in `internal/server/resolve.go`): `QUIVER_PROJECT` first, then that folder. A folder outside any project gets only the global skills.
3. **Skills.** The IDs come from `Registry.AlwaysOnServed` (`internal/registry/assign.go`): global ones first, then the project's, each in the order it was marked, each ID once. IDs no longer in the library are left out, as in the catalog. The text of each skill is what `get_skill` returns: move the code that builds it (`skillText` in `internal/server/skills.go`) where both can use it.
4. **Block.** Print one marked block, so the MCP server's pointer (TASK54) can refer to it:
   ```
   <quiver-always-on>
   Always-on skills from Quiver. Follow each one for the whole conversation.

   ## caveman/caveman
   <the skill's SKILL.md body>

   ## reply-style
   <...>

   Extra files of these skills are loaded with the get_skill_file tool of the quiver MCP server.
   </quiver-always-on>
   ```
5. **Size.** Keep the whole block under 10,000 characters, Claude Code's limit (TASK52.1). The Codex hook turns Codex's own limit off (TASK55.3), so the same limit applies there; it also stays below Antigravity's cut at about 48,900 bytes (TASK52.4). If the skills do not fit, print whole skills in order while they fit, and list the rest by ID with a line to load them with `get_skill` before the first reply.
6. **Sub-agents.** `quiver hook subagent-start <assistant>` prints the same block for the assistant's sub-agent event (`SubagentStart` in Claude Code, Codex and VS Code; Antigravity runs `SessionStart` for each sub-agent, and Cursor's `subagentStart` cannot add text). Its input has the same `cwd`. Claude Code takes a sub-agent's context only from `additionalContext`, and `hookEventName` must name the event: with a wrong name the hook fails (TASK52.1).
7. **Never break a session.** No network. On any error, print what is known, write the error to `~/.quiver/logs/hook.log`, and exit 0. Log one line per run: time, assistant, project and skill IDs. The run must stay within the budget of TASK67.3.

**Out of scope:** installing the hook (TASK55), per-prompt hooks.

**Done when:** tests in `internal/cli` cover the order, an ID in both scopes, a removed skill, the block, the size limit, a folder outside any project, `QUIVER_PROJECT`, a broken `registry.json` (exit 0 and a log line), each output format and `subagent-start`.
