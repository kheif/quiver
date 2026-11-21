# TASK63: Tool bundles

**Phase:** Phase 7c: Tool bundles (v1.7)

**Status:** Todo

**Depends on:** TASK62.10

Quiver manages skills. Many tools that help an assistant ship more than a skill: a command-line program, an MCP server, hooks, a block of instructions, files in each project, sometimes a background process with its own data. A code-graph tool such as Graphify (a Python program, a skill, an optional MCP server, hooks in the assistant's settings and in git, a `graphify-out/` folder in each project) and a memory tool such as claude-mem (a plugin with hooks, an MCP server, a background worker and a database) are examples. Today each of them has its own installer, run per assistant and per project, and Quiver knows nothing about what they wrote, so nothing can be listed, removed or kept the same on another project from one place.

This task adds one generic way to manage them: a **bundle**. One engine reads a manifest that lists the components of a tool; there is no code per tool.

| Component | What it is | Quiver writes it | Quiver removes it |
|---|---|---|---|
| Package | a program with install, upgrade, uninstall and status commands | no: the manifest's commands run, after the user has seen them | by the uninstall command |
| MCP | a server entry in an assistant's config | yes, for every assistant, in the global or the project scope | from the ledger |
| Hooks | entries in an assistant's hook config | yes | from the ledger |
| Instructions | a marked block in a rules file | yes | from the ledger |
| Files | files in a project, kept out of version control through `.git/info/exclude` | yes | from the ledger |
| Environment | variables a tool needs | listed, shown, never written to a shell profile | nothing to remove |
| Skills | `SKILL.md` folders | already managed by sources | already managed |

- **Vocabulary.** A *bundle* is what a manifest describes, in the manifest and in `internal/bundle`; the commands are `quiver tool ...`, because a tool is what a bundle installs.
- **Declarative first** (TASK63.2, TASK63.4): what Quiver can write itself is declared, so one declaration reaches every assistant that `quiver install` supports; a tool's own installer is per assistant. A command is for what a tool insists on doing itself, and it runs only after the user has seen it (TASK63.5).
- **Ownership ledger** (TASK63.3): every entry Quiver writes is recorded with where it went and a hash of what was written. Removal reverses exactly that, never an entry Quiver did not write, and a changed entry is reported, not overwritten.
- **A skill is served once.** If a tool installs its own copy of a skill, the manifest says so, and Quiver does not serve the same skill again.
- **Kinds.** A manifest may name a kind such as `memory`. Adding a second tool of one kind warns, because two tools of a kind put text into the same session.
- **Runtimes.** A manifest names what it needs (Python with `uv`, Node). Quiver checks and says what is missing; it never installs or starts a runtime, and neither does the local search model of TASK62.7.

PROJECT.md lists two limits that bundles change: "Copying or linking skills into projects" (the MCP server is the only delivery path; `.quiver.json` is the only project file) and "Running scripts". Skills keep both limits. A bundle may write the files and entries of its manifest and run the commands of its manifest, each after consent, and TASK63.8 words this in PROJECT.md.

Out of scope: a central catalog of bundles for a company, locked bundles and bundles pushed through an assistant's managed configuration (TASK72); hosting bundles; starting, stopping or watching a tool's background process; collecting a tool's metrics (a status command of the manifest may be shown, nothing is collected). Documentation on the website follows `web/README.md`: what is not released carries the "Next release" marks, and the pages are committed only after the maintainer confirms.

**Done when:** every subtask is done; the bundles of Graphify and claude-mem add and remove on Claude Code from a clean home, and after removal the assistant's config files hold what they held before, except entries the user changed in between (reported, not removed).

**Subtasks:**
- [TASK63.1](TASK63.1-bundle-spike.md): Bundle spike
- [TASK63.2](TASK63.2-bundle-manifest.md): Bundle manifest
- [TASK63.3](TASK63.3-ownership-ledger.md): Ownership ledger
- [TASK63.4](TASK63.4-component-writers.md): Component writers
- [TASK63.5](TASK63.5-tool-commands.md): Tool commands
- [TASK63.6](TASK63.6-tools-page-in-the-panel.md): Tools page in the panel
- [TASK63.7](TASK63.7-first-bundles.md): First bundles
- [TASK63.8](TASK63.8-release-v1-7-0.md): Release v1.7.0
