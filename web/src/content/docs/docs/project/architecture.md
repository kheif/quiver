---
title: Architecture
description: How Quiver is built, the decisions behind it, what is out of scope and where it is heading.
---

Quiver is one Go binary with three roles: a CLI that manages the library, an MCP server, `quiver mcp`, that delivers skills to assistants, and `quiver hook`, which assistants run when a session starts to hand the model its skills. All three read the same files in the Quiver home.

```text
   quiver CLI                 manage: which skills, which projects
        │
   ~/.quiver/                 library + assignments
   local/  sources/  registry.json
        │
   quiver mcp                 serves the right skills per project
        │  instructions · tools · prompts
   Claude Code · Codex · Cursor · Claude Desktop · Antigravity · VS Code · …
        ▲
   quiver hook                session-start text for assistants that run hooks
```

## Design decisions

- **One source of skills.** Skills are read from the Quiver home only. Nothing is copied or linked into projects; a project holds only `.quiver.json`. The MCP server serves them, and `quiver hook` puts the always-on skills and the skill list into a session's context from the same files.
- **The standard format.** Skills are Agent Skills folders with a `SKILL.md`. Quiver adds no format of its own, so existing skills and repositories work as they are.
- **Git for storage.** Your skills are a git repository Quiver commits to; sources are git clones. History, review and sharing come from git.
- **Skills are delivered by a hook.** Claude Code defers MCP tools, so the model does not see their descriptions, and it cuts long `instructions` at about 4096 characters. A session-start hook adds the always-on skills in full and the ID and description of the other skills to the context. An assistant takes only so much text from one hook command, so Quiver splits the text over as many commands as needed. The server's `instructions` stay short.
- **The assistant decides.** Quiver lists skills; the assistant loads the ones a task needs. Only always-on skills are loaded unconditionally.
- **Failures are named.** A hook names what it could not load in a `<quiver-notice>` block, and text that reaches the model is cleaned of hidden characters and Quiver's own tags.
- **Assignments are personal.** They live in the Quiver home, not in projects, so a teammate's clone is not affected by your choices.

## Code layout

| Package | Role |
| --- | --- |
| `cmd/quiver` | The entry point. |
| `internal/cli` | Commands, prompts and help text. |
| `internal/server` | The MCP server: instructions, tools, prompts, live updates, and the session-start text. |
| `internal/library` | Loading skills from `local/` and `sources/`. |
| `internal/registry` | Assignments in `registry.json`. |
| `internal/project` | `.quiver.json` and project discovery. |
| `internal/source` | Cloning, discovery (including plugin marketplaces), updates and pins. |
| `internal/skill` | Parsing and validating `SKILL.md`. |
| `internal/assistant` | Registering the server and the hooks with each assistant. |
| `internal/homesync` | `quiver sync`. |
| `e2e/` | End-to-end tests that build and run the binary. |

The design document is [PROJECT.md](https://github.com/kheif/quiver/blob/main/PROJECT.md); technical decisions are recorded in [tasks/decisions.md](https://github.com/kheif/quiver/blob/main/tasks/decisions.md).

## Out of scope

- Copying or linking skills into projects.
- Running scripts. The assistant runs a skill's scripts itself.
- Defining a new skill format.
- Hosting a central marketplace. Sources are plain git repositories.

## Direction

These are plans, not features. The roadmap in [PROJECT.md](https://github.com/kheif/quiver/blob/main/PROJECT.md#roadmap) has the current state.

- A test phase before the team track: end-to-end journeys, Windows, versioned data formats, a benchmark of assistant behaviour.
- An interactive screen for searching the library and toggling skills, with update notices.
- Search across published skills, so people can find skills others shared.
- Quiver for teams: company and team layers managed in one place, free to run yourself; hosting by us is coming soon. See [Quiver for teams](/teams/).
