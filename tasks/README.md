# Quiver — Implementation Tasks

This folder breaks [PROJECT.md](../PROJECT.md) into ordered tasks, one file per task. Work through them top to bottom.

- [decisions.md](decisions.md): technical decisions that apply to every task
- [data-model.md](data-model.md): file formats, project IDs and skill IDs
- [backlog.md](backlog.md): open concerns from the design review; check it before starting a related task
- [docs/spike-b01-results.md](../docs/spike-b01-results.md): measured assistant behaviour that shaped the MCP design

## Conventions

- **Status** is set in `scripts/gen_tasks.py` (`STATUS`) and shown in the index below and at the top of each task: ✅ done, 🚧 in progress, empty for todo.
- **Depends on:** tasks that must be finished first.
- **Done when:** acceptance criteria. A task is not done until every item passes and has a test.
- Every command needs unit tests. Every user-facing flow also needs an end-to-end test (see TASK30).
- MCP tasks are tested through the in-memory harness from TASK16.

## First Milestone

Phases 0–2 are the first usable version. After TASK21 you can run:

```bash
quiver source add JuliusBrussee/caveman --name caveman
quiver add caveman/caveman --global --always
quiver mcp install
```

and every new assistant session follows caveman from the first message. Use Quiver this way every day before you start Phase 3.

## Index

### Phase 0 — Foundation

| Status | Task | Title | Depends on |
|---|---|---|---|
| ✅ | [TASK01](TASK01-repository-setup.md) | Repository setup | — |
| ✅ | [TASK02](TASK02-paths-atomic-writes-and-locking.md) | Paths, atomic writes and locking | TASK01 |
| ✅ | [TASK03](TASK03-config.md) | Config | TASK02 |
| ✅ | [TASK04](TASK04-quiver-init.md) | `quiver init` | TASK03 |
| ✅ | [TASK05](TASK05-skill-parsing-and-validation.md) | Skill parsing and validation | TASK01 |
| ✅ | [TASK06](TASK06-project-identification-quiver-json.md) | Project identification (`.quiver.json`) | TASK02 |
| ✅ | [TASK07](TASK07-registry-and-assignments.md) | Registry and assignments | TASK02, TASK06 |

### Phase 1 — Remote Sources

| Status | Task | Title | Depends on |
|---|---|---|---|
| ✅ | [TASK08](TASK08-library-local-skills.md) | Library (local skills) | TASK05 |
| ✅ | [TASK09](TASK09-git-wrapper.md) | Git wrapper | TASK01 |
| ✅ | [TASK10](TASK10-source-url-parsing.md) | Source URL parsing | — |
| ✅ | [TASK11](TASK11-skill-discovery.md) | Skill discovery | TASK05 |
| ✅ | [TASK12](TASK12-library-with-remote-sources.md) | Library with remote sources | TASK08, TASK11 |
| ✅ | [TASK13](TASK13-quiver-source-add-list-remove.md) | `quiver source add \| list \| remove` | TASK09, TASK10, TASK12 |
| ✅ | [TASK14](TASK14-quiver-add-quiver-remove.md) | `quiver add` / `quiver remove` | TASK07, TASK08 |
| ✅ | [TASK15](TASK15-quiver-update.md) | `quiver update` | TASK13 |

### Phase 2 — MCP Server (first usable version)

| Status | Task | Title | Depends on |
|---|---|---|---|
| ✅ | [TASK16](TASK16-mcp-server-skeleton-quiver-mcp.md) | MCP server skeleton (`quiver mcp`) | TASK04, TASK08 |
| ✅ | [TASK17](TASK17-project-resolution-in-the-server.md) | Project resolution in the server | TASK16, TASK06, TASK07 |
| ✅ | [TASK18](TASK18-list-skills-get-skill-get-skill-file.md) | `list_skills`, `get_skill`, `get_skill_file` | TASK17 |
| ✅ | [TASK19](TASK19-always-on-skills-mcp-instructions.md) | Always-on skills (MCP instructions) | TASK18, TASK14 |
| ✅ | [TASK20](TASK20-quiver-mcp-install.md) | `quiver mcp install` | TASK16 |
| ✅ | [TASK21](TASK21-behaviour-check-against-the-real-server.md) | Behaviour check against the real server | TASK18, TASK19 |

### Phase 3 — Local Skills and Management

