---
title: Architecture
description: How Quiver is built, the decisions behind it, what is out of scope and where it is heading.
---

Quiver is one Go binary with two roles: a CLI that manages the library, and an MCP server, `quiver mcp`, that delivers skills to assistants. Both read the same files in the Quiver home.

```text
   quiver CLI                 manage: which skills, which projects
        │
   ~/.quiver/                 library + assignments
   local/  sources/  registry.json
        │
   quiver mcp                 serves the right skills per project
        │  instructions · tools · prompts
   Claude Code · Codex · Cursor · Claude Desktop · Antigravity · …
```

## Design decisions

- **One delivery path.** Skills reach assistants only through MCP. Nothing is copied or linked into projects; a project holds only `.quiver.json`.
- **The standard format.** Skills are Agent Skills folders with a `SKILL.md`. Quiver adds no format of its own, so existing skills and repositories work as they are.
- **Git for storage.** Your skills are a git repository Quiver commits to; sources are git clones. History, review and sharing come from git.
- **Instructions, not tool descriptions, carry the catalog.** Claude Code defers MCP tools, so the model does not see their descriptions. The skill list goes into the server's `instructions`, which assistants add to the system prompt.
- **The assistant decides.** Quiver lists skills; the assistant loads the ones a task needs. Only always-on skills are loaded unconditionally.
- **Assignments are personal.** They live in the Quiver home, not in projects, so a teammate's clone is not affected by your choices.

## Code layout

| Package | Role |
| --- | --- |
| `cmd/quiver` | The entry point. |
| `internal/cli` | Commands, prompts and help text. |
| `internal/server` | The MCP server: instructions, tools, prompts, live updates. |
| `internal/library` | Loading skills from `local/` and `sources/`. |
| `internal/registry` | Assignments in `registry.json`. |
| `internal/project` | `.quiver.json` and project discovery. |
| `internal/source` | Cloning, discovery (including plugin marketplaces), updates and pins. |
| `internal/skill` | Parsing and validating `SKILL.md`. |
| `internal/assistant` | Registering the server with each assistant. |
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

- Always-on skills added to the context by a session-start hook, which `quiver install` sets up next to the MCP server.
- A test phase before the team track: end-to-end journeys, Windows, versioned data formats, a benchmark of assistant behaviour.
- An interactive screen for searching the library and toggling skills, with update notices.
- Search across published skills, so people can find skills others shared.
- Quiver for teams: company and team layers managed in one place, free to run yourself; hosting by us is coming soon. See [Quiver for teams](/teams/).
