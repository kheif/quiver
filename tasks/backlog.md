# Backlog

Open concerns from the design review. They are not tasks yet. Before starting the related task, decide on each item, then either turn it into a task or move it into that task's file.

Items are ordered by priority within each section.

## Architecture

### B01 — Run the behaviour spike first
- **Resolved 2026-10-15:** Spike done before TASK01. With the catalog in `instructions`, fitting prompts loaded the skill 6 of 6 times and non-fitting prompts never did. Tool descriptions are invisible because Claude Code defers MCP tools. See `docs/spike-b01-results.md`; design updated in TASK18.

### B02 — Verify always-on via `instructions`
- **Resolved 2026-10-15:** `instructions` are cut at about 4096 characters, too short for caveman. Always-on now uses a pointer plus `get_skill` (10 of 10 runs, held over five turns). Subagents are unreliable. Compaction checked in TASK21: the rules survive `/compact` through the summary (4 of 4).

### B03 — Project resolution beyond cwd
- **Resolved 2026-10-15:** MCP roots cannot be used (protocol 2026-07-28, SEP-2322). TASK17 now uses `QUIVER_PROJECT`, then the working directory, walking up to `.quiver.json`.

## Data model

### B04 — Split synced and machine-local state
- **Concern:** `~/.quiver/` is meant to be versioned with git, but `registry.json` holds absolute `paths`. Syncing between machines will conflict constantly.
- **Proposal:** Move `paths`, last update checks and other machine-local data into a gitignored `~/.quiver/state.json`. Keep only portable data in `registry.json`.
- **Related:** TASK07, data-model.md, Multiple Machines in PROJECT.md
- **Planned 2026-10-29:** TASK42.

### B05 — Stable skill IDs
- **Concern:** `<source>/<plugin>/<name>` is used only when names collide. If an update introduces a new collision, an existing ID changes and its assignments break silently.
- **Proposal:** Fix the ID when a skill is first discovered and keep it. Alternatively, always use plugin-qualified IDs for marketplace repos.
- **Related:** TASK11, data-model.md
- **Planned 2026-10-29:** TASK43.

## Security

### B06 — Review changes to remote always-on skills
- **Concern:** After `quiver update`, a remote always-on skill's new text goes straight into every session's system prompt. This is a supply-chain risk. PROJECT.md also says `quiver update` shows what changed, but the roadmap puts the diff preview in v2.
- **Proposal:** Move the diff preview into v1, at least for always-on skills. Fix the contradiction in PROJECT.md.
- **Related:** TASK15, PROJECT.md (Security, Roadmap)

### B07 — Approval for global `create_skill`
- **Concern:** A prompt injection in a conversation can make the assistant write a persistent global skill.
- **Proposal:** Save skills created with `scope: "global"` as pending. Activate them only after `quiver approve <id>` (or a TUI confirmation).
- **Related:** create_skill task (Phase 3), TASK32 (decide before release)
- **Decided 2026-10-29: not planned.** Global skills created with `create_skill` are active at once. They are committed in `local/`, so they can be reviewed and reverted there.

## Practical

### B08 — Non-interactive git
- **Concern:** If a private repo asks for credentials, a background `git ls-remote` or `git pull` hangs.
- **Proposal:** Always run git with `GIT_TERMINAL_PROMPT=0` and a timeout.
- **Related:** TASK09, background update checks (Phase 5)
- **Planned 2026-10-29:** `GIT_TERMINAL_PROMPT=0` is set; timeouts are TASK34.

### B09 — `mcp install` writes an absolute path
- **Concern:** GUI apps such as Claude Desktop do not see the shell's `PATH`, so a bare `quiver` command may not start.
- **Proposal:** Write the absolute path of the binary into the MCP config.
- **Related:** TASK20
- **Planned 2026-10-29:** kept for the new assistants in TASK37.

### B10 — Cap the `list_skills` description
- **Resolved 2026-10-15:** The catalog moved into `instructions`, with a 3800-character budget and a `list_skills` hint (TASK18).

### B11 — File watching and platform support
- **Concern:** `list_changed` needs file watching, but decisions.md lists no library for it. `flock` is Unix-only, and it is not decided whether Windows is a target.
- **Proposal:** Add `fsnotify` to decisions.md. Decide on Windows support. If it is a target, choose a cross-platform lock.
- **Partly resolved 2026-10-28:** `fsnotify` is in decisions.md (TASK26). Windows support and the lock are still open.
- **Related:** decisions.md, TASK02

### B12 — Roadmap duplication
- **Concern:** "First usable version" and v1 in PROJECT.md list the same items.
- **Proposal:** Keep only the additions under v1.
- **Related:** PROJECT.md (Roadmap)

## Other

### B13 — Native duplicates
- **Concern:** A skill can be served by Quiver and also be installed natively (`~/.claude/skills`, plugins), so it loads twice. Example: the caveman plugin.
- **Proposal:** `quiver doctor` detects these duplicates and suggests disabling the native copy.
- **Related:** doctor task (Phase 3)
- **Planned 2026-10-29:** TASK35 (import removes originals, `doctor` and `status` find copies).

### B14 — Name collision
- **Concern:** "Quiver" is already used by other products, for example a notes app.
- **Proposal:** Check the name and the binary name before release.
- **Related:** Phase 4

### B15 — Short slash command names
- **Concern:** TASK25 names prompts after the full skill ID, so `caveman/caveman` becomes `/mcp__quiver__caveman__caveman`. That is long to type.
- **Proposal:** Use the plain skill `name` as the prompt name when it is unique among served skills (`/mcp__quiver__caveman`). Fall back to the full ID only on a clash. TASK25 depends only on TASK18, so it can move into Phase 2 if slash commands are wanted early.
- **Not chosen:** Writing command files into `~/.claude/commands/` would give `/caveman`, but it brings back the generated files that Quiver avoids.
- **Related:** TASK25

### B16 — Stricter skill name rule
- **Concern:** The Agent Skills format does not allow a trailing dash (`x-`) or two dashes in a row (`a--b`). `skill.Validate` accepts both, because TASK05 uses `^[a-z0-9][a-z0-9-]*$`.
- **Proposal:** Report these names as warnings, not errors, so existing remote skills keep loading.
- **Related:** TASK05, TASK25

### B17 — Show new skills in an open Claude Code session
- **Concern:** The catalog reaches the assistant only when a session starts. A skill added during a session is found only through `list_skills`, so the assistant usually does not know about it.
- **Proposal:** `mcp install` adds a Claude Code `UserPromptSubmit` hook that runs `quiver`. When the served skills changed since the session started, it adds a short note for the assistant ("new skill: deploy: …"). The same hook can show the user a new Quiver version (TASK39).
- **Status:** not planned for v1.2.

### B18 — Measure skill loading against the real assistant
- **Concern:** That the assistant loads a fitting skill and follows always-on skills was measured by hand (docs/spike-b01-results.md). A new model can change this.
- **Proposal:** A workflow started by hand (`workflow_dispatch`) runs `claude -p` with a few prompts against a test library and counts `get_skill` calls. It needs an API key secret and costs a few cents per run, so it runs now and then, such as before a release. Copilot on GitHub would only test Copilot, not Claude.
- **Status:** not planned yet.
