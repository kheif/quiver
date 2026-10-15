# Technical Decisions

These apply to every task. Change them here first if you change them at all.

| Topic | Decision | Reason |
|---|---|---|
| Language | Go 1.23+ | Ships as a single binary with no runtime. The CLI and the MCP server are the same binary. |
| CLI framework | `spf13/cobra` | The standard choice for Go CLIs. Provides subcommands, flags and shell completion. |
| MCP | Official Go SDK `github.com/modelcontextprotocol/go-sdk`, stdio transport | Maintained alongside the MCP spec. Stdio is what Claude Code, Cursor and Claude Desktop launch by default. |
| TUI | Bubble Tea (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`) | The most widely used Go TUI stack. Its models can be tested without a real terminal. |
| Frontmatter parsing | `gopkg.in/yaml.v3` | `SKILL.md` frontmatter is YAML. |
| Git | Run the system `git` binary through `os/exec` | Reuses the user's credentials (SSH keys, `gh` auth) with no extra setup. |
| Storage | JSON files, written atomically (write to a temp file, then rename) | Simple and readable. A crash cannot leave a half-written file. |
| Concurrency | A file lock on `~/.quiver/.lock` for every write | Several MCP server processes (one per assistant session) and the CLI can run at the same time. |
| Home directory | `~/.quiver`, overridable with `QUIVER_HOME` | The override lets tests run in an isolated temp directory. |
| Delivery | MCP only | Skills are never copied or linked into a project. |
| Project identity | A `.quiver.json` with only an `id` | Survives moves and renames, and works without git. |