| Status | Task | Title | Depends on |
|---|---|---|---|
| ✅ | [TASK22](TASK22-quiver-list-and-quiver-status.md) | `quiver list` and `quiver status` | TASK07, TASK08 |
| ✅ | [TASK23](TASK23-quiver-new-and-quiver-edit.md) | `quiver new` and `quiver edit` | TASK04, TASK07, TASK08 |
| ✅ | [TASK24](TASK24-create-skill.md) | `create_skill` | TASK18, TASK23 |
| ✅ | [TASK25](TASK25-skills-as-mcp-prompts.md) | Skills as MCP prompts | TASK18 |
| ✅ | [TASK26](TASK26-live-updates.md) | Live updates | TASK24, TASK25 |
| ✅ | [TASK27](TASK27-quiver-where-and-quiver-projects.md) | `quiver where` and `quiver projects` | TASK07 |
| ✅ | [TASK28](TASK28-quiver-import.md) | `quiver import` | TASK23, TASK14 |
| ✅ | [TASK29](TASK29-quiver-doctor.md) | `quiver doctor` | TASK22, TASK27 |

### Phase 4 — v1 Release

| Status | Task | Title | Depends on |
|---|---|---|---|
| ✅ | [TASK30](TASK30-end-to-end-test-suite.md) | End-to-end test suite | TASK20, TASK15, TASK29 |
| ✅ | [TASK31](TASK31-readme-and-help-text.md) | README and help text | TASK30 |
| ✅ | [TASK32](TASK32-release.md) | Release | TASK31 |

### Phase 4b — v1.1

| Status | Task | Title | Depends on |
|---|---|---|---|
| ✅ | [TASK33](TASK33-v1-1-import-one-skill-or-a-folder-version-for-go-install.md) | v1.1: import one skill or a folder, version for go install | TASK32 |

### Phase 4c — v1.2

| Status | Task | Title | Depends on |
|---|---|---|---|
|  | [TASK34](TASK34-hardening-git-timeouts-race-and-fuzz-tests-dependency-checks.md) | Hardening: git timeouts, race and fuzz tests, dependency checks | TASK33 |
|  | [TASK35](TASK35-duplicate-skills-import-removes-originals-doctor-and-status-find-copies.md) | Duplicate skills: import removes originals, doctor and status find copies | TASK33 |
|  | [TASK36](TASK36-quiver-show-quiver-search-every-skill-in-the-catalog.md) | `quiver show`, `quiver search`, every skill in the catalog | TASK33 |
|  | [TASK37](TASK37-quiver-mcp-install-for-cursor-and-claude-desktop.md) | `quiver mcp install` for Cursor and Claude Desktop | TASK33 |
|  | [TASK38](TASK38-first-run-and-rough-edges.md) | First run and rough edges | TASK35 |
|  | [TASK39](TASK39-notice-of-a-new-quiver-version.md) | Notice of a new Quiver version | TASK33 |
|  | [TASK40](TASK40-install-ps1-for-windows.md) | `install.ps1` for Windows | TASK33 |
|  | [TASK41](TASK41-release-v1-2-0.md) | Release v1.2.0 | TASK34, TASK35, TASK36, TASK37, TASK38, TASK39, TASK40 |

### Phase 5 — v2

| Status | Task | Title | Depends on |
|---|---|---|---|
|  | [TASK42](TASK42-split-machine-local-state-out-of-the-registry.md) | Split machine-local state out of the registry | TASK41 |
|  | [TASK43](TASK43-stable-skill-ids.md) | Stable skill IDs | TASK42 |
|  | [TASK44](TASK44-update-skill-mcp-tool.md) | `update_skill` MCP tool | TASK24 |
|  | [TASK45](TASK45-search-skills-mcp-tool.md) | `search_skills` MCP tool | TASK36, TASK18 |
|  | [TASK46](TASK46-quiver-fork.md) | `quiver fork` | TASK12, TASK14 |
|  | [TASK47](TASK47-pinning.md) | Pinning | TASK15 |
|  | [TASK48](TASK48-update-preview.md) | Update preview | TASK15 |
|  | [TASK49](TASK49-library-sync-between-machines.md) | Library sync between machines | TASK30 |
|  | [TASK50](TASK50-mcp-install-for-more-assistants.md) | `mcp install` for more assistants | TASK37 |
|  | [TASK51](TASK51-interactive-tui-quiver-with-no-arguments.md) | Interactive TUI (`quiver` with no arguments) | TASK45, TASK14 |
|  | [TASK52](TASK52-background-update-checks.md) | Background update checks | TASK51, TASK15 |
|  | [TASK53](TASK53-prefix-free-commands-in-claude-code.md) | Prefix-free commands in Claude Code | TASK25, TASK14, TASK20 |

## Later (not yet planned in detail)

- Graphical app with the same features as the TUI.
- Team libraries with shared assignments.
