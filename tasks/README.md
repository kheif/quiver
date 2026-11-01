# Quiver — Implementation Tasks

This folder breaks [PROJECT.md](../PROJECT.md) into ordered tasks, one file per task. Work through them top to bottom.

- [decisions.md](decisions.md): technical decisions that apply to every task
- [data-model.md](data-model.md): file formats, project IDs and skill IDs
- [backlog.md](backlog.md): open concerns from the design review; check it before starting a related task
- [docs/spike-b01-results.md](../docs/spike-b01-results.md): measured assistant behaviour that shaped the MCP design

## Conventions

- **Status** is set in `scripts/gen_tasks.py` (`STATUS`) and shown in the index below and at the top of each task: ✅ done, 🚧 in progress, empty for todo.
- **Depends on:** tasks that must be finished first.
- **Subtasks:** a large task is split into subtasks (TASK54.1, TASK54.2, …), one file each, listed under their parent in the index. Each subtask is finished and committed on its own; the parent is done when all its subtasks are.
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
| ✅ | [TASK34](TASK34-hardening-git-timeouts-race-and-fuzz-tests-dependency-checks.md) | Hardening: git timeouts, race and fuzz tests, dependency checks | TASK33 |
| ✅ | [TASK35](TASK35-duplicate-skills-import-removes-originals-doctor-and-status-find-copies.md) | Duplicate skills: import removes originals, doctor and status find copies | TASK33 |
| ✅ | [TASK36](TASK36-quiver-show-quiver-search-every-skill-in-the-catalog.md) | `quiver show`, `quiver search`, every skill in the catalog | TASK33 |
| ✅ | [TASK37](TASK37-quiver-mcp-install-for-cursor-and-claude-desktop.md) | `quiver mcp install` for Cursor and Claude Desktop | TASK33 |
| ✅ | [TASK38](TASK38-first-run-and-rough-edges.md) | First run and rough edges | TASK35 |
| ✅ | [TASK39](TASK39-notice-of-a-new-quiver-version.md) | Notice of a new Quiver version | TASK33 |
| ✅ | [TASK40](TASK40-install-ps1-for-windows.md) | `install.ps1` for Windows | TASK33 |
| ✅ | [TASK41](TASK41-release-v1-2-0.md) | Release v1.2.0 | TASK34, TASK35, TASK36, TASK37, TASK38, TASK39, TASK40 |

### Phase 5 — v2

| Status | Task | Title | Depends on |
|---|---|---|---|
| ✅ | [TASK42](TASK42-split-machine-local-state-out-of-the-registry.md) | Split machine-local state out of the registry | TASK41 |
| ✅ | [TASK43](TASK43-stable-skill-ids.md) | Stable skill IDs | TASK42 |
| ✅ | [TASK44](TASK44-update-skill-mcp-tool.md) | `update_skill` MCP tool | TASK24 |
| ✅ | [TASK45](TASK45-search-skills-mcp-tool.md) | `search_skills` MCP tool | TASK36, TASK18 |
| ✅ | [TASK46](TASK46-quiver-fork.md) | `quiver fork` | TASK12, TASK14 |
| ✅ | [TASK47](TASK47-pinning.md) | Pinning | TASK15 |
| ✅ | [TASK48](TASK48-update-preview.md) | Update preview | TASK15 |
| ✅ | [TASK49](TASK49-library-sync-between-machines.md) | Library sync between machines | TASK30 |
| ✅ | [TASK50](TASK50-mcp-install-for-more-assistants.md) | `mcp install` for more assistants | TASK37 |
|  | [TASK51](TASK51-interactive-tui-quiver-with-no-arguments.md) | Interactive TUI (`quiver` with no arguments) | TASK45, TASK14 |
|  | [TASK52](TASK52-background-update-checks.md) | Background update checks | TASK51, TASK15 |
|  | [TASK53](TASK53-prefix-free-commands-in-claude-code.md) | Prefix-free commands in Claude Code | TASK25, TASK14, TASK20 |

### Phase 6: Session-start delivery

