# TASK65.3: Installed binary runs the hook

**Phase:** Phase 8: Test phase

**Part of:** [TASK65](TASK65-windows.md): Windows

**Status:** Todo

**Depends on:** TASK65.1

The release checks install the binary but do not run it with an assistant.

1. In the Release workflow, after each install (`install.sh`, `install.ps1`, npm), run `quiver install` with the fake assistants and run the installed hook command as the assistant would, from a project folder.
2. Add the same check to the `release-dry-run` job of CI, so it runs on every push and not only on a tag.

**Done when:** the checks pass on Linux, macOS and Windows.
