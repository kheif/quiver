# TASK57.1: Rename to `quiver install` and `quiver uninstall`

**Phase:** Phase 6: Session-start delivery

**Part of:** [TASK57](TASK57-quiver-install.md): `quiver install`

**Status:** Todo

**Depends on:** TASK50

1. Move the commands from `internal/cli/mcp.go` to `internal/cli/install.go` as `quiver install [assistant...]` and `quiver uninstall [assistant...]`, with the same names, completion and output. Remove `quiver mcp install`, `quiver mcp uninstall` and the deprecated `--print`; keep no alias.
2. Update every caller and message: `guidedSetup` in `internal/cli/init.go`, the fix hints of `doctor` ("run: quiver mcp install ..."), the npm note in `mcpInstall`, and any other output a search for `mcp install` finds.
3. Update the docs: README (command table and the MCP server section), PROJECT.md (CLI and How It Works), CONTRIBUTING.md if it names the command, and the First Milestone of `tasks/README.md` in `scripts/gen_tasks.py`. The website is updated by the maintainer.
4. The GitHub release notes of the next release say that the command changed.

**Done when:** the tests and completion use the new commands, and a search for `mcp install` finds nothing outside the files of finished tasks.