| Status | Task | Title | Depends on |
|---|---|---|---|
|  | [TASK54](TASK54-spike-session-start-hooks-per-assistant.md) | Spike: session-start hooks per assistant | TASK50 |
|  | ↳ [TASK54.1](TASK54.1-claude-code-hooks.md) | Claude Code hooks | TASK50 |
|  | ↳ [TASK54.2](TASK54.2-codex-hooks.md) | Codex hooks | TASK50 |
|  | ↳ [TASK54.3](TASK54.3-vs-code-copilot.md) | VS Code Copilot | TASK50 |
|  | ↳ [TASK54.4](TASK54.4-cursor-antigravity-and-global-rules-files.md) | Cursor, Antigravity and global rules files | TASK50 |
|  | [TASK55](TASK55-quiver-hook-session-start.md) | `quiver hook session-start` | TASK54, TASK19 |
|  | [TASK56](TASK56-always-on-pointer-becomes-the-fallback.md) | Always-on pointer becomes the fallback | TASK55 |
|  | [TASK57](TASK57-quiver-install.md) | `quiver install` | TASK55, TASK50 |
|  | ↳ [TASK57.1](TASK57.1-rename-to-quiver-install-and-quiver-uninstall.md) | Rename to `quiver install` and `quiver uninstall` | TASK50 |
|  | ↳ [TASK57.2](TASK57.2-claude-code-hook.md) | Claude Code hook | TASK54.1, TASK55, TASK57.1 |
|  | ↳ [TASK57.3](TASK57.3-codex-hook.md) | Codex hook | TASK54.2, TASK55, TASK57.1 |
|  | ↳ [TASK57.4](TASK57.4-delivery-report-doctor-and-status.md) | Delivery report, doctor and status | TASK57.2 |
|  | [TASK58](TASK58-vs-code-copilot.md) | VS Code Copilot | TASK54.3, TASK57.1 |
|  | [TASK59](TASK59-rules-file-pointer-for-assistants-without-hooks.md) | Rules-file pointer for assistants without hooks | TASK54.4, TASK57.4 |

### Phase 7: Test phase

| Status | Task | Title | Depends on |
|---|---|---|---|
|  | [TASK60](TASK60-coverage-baseline.md) | Coverage baseline | TASK59 |
|  | [TASK61](TASK61-end-to-end-user-journeys.md) | End-to-end user journeys | TASK59, TASK30 |
|  | ↳ [TASK61.1](TASK61.1-journey-setup-and-delivery.md) | Journey: setup and delivery | TASK59, TASK30 |
|  | ↳ [TASK61.2](TASK61.2-journey-library-lifecycle.md) | Journey: library lifecycle | TASK59, TASK30 |
|  | ↳ [TASK61.3](TASK61.3-journey-two-machines.md) | Journey: two machines | TASK59, TASK30 |
|  | ↳ [TASK61.4](TASK61.4-journey-failures.md) | Journey: failures | TASK59, TASK30 |
|  | [TASK62](TASK62-windows.md) | Windows | TASK59 |
|  | ↳ [TASK62.1](TASK62.1-fake-assistants-in-go.md) | Fake assistants in Go | TASK59 |
|  | ↳ [TASK62.2](TASK62.2-hooks-and-paths-on-windows.md) | Hooks and paths on Windows | TASK62.1 |
|  | ↳ [TASK62.3](TASK62.3-installed-binary-runs-the-hook.md) | Installed binary runs the hook | TASK62.1 |
|  | [TASK63](TASK63-versioned-data-formats-and-migration.md) | Versioned data formats and migration | TASK59 |
|  | [TASK64](TASK64-assistant-behaviour-benchmark.md) | Assistant behaviour benchmark | TASK59 |
|  | ↳ [TASK64.1](TASK64.1-benchmark-runner.md) | Benchmark runner | TASK59 |
|  | ↳ [TASK64.2](TASK64.2-assistant-adapters.md) | Assistant adapters | TASK64.1 |
|  | ↳ [TASK64.3](TASK64.3-benchmark-cases.md) | Benchmark cases | TASK64.2 |
|  | ↳ [TASK64.4](TASK64.4-baselines-and-reports.md) | Baselines and reports | TASK64.3 |
|  | [TASK65](TASK65-robustness.md) | Robustness | TASK59 |
|  | ↳ [TASK65.1](TASK65.1-fuzzing.md) | Fuzzing | TASK59 |
|  | ↳ [TASK65.2](TASK65.2-parallel-processes.md) | Parallel processes | TASK59 |
|  | ↳ [TASK65.3](TASK65.3-performance-budgets.md) | Performance budgets | TASK59 |
|  | [TASK66](TASK66-security-review.md) | Security review | TASK59 |
|  | [TASK67](TASK67-test-phase-review.md) | Test phase review | TASK60, TASK61, TASK62, TASK63, TASK64, TASK65, TASK66 |

### Phase 8: Enterprise

| Status | Task | Title | Depends on |
|---|---|---|---|
|  | [TASK68](TASK68-enterprise.md) | Enterprise | TASK67 |

## Later (not yet planned in detail)

- Graphical app with the same features as the TUI.
