# TASK57: Release v1.4.0

**Phase:** Phase 6: Session-start delivery (v1.4)

**Status:** ✅ Done

**Depends on:** TASK52, TASK53, TASK54, TASK55, TASK55.7, TASK55.8, TASK56

1. **Old names still work.** `quiver mcp install` and `quiver mcp uninstall`, with `--for` and `--print`, work as in v1.3.0 and name the new commands (TASK55.1).
2. **Upgrade check.** As in TASK51, from a home made by the v1.3.0 binary: `quiver install` gives every installed assistant its hook, the same skills are served, and `doctor` is clean.
3. README, help text and the website docs cover every change of this phase.
4. The GitHub release notes say that `quiver install` replaces `quiver mcp install`, and which assistants get always-on skills from a hook.
5. Tag `v1.4.0` only after the maintainer confirms. The release workflow publishes to GitHub and npm and installs on clean machines.
6. **Website.** The status marks follow the release (see `web/README.md`). Website changes are committed only after the maintainer confirms.

**Done when:** the release workflow is green.
