# TASK41 — Interactive TUI (`quiver` with no arguments)

**Phase:** Phase 5 — v2

**Status:** Todo

**Depends on:** TASK35, TASK14

Running `quiver` with no arguments in a terminal opens an interactive screen. Build it with Bubble Tea (`github.com/charmbracelet/bubbletea`, `bubbles` and `lipgloss`).

**Layout:**

```
┌ Quiver ─ project: app (app-k3f9q2) ──────────────────────────┐
│ Search: migr_                                                │
│                                                              │
│  [P] db-migration        Rules for database migrations       │
│  [ ] team/migrate-check  Checks a migration before merge     │
│  [G] commit-rules        Commit message conventions          │
│                                                              │
│ ┌ Preview ───────────────────────────────────────────────┐   │
│ │ SKILL.md of the selected skill                         │   │
│ └────────────────────────────────────────────────────────┘   │
│                                  ┌───────────────────────┐   │
│ space: project  g: global        │ anthropic: 3 updated  │   │
│ e: edit  u: updates  q: quit     │ press u to pull       │   │
└──────────────────────────────────┴───────────────────────┴───┘
```

1. **Search:** typing filters the whole library live, using the ranking from TASK35.
2. **Markers:** `[G]` global, `[P]` this project, `[ ]` not assigned here.
3. **Keys:**
   - `space`: toggle the skill for the current project
   - `g`: toggle global
   - `enter`: full preview
   - `e`: edit in `$EDITOR` (local skills only)
   - `/`: focus search
   - `tab`: switch between the views Skills, Projects and Sources
   - `q`: quit
4. Every change goes through the same functions as the CLI commands. The TUI contains no separate logic.
5. Outside a project, `space` asks which project to assign to.
6. When stdout is not a terminal, `quiver` prints the help text instead of opening the TUI.

**Done when:** Tests drive the model with key messages (no real terminal) for search, toggling and editing. A manual check on macOS, Linux and Windows terminals shows no layout glitches.
