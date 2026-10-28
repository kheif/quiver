# TASK43 — Prefix-free commands in Claude Code

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK25, TASK14, TASK20

MCP prompts (TASK25) appear in Claude Code as `/quiver:<id>`, and the server name cannot be dropped. Claude Code's own skills have no prefix: a folder `.claude/skills/<name>/SKILL.md` becomes `/<name>`. Quiver writes a small Claude Code skill for every served skill, so the user types `/review ship 1.2`.

Facts from the Claude Code docs (checked 2026-10-28):
- Personal skills live in `~/.claude/skills/<name>/`, project skills in `<project>/.claude/skills/<name>/`. On a name clash, personal wins over project.
- `$ARGUMENTS` holds all the text after the command; quotes keep words together. No 32-word limit as with prompts.
- `disable-model-invocation: true` keeps Claude from loading the skill by itself, so it does not repeat the MCP catalog.
- Claude Code watches these folders and picks up changes in the running session.

1. For each served skill, write one Claude Code skill:
   - global skills: `~/.claude/skills/<command>/SKILL.md`
   - project skills: `<project>/.claude/skills/<command>/SKILL.md` in every recorded project path, and add the folder to the repository's `.git/info/exclude` so it is never committed
2. The file has frontmatter `name`, `description`, `disable-model-invocation: true` and a marker `quiver-skill: <id>`. Its body tells the assistant to load `<id>` with the quiver server's `get_skill` tool, follow it, and use `$ARGUMENTS` as the task.
3. The command is the skill's `name`: `acme/review` gives `/review`. If two served skills share a name, both use their ID with `/` replaced by `-` (`/acme-review`).
4. Never change or delete a folder without the marker. If the user's own skill has the name, skip it and warn in `quiver status` and `quiver doctor`.
5. Sync after every change to what is served: `add`, `remove`, `new`, `create_skill`, `update`, `source remove`. Delete generated folders whose skill is no longer served.
6. Opt in: `quiver mcp install` asks once (or `--commands`), saved in the config. `quiver mcp uninstall` removes every generated folder.

**Decide first:** the body could instead hold a copy of the skill. That needs no tool call and works when the server is down, but extra files (`references/…`) are then missing, so the pointer is the default.

**Done when:**
- Tests cover naming, name clashes, the marker rule, removal of stale folders and `.git/info/exclude`.
- In Claude Code, `/review ship version 1.2` loads the skill and gets the whole task.
