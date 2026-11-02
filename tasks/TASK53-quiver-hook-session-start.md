# TASK53: `quiver hook session-start`

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** Todo

**Depends on:** TASK52, TASK19

A command that a session-start hook runs: it prints the always-on skills of the folder the session starts in, in the format the assistant expects. Installing the hook is TASK55.

1. **Command.** `quiver hook session-start <assistant>` in a new `internal/cli/hook.go`. `<assistant>` is a name from `quiver install` (such as `claude-code`, `codex`) and selects the input and output format found in TASK52. It is listed in the help like `quiver mcp`, with a Long text that says assistants run it.
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
5. **Size.** Keep the whole block under the assistant's limit (TASK52). If the skills do not fit, print whole skills in order while they fit, and list the rest by ID with a line to load them with `get_skill` before the first reply.
6. **Never break a session.** No network. On any error, print what is known, write the error to `~/.quiver/logs/hook.log`, and exit 0. Log one line per run: time, assistant, project and skill IDs. The run must stay within the budget of TASK68.3.

**Out of scope:** installing the hook (TASK55), per-prompt hooks.

**Done when:** tests in `internal/cli` cover the order, an ID in both scopes, a removed skill, the block, the size limit, a folder outside any project, `QUIVER_PROJECT`, a broken `registry.json` (exit 0 and a log line) and each output format.
