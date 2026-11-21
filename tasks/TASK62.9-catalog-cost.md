# TASK62.9: Catalog cost

**Phase:** Phase 7b: Skill discovery (v1.6)

**Part of:** [TASK62](TASK62-finding-unassigned-skills.md): Finding unassigned skills

**Status:** Todo

**Depends on:** TASK58.1, TASK62.4, TASK62.5

Make the price of an assignment visible, so a skill that is rarely needed can move to *Searchable* before the session-start text grows.

1. **One builder.** The line builder of `internal/server/alwayson.go` (`textCache.get` makes the `- id: description` line, but also builds the skill's full text, which reads its files) is split, so that a function takes a set of skills and returns the characters of their catalog lines without reading a file. `BuildSessionContext`, which returns the text already split into parts for an assistant (`internal/cli/hookparts.go` and its per-assistant limits) and sees only the served skills, uses it for the catalog, and so does everything that reports a size, so the numbers cannot drift from what a session receives. The size of the always-on text is reported separately, before the split. The token figure is the characters divided by four and is labelled an estimate.
2. **`quiver status`.** Under the served skills, one line: how many skills are listed in this folder, the size of the listing in characters and as estimated tokens, the always-on skills counted separately, and how many skills are Searchable (they add nothing to the listing). `quiver status --json` carries the same numbers.
3. **Largest entries.** `quiver status -v` lists the ten listed skills with the longest description, each with the command that moves it to *Searchable* (`quiver add <id> --searchable`, with `--global` for a global one). `status` reads the `-v` flag for this; it ignores it today.
4. **Panel.** The Skills section shows the same count in its header, for the project the panel manages (which may be another folder than the one `quiver status` is run in), and the detail of a skill says that *Searchable* skills cost nothing in the listing and are found by search.
5. **No limit.** The numbers inform; nothing here refuses an assignment or trims a listing.
6. **Hint after the upgrade.** `search_skills` finds only Searchable skills (TASK62.5), where v1.5 searched the whole library. While the library holds skills that are at no level in this folder and the folder has no Searchable skill at all, `quiver status` adds one line: how many skills of the library the assistant's search does not find, and `quiver add --source <name> --searchable --global` (or `quiver add <id> --searchable`) to change that, naming the source with the most of them. The line goes away when the first Searchable skill exists. It is text only, not in `--json`, and refuses nothing.

**Done when:** tests cover an empty folder, a folder with global and project assignments, always-on skills counted apart, the figure equal to the length of the lines of the same skill set, the `--json` fields, `-v` with fewer and with more than ten skills, the panel header for a project in another folder, the panel detail text for a *Searchable* skill, and the hint (shown for a library with skills at no level and no Searchable skill, gone after the first Searchable skill, absent from `--json`, naming the largest source). An end-to-end test (TASK30) runs `status` and `status -v` through the binary.
