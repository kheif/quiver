# TASK51 — Release v1.3.0

**Phase:** Phase 5 — v1.3

**Status:** ✅ Done

**Depends on:** TASK42, TASK43, TASK44, TASK45, TASK46, TASK47, TASK48, TASK49, TASK50

Versions follow Semantic Versioning: only a breaking change needs a new major version. Everything since v1.2.0 adds features except one dropped flag, so this phase is released as v1.3.0 once that flag is back.

1. **`--for` again.** v1.2.0 named one assistant with `quiver mcp install --for <assistant>`; TASK50 made the assistants arguments and dropped the flag. Bring it back on `install` and `uninstall` as a hidden, deprecated flag (`MarkDeprecated`, like `--print`) that counts as one more argument, so `quiver mcp install --for claude-desktop` does what it did in v1.2.0 and points to `quiver mcp install claude-desktop`. Test both commands with the flag.
2. **Upgrade check.** Make a Quiver home with the v1.2.0 binary (init, source add, add, add --always, a project), run the new binary on it, and check that the same skills, assignments and always-on skills are served and `doctor` is clean.
3. README and help text cover every change of this phase; PROJECT.md's roadmap lists them under v1.3.
4. The GitHub release notes list the changes since v1.2.0 and the deprecated `--for`.
5. Tag `v1.3.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.
6. **Website.** The status marks follow the release (`latestRelease`, `unreleased` and the **Next release** asides; see `web/README.md`). `quiver install` is not in v1.3.0, so its asides stay. Website changes are committed only after the maintainer confirms.

**Done when:** the release workflow is green and `quiver mcp install --for <assistant>` works in the release.

**Notes:**
- `addForFlag` and `withFor` in `internal/cli/mcp.go`; `TestMCPInstallFor` covers install, uninstall with arguments, the deprecation line and the help.
- Upgrade check on 2026-11-02, with a v1.2.0 binary built from the tag and a fixture source (two global skills, one always-on, and an always-on project skill): `status`, the MCP `instructions` and `list_skills` were the same before and after the first write, `registry.json` became version 2 with `state/projects.json`, and `doctor` reported no errors.
- The README and PROJECT.md already covered the phase; each task had updated them. The release notes are GoReleaser's changelog of the commits, as for v1.2.0.
- Website: the **Next release** asides and badges of the phase are gone, `latestRelease` is v1.3.0 with an empty `unreleased`, and the `quiver install` aside names v1.3.0. Three tests in `web/tests` that held the v1.2.0 state follow it.
- Tagged `v1.3.0` on 2026-11-02.
