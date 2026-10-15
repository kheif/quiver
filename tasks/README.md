# Quiver — Implementation Tasks

This folder breaks [PROJECT.md](../PROJECT.md) into ordered tasks, one file per task. Work through them top to bottom.

- [decisions.md](decisions.md): technical decisions that apply to every task
- [data-model.md](data-model.md): file formats, project IDs and skill IDs
- [backlog.md](backlog.md): open concerns from the design review; check it before starting a related task
- [docs/spike-b01-results.md](../docs/spike-b01-results.md): measured assistant behaviour that shaped the MCP design

## Conventions

- **Status** is tracked only in the index below: `[ ]` todo, `[~]` in progress, `[x]` done.
- **Depends on:** tasks that must be finished first.
- **Done when:** acceptance criteria. A task is not done until every item passes and has a test.
- Every command needs unit tests. Every user-facing flow also needs an end-to-end test (see TASK30).
- MCP tasks are tested through the in-memory harness from TASK16.

## First Milestone

Phases 0–2 are the first usable version. After TASK21 you can run:

```bash
quiver source add JuliusBrussee/caveman
quiver add caveman/caveman --global --always
quiver mcp install
```

and every new assistant session follows caveman from the first message. Use Quiver this way every day before you start Phase 3.

## Index

### Phase 0 — Foundation

| Status | Task | Title | Depends on |
|---|---|---|---|
| [x] | [TASK01](TASK01-repository-setup.md) | Repository setup | — |
| [ ] | [TASK02](TASK02-paths-atomic-writes-and-locking.md) | Paths, atomic writes and locking | TASK01 |
| [ ] | [TASK03](TASK03-config.md) | Config | TASK02 |
| [ ] | [TASK04](TASK04-quiver-init.md) | `quiver init` | TASK03 |
| [ ] | [TASK05](TASK05-skill-parsing-and-validation.md) | Skill parsing and validation | TASK01 |
| [ ] | [TASK06](TASK06-project-identification-quiver-json.md) | Project identification (`.quiver.json`) | TASK02 |
| [ ] | [TASK07](TASK07-registry-and-assignments.md) | Registry and assignments | TASK02, TASK06 |

### Phase 1 — Remote Sources

| Status | Task | Title | Depends on |
|---|---|---|---|
| [ ] | [TASK08](TASK08-library-local-skills.md) | Library (local skills) | TASK05 |
| [ ] | [TASK09](TASK09-git-wrapper.md) | Git wrapper | TASK01 |
| [ ] | [TASK10](TASK10-source-url-parsing.md) | Source URL parsing | — |
| [ ] | [TASK11](TASK11-skill-discovery.md) | Skill discovery | TASK05 |
| [ ] | [TASK12](TASK12-library-with-remote-sources.md) | Library with remote sources | TASK08, TASK11 |
| [ ] | [TASK13](TASK13-quiver-source-add-list-remove.md) | `quiver source add \| list \| remove` | TASK09, TASK10, TASK12 |
| [ ] | [TASK14](TASK14-quiver-add-quiver-remove.md) | `quiver add` / `quiver remove` | TASK07, TASK08 |
| [ ] | [TASK15](TASK15-quiver-update.md) | `quiver update` | TASK13 |

### Phase 2 — MCP Server (first usable version)

| Status | Task | Title | Depends on |
|---|---|---|---|
| [ ] | [TASK16](TASK16-mcp-server-skeleton-quiver-mcp.md) | MCP server skeleton (`quiver mcp`) | TASK04, TASK08 |
| [ ] | [TASK17](TASK17-project-resolution-in-the-server.md) | Project resolution in the server | TASK16, TASK06, TASK07 |
| [ ] | [TASK18](TASK18-list-skills-get-skill-get-skill-file.md) | `list_skills`, `get_skill`, `get_skill_file` | TASK17 |
| [ ] | [TASK19](TASK19-always-on-skills-mcp-instructions.md) | Always-on skills (MCP instructions) | TASK18, TASK14 |
| [ ] | [TASK20](TASK20-quiver-mcp-install.md) | `quiver mcp install` | TASK16 |
| [ ] | [TASK21](TASK21-behaviour-check-against-the-real-server.md) | Behaviour check against the real server | TASK18, TASK19 |

### Phase 3 — Local Skills and Management

| Status | Task | Title | Depends on |
|---|---|---|---|
| [ ] | [TASK22](TASK22-quiver-list-and-quiver-status.md) | `quiver list` and `quiver status` | TASK07, TASK08 |
| [ ] | [TASK23](TASK23-quiver-new-and-quiver-edit.md) | `quiver new` and `quiver edit` | TASK04, TASK07, TASK08 |
| [ ] | [TASK24](TASK24-create-skill.md) | `create_skill` | TASK18, TASK23 |
| [ ] | [TASK25](TASK25-skills-as-mcp-prompts.md) | Skills as MCP prompts | TASK18 |
| [ ] | [TASK26](TASK26-live-updates.md) | Live updates | TASK24, TASK25 |
| [ ] | [TASK27](TASK27-quiver-where-and-quiver-projects.md) | `quiver where` and `quiver projects` | TASK07 |
| [ ] | [TASK28](TASK28-quiver-import.md) | `quiver import` | TASK23, TASK14 |
| [ ] | [TASK29](TASK29-quiver-doctor.md) | `quiver doctor` | TASK22, TASK27 |

### Phase 4 — v1 Release

| Status | Task | Title | Depends on |
|---|---|---|---|
| [ ] | [TASK30](TASK30-end-to-end-test-suite.md) | End-to-end test suite | TASK20, TASK15, TASK29 |
| [ ] | [TASK31](TASK31-readme-and-help-text.md) | README and help text | TASK30 |
| [ ] | [TASK32](TASK32-release.md) | Release | TASK31 |

### Phase 5 — v2

| Status | Task | Title | Depends on |
|---|---|---|---|
| [ ] | [TASK33](TASK33-update-skill-mcp-tool.md) | `update_skill` MCP tool | TASK24 |
| [ ] | [TASK34](TASK34-quiver-search-and-search-skills.md) | `quiver search` and `search_skills` | TASK12, TASK18 |
| [ ] | [TASK35](TASK35-quiver-fork.md) | `quiver fork` | TASK12, TASK14 |
| [ ] | [TASK36](TASK36-pinning.md) | Pinning | TASK15 |
| [ ] | [TASK37](TASK37-update-preview.md) | Update preview | TASK15 |
| [ ] | [TASK38](TASK38-library-sync-between-machines.md) | Library sync between machines | TASK30 |
| [ ] | [TASK39](TASK39-mcp-install-for-more-assistants.md) | `mcp install` for more assistants | TASK20 |
| [ ] | [TASK40](TASK40-interactive-tui-quiver-with-no-arguments.md) | Interactive TUI (`quiver` with no arguments) | TASK34, TASK14 |
| [ ] | [TASK41](TASK41-background-update-checks.md) | Background update checks | TASK40, TASK15 |

## Later (not yet planned in detail)

- Graphical app with the same features as the TUI.
- Team libraries with shared assignments.
