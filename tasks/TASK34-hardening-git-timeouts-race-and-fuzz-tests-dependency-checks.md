# TASK34 — Hardening: git timeouts, race and fuzz tests, dependency checks

**Phase:** Phase 4c — v1.2

**Status:** ✅ Done

**Depends on:** TASK33

From the v1 review (2026-10-29). Small fixes that make Quiver safer to run and to release.

1. **Git timeouts (backlog B08).** `internal/git` already sets `GIT_TERMINAL_PROMPT=0`, but a stalled network makes `git clone` or `git pull` wait forever, and `quiver update` hangs. Run git with `exec.CommandContext`: 5 minutes for clone, fetch and pull, 30 seconds for local commands. On a timeout, say which source and that the network did not answer.
2. **Race detector.** CI runs `go test -race ./...` (the server watches files while it answers requests).
3. **Fuzz test.** `FuzzParse` for the frontmatter parser in `internal/skill`: skills come from the internet, so no input may make it panic. A short `-fuzztime` run in CI.
4. **govulncheck.** A CI job runs `govulncheck ./...` and fails on a reachable known vulnerability.
5. **Dependabot.** `.github/dependabot.yml` for `gomod`, `github-actions` and `npm` (`npm/quiver`), monthly, with all updates of an ecosystem grouped into one PR.
6. **Pin actions to commits.** Every `uses:` in `.github/workflows/` names a full commit SHA with the version as a comment (`actions/checkout@<sha> # v7`). A tag can be moved to other code; a SHA cannot, and the release job can publish to npm. Dependabot keeps the SHAs current.
7. **Open-source files.** `SECURITY.md` (how to report a vulnerability privately), `.github/ISSUE_TEMPLATE/` (bug report with `quiver --version` and `quiver doctor` output, feature request), `CONTRIBUTING.md` (build, test, the task files).

**Done when:** a test makes a fake git hang and gets a timeout error; CI runs race, fuzz and govulncheck; Dependabot opens grouped PRs; no workflow uses a tag.

**Notes:**
- `internal/git` runs local commands with a 1 minute limit and clone, fetch and pull with 5 minutes (`runNetwork`); `WaitDelay` stops waiting for helpers such as `git-remote-https` that keep the output open.
- Releases up to v1.1.0 were built with Go 1.25.0, because `setup-go` read the `go` line of `go.mod`, which is only the minimum. govulncheck found over 30 fixed standard library vulnerabilities in that binary. The workflows now use `go-version: stable` with `check-latest`, and the release job runs govulncheck before publishing.
- The race detector runs on Linux and macOS; the Windows runner has no cgo.
- `skill.parse` is the parser without file access; `FuzzParse` found nothing in 10 million runs.
- Private vulnerability reporting must be turned on in the repository settings for SECURITY.md's link to work.
