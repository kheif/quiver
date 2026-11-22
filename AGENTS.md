# Notes for coding agents

Quiver is a single Go binary: a CLI, an MCP server and a terminal panel that serve skills to AI assistants. Read [PROJECT.md](PROJECT.md) for the product and [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow; this file only adds what an agent needs to work here without surprises.

## Build and test

```bash
make build              # bin/quiver
make test               # go test ./..., including the end-to-end suite in e2e/
make lint               # golangci-lint, must report 0 issues
go test -race ./...     # as CI runs it on Linux and macOS
cd web && npm run check && npm test   # the website, when it changes
```

Set `QUIVER_HOME` to a temporary folder before you run the binary by hand, so that a try-out never touches `~/.quiver`. Tests already do this.

## Layout

- `cmd/quiver`, `internal/cli` (commands), `internal/server` (MCP tools and the session-start text), `internal/library`, `internal/registry` (what is assigned where), `internal/embed` (the in-process search model), `internal/tui` (the panel), `e2e/` (tests through the built binary).
- `tasks/` holds the plan, one file per task, with `decisions.md` and `backlog.md`. `go test ./tasks` checks that the index, the files and their links agree.
- `web/` is the website (Astro and Starlight). `web/src/content/docs/docs/reference/cli.md` is generated: run `cd web && npm run cli-reference` after a change to a help text.

## Rules that are easy to miss

- Every file Quiver writes in the Quiver home carries a format version; a file from a newer Quiver is refused, never rewritten. A file under `cache/` is derived data and is rebuilt.
- Writes to the Quiver home go through the lock and `WriteJSONAtomic`. Do not write a file there another way.
- A search makes no network call. The only downloads are git operations on sources and the sync remote, the daily release check and the one download of the search model file.
- Text that reaches the model is cleaned; do not add a path that passes skill text to the model without it.
- Update the README, the help text and the website when behaviour changes, and add tests with the change.
- Do not commit files outside the change you made. The working tree can hold unrelated local files.
- Commit subjects follow Conventional Commits. Keep the subject in the imperative and describe what changed for the user.
