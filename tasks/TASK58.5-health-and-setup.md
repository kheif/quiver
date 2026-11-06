# TASK58.5: Health section and first-run setup

**Phase:** Phase 7: TUI and slash commands (v1.5)

**Part of:** [TASK58](TASK58-interactive-tui-quiver-with-no-arguments.md): Interactive panel (`quiver` with no arguments)

**Status:** ✅ Done

**Depends on:** TASK58.1

The panel shows whether Quiver works for the assistants, and can set it up. Without this, a skill marked *Always on* can silently do nothing.

1. **Checks.** Health lists the results of `quiver doctor` (`error`, `warn`, `ok`), problems first, in two views: *Problems* and *All checks*. It opens on the problems, or on every check when there are none. The detail shows the check's own lines. The checks that ask the network (the latest release) are left out. The checks run in the background, when the panel opens and after a change that affects them, never after every assignment, because they ask the assistants.
2. **Assistants.** One result says how always-on skills reach each installed assistant, as `quiver status` does, and names an assistant whose session-start hook is missing.
3. **Header badge.** When a check finds a problem, every other section's header says `! N problems · 4 Health`.
4. **Actions.** `i` registers the MCP server and the hooks like `quiver install`, `x` removes them like `quiver uninstall`, `f` applies the safe fixes of `quiver doctor --fix` (the fixes that need an answer stay with the command), `r` checks again. Each of `i`, `x` and `f` asks first and says what it edits.
5. **First run.** Without a Quiver home, the panel opens on a screen that says Quiver is not set up and offers to create the library and register it in the assistants (the first steps of `quiver init`). Nothing is created before the answer.
6. **Doctor.** A source clone that git cannot read is reported as a source that is not downloaded; the panel says "Clone missing or damaged".

**Done when:** tests cover the order of the checks and the views, the badge, the checking state, install, fix and uninstall with their questions, the first-run screen and its setup, and, in the CLI backend, the parsing of the doctor output, the setup, the install and uninstall, and the fix.
