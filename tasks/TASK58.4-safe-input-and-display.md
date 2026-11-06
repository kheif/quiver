# TASK58.4: Safe input and display

**Phase:** Phase 7: TUI and slash commands (v1.5)

**Part of:** [TASK58](TASK58-interactive-tui-quiver-with-no-arguments.md): Interactive panel (`quiver` with no arguments)

**Status:** ✅ Done

**Depends on:** TASK58.1

What makes the panel safe to use with skills it did not write, and honest about what happened.

1. **Clean text.** Everything that comes from a skill (ID, name, description, instructions) is cleaned before it is drawn: newlines, carriage returns, tabs and other control characters, and characters that reorder text, are replaced or removed. A skill cannot move the cursor, recolor the screen or break the frame.
2. **Confirmation.** A change for every folder, a removal of a source and a forget ask first and say what they remove, including an always-on mark. Nothing that changes every folder runs from a single key without it.
3. **Status line.** One line of its own with the result of the last change (`✓` for success, `✗` for failure), kept until the next change. A long message wraps over up to three lines instead of being cut. For an error the reason comes first and the path last.
4. **Failures keep the state visible.** A failed edit, a corrupt registry or a missing folder shows its message and the cause; the panel never shows an empty library as if nothing were wrong. A folder that disappeared while the panel was open is never created again by a key.
5. **Input.** Pasted text goes into the search on one line. While typing a search, every key is text. `ctrl+c` always quits. An unknown key never does anything but say so.
6. **Display.** Works with `NO_COLOR` and `TERM=dumb` (the selected row, the current section and the focused pane are marked with characters, not only attributes). The terminal is restored on every way out, including after the editor.
7. The key line shows only the keys that work in the current context, each named by what it does, and keeps `? help` and `q quit` when the terminal is narrow.

**Done when:** tests cover a skill whose text has newlines and control characters, the confirmation, long and failing messages, paste, small sizes, and the missing-folder case.
