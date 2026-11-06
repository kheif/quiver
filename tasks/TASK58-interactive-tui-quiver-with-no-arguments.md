# TASK58: Interactive panel (`quiver` with no arguments)

**Phase:** Phase 7: TUI and slash commands (v1.5)

**Status:** 🚧 In progress

**Depends on:** TASK45, TASK14

Running `quiver` with no arguments in a terminal opens a management panel. It has sections, in the style of k9s or lazygit, not a search box with a list: a person moves between sections, searches the list of the current one, and acts on the selected row. Its purpose is to make the relations between skills, projects and the global scope visible, and to change them safely from the keyboard.

Build it with Bubble Tea v2 (`charm.land/bubbletea/v2`, `charm.land/bubbles/v2` and `charm.land/lipgloss/v2`). Version 1 of Bubble Tea queries the terminal for its background color when the package loads, which would slow down every quiver command, not only the panel.

**Sections:**

| Section | Shows | Subtask |
|---|---|---|
| Skills | the skills in four views: available here, project assignments, global assignments, all skills; a detail pane with the two assignments of the selected skill | TASK58.1 |
| Projects | the registered projects, their folders and skills; manage, rename, forget, split | TASK58.2 |
| Sources | the remote sources, how they are used, and their update state | TASK58.3 |
| Health | the checks of `quiver doctor`, how always-on skills reach each assistant, and the setup of Quiver in them | TASK58.5 |

The panel opens on Skills, in the view *Available here*. There is no separate overview page and no section that only holds counters.

**Layout:** a header line (`quiver ›`, the project being managed and its folder), the sections as tabs with a rule that is thicker under the current one, then a list on the left and the detail of the selected item on the right, and at the bottom a status line and a key bar with the few keys that work now. The left pane has its title, the views, the search field and a column header; the detail pane has the name, the description, the assignments as two bordered choices (*This project*, *Every folder*) and a preview of `SKILL.md`. From 100 columns the detail sits beside the list; below that the list takes the full width, `enter` opens the detail and `esc` returns to the same row and the same search. From 30 lines the layout is spacious (blank lines, rows of two lines with the source below the name); below that it is compact. The selected row has a faint green band once the terminal has reported its background, and is only bold and green before that. No boxes around sections; thin rules separate the parts.

**Rules for the whole panel:**

1. **Keys.** `1` `2` `3` `4` choose the section. `↑` `↓` `j` `k` move; `pgup`, `pgdn`, `home` and `end` jump. `←` `→` change the view. `/` searches the current section; while typing, every key is text. `enter` opens or applies, `esc` goes back one context, `tab` moves between the list and the detail, `?` shows the keys of the current view and the version, `q` quits. Everything can be done from the keyboard.
2. **No separate logic.** Every read and every change goes through the same functions as the CLI commands (`add`, `remove`, `edit`, `fork`, `update`, `source remove`, `projects`, search). The panel only draws and sends keys.
3. **The scopes are independent.** The detail of a skill shows two settings, *Project · name* and *Global*, each *Not assigned*, *On demand* or *Always on*. A change affects only its own scope. A view never decides what a change applies to.
4. **Visible target.** The header names the project that changes apply to (`managing project api`, with the folder the terminal was opened in). Choosing a project on the Projects section changes this target for the panel only; it changes neither the working directory nor the registry. When the terminal was opened elsewhere, the header says where.
5. **Safe by default.** A change for every folder, a removal and a forget ask first and say what they do. A repeated key never starts the same operation twice. Every change reports what it did, on its own line, until the next change, and says when it applies from the next session.
6. **Honest state.** A source is never reported as up to date before it was checked. A project without a folder on this machine cannot be changed. A failure shows its reason first.
7. **Honest about the terminal.** Nothing the skills contain can alter the screen. The panel works without color, in small terminals (40×12) and over `TERM=dumb`. One green accent; the selection and the focus are also marked with characters.
8. When stdout is not a terminal, `quiver` prints the overview and help as before. Without a Quiver home, the panel opens on a first-run screen that offers to set Quiver up.
9. **Nothing needs the command line for the work the panel is for.** Setting up, checking, assigning, creating, importing, editing, forking, pinning, deleting, adding, updating and removing sources, and managing projects all have a key. What stays in the CLI: `quiver sync` (needs a remote and answers about incoming changes), `import --deep/--keep/--name`, `show --json` and the other `--json` outputs, and shell completion.

There is no Settings section: Quiver has no setting that a person changes (`config.json` holds sources and pins), so the version and the library path are shown in the help.

**Done when:**
- Every subtask is done.
- Tests drive the model with key messages (no real terminal) for every key of every section, including the scope changes and what the search does with keys.
- A manual check on macOS, Linux and Windows terminals shows no layout glitches.
