# Technical Decisions

These apply to every task. Change them here first if you change them at all.

| Topic | Decision | Reason |
|---|---|---|
| Language | Go 1.25+ | Ships as a single binary with no runtime. The CLI and the MCP server are the same binary. 1.25 is the minimum of the MCP SDK (v1.8.0). |
| CLI framework | `spf13/cobra` | The standard choice for Go CLIs. Provides subcommands, flags and shell completion. |
| MCP | Official Go SDK `github.com/modelcontextprotocol/go-sdk`, stdio transport | Maintained alongside the MCP spec. Stdio is what Claude Code, Cursor and Claude Desktop launch by default. |
| TUI | Bubble Tea v2 (`charm.land/bubbletea/v2`, `bubbles/v2`, `lipgloss/v2`) | The most widely used Go TUI stack. Its models can be tested without a real terminal. Version 1 queries the terminal for its background color when the package loads, which would delay every command; version 2 does not. |
| Frontmatter parsing | `gopkg.in/yaml.v3` | `SKILL.md` frontmatter is YAML. |
| File watching | `fsnotify/fsnotify` | Cross-platform and widely used. It does not watch subfolders, so the server adds each folder it needs (TASK26). |
| Git | Run the system `git` binary through `os/exec` | Reuses the user's credentials (SSH keys, `gh` auth) with no extra setup. |
| Storage | JSON files, written atomically (write to a temp file, then rename) | Simple and readable. A crash cannot leave a half-written file. |
| Concurrency | A file lock on `~/.quiver/.lock` for every write | Several MCP server processes (one per assistant session) and the CLI can run at the same time. |
| Home directory | `~/.quiver`, overridable with `QUIVER_HOME` | The override lets tests run in an isolated temp directory. |
| Delivery | MCP only | Skills are never copied or linked into a project. |
| Project identity | A `.quiver.json` with only an `id` | Survives moves and renames, and works without git. |
| Library sync | `quiver sync init` makes `~/.quiver` one git repository; `local/` becomes part of it, not a submodule, and its old history is kept as a parent of the first commit | A sync is one commit and one push to one remote. A submodule would need a second remote for `local/`, two pushes per sync, and a submodule commit after every skill edit. |
| Search by meaning | The `local` engine runs the embedding model inside the Quiver process, in plain Go (no cgo), from a model file in `~/.quiver/cache/models/`; Quiver installs and starts no runtime and no background process | One binary and one download, nothing to keep running. A query takes tens of milliseconds and the weight file is mapped, so each process loads it itself and the operating system shares the pages; a central process would be a service to start, stop, upgrade and secure. A server on a loopback address is a later second route (backlog B20). |
| Embedding dependency | `golang.org/x/text/unicode/norm` joins the module (the BERT tokenizer strips accents, which needs Unicode decomposition); nothing else is added for the `local` engine: the model code, the weight file reader and the index are Quiver's own | The model needs no library. `x/text` is a maintained part of the Go project, and the spike's own `go.mod` already used it. |
| Format versions | Every file Quiver writes in the Quiver home carries a format version from the release that introduces it; a file from a newer Quiver is refused with the version needed. A file under `cache/` is derived data: another format version is ignored and rebuilt, never refused. Each release has a golden home under `testdata/formats/<version>/`, made by running that release's binary through `testdata/formats/make-home.sh`, and a test upgrades it with the current code | An older or newer Quiver on a synced home must never read a file it does not understand and write it back damaged. A golden home made by the real release is the only proof that the next release still reads what that one wrote. |
