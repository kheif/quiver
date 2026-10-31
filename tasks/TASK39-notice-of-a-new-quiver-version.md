# TASK39 — Notice of a new Quiver version

**Phase:** Phase 4c — v1.2

**Status:** ✅ Done

**Depends on:** TASK33

1. At most once a day, a command asks GitHub for the latest release (`api.github.com/repos/kheif/quiver/releases/latest`) with a short timeout, in the background, and stores the answer in `~/.quiver/state/version-check.json`.
2. When a newer version is known, commands end with one line on stderr: `Quiver 1.3.0 is out (you have 1.2.0): npm install -g @kheif/quiver` (or the curl line, or `go install`, depending on how it was installed; `QUIVER_NPM` tells npm).
3. Never in `quiver mcp` (stdout is the protocol), with `--json`, outside a terminal, for `dev` builds, or with `QUIVER_NO_UPDATE_CHECK=1`. A failed check is silent.
4. `quiver doctor` shows the latest version as an info line.

**Done when:** tests with a fake release endpoint cover newer, same, failed and disabled, and that the check runs at most once a day.

**Notes:**
- `internal/release` does the check: `Check` asks GitHub only when the saved check is a day old (or from the future), saves a failed request too, so it is tried again only the next day, and keeps the version found before. A broken state file is replaced; without a Quiver home nothing is asked. `Newer` compares major.minor.patch; a release is newer than its pre-releases. `LatestURL` is replaced in tests.
- `cli.execute` starts the check before the command and waits for it at the end; `release.Timeout` (2 seconds) bounds the wait, once a day at most. `quiver mcp` and its shell completion never start it; `mcp install` and the other subcommands do. A command with `--json` set runs the check but prints no line.
- A development build is `dev`, a Go pseudo-version (a commit between releases) or a version with `+` (`+dirty`). The install line: npm (`QUIVER_NPM=1`), `go install` (no ldflags version), the releases page on Windows until TASK40, else the curl line, with `QUIVER_INSTALL_DIR` when the binary is not in `~/.local/bin`. `assistant.ShellQuote` is now exported for it.
- `doctor` shows the latest version as an `info` line, which counts as neither warning nor error, and skips it with `QUIVER_NO_UPDATE_CHECK` or for a development build.
- Checked against the real GitHub API on 2026-10-31: a 1.0.0 build reported 1.1.0.
