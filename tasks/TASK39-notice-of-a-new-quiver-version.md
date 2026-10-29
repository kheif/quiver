# TASK39 — Notice of a new Quiver version

**Phase:** Phase 4c — v1.2

**Status:** Todo

**Depends on:** TASK33

1. At most once a day, a command asks GitHub for the latest release (`api.github.com/repos/kheif/quiver/releases/latest`) with a short timeout, in the background, and stores the answer in `~/.quiver/state/version-check.json`.
2. When a newer version is known, commands end with one line on stderr: `Quiver 1.3.0 is out (you have 1.2.0): npm install -g @kheif/quiver` (or the curl line, or `go install`, depending on how it was installed; `QUIVER_NPM` tells npm).
3. Never in `quiver mcp` (stdout is the protocol), with `--json`, outside a terminal, for `dev` builds, or with `QUIVER_NO_UPDATE_CHECK=1`. A failed check is silent.
4. `quiver doctor` shows the latest version as an info line.

**Done when:** tests with a fake release endpoint cover newer, same, failed and disabled, and that the check runs at most once a day.
