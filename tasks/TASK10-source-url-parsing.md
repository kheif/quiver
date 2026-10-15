# TASK10 — Source URL parsing

**Phase:** Phase 1 — Remote Sources

**Status:** Todo

**Depends on:** —

`source.Parse(input)` returns a clone URL and a suggested name:

| Input | URL | Default name |
|---|---|---|
| `anthropics/skills` | `https://github.com/anthropics/skills.git` | `anthropics` |
| `github.com/a/b` | `https://github.com/a/b.git` | `a` |
| `https://gitlab.com/a/b` | `https://gitlab.com/a/b.git` | `a` |
| `git@github.com:a/b.git` | unchanged | `a` |
| `/path/to/repo` | unchanged (local path) | folder name |

**Notes:**
- Reject input that starts with `-` and git transport prefixes such as `ext::`. They can run commands through git (see TASK09).

**Done when:** Table tests cover every row and invalid input.
