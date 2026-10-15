# TASK10 — Source URL parsing

**Phase:** Phase 1 — Remote Sources

**Depends on:** —

`source.Parse(input)` returns a clone URL and a suggested name:

| Input | URL | Default name |
|---|---|---|
| `anthropics/skills` | `https://github.com/anthropics/skills.git` | `anthropics` |
| `github.com/a/b` | `https://github.com/a/b.git` | `a` |
| `https://gitlab.com/a/b` | `https://gitlab.com/a/b.git` | `a` |
| `git@github.com:a/b.git` | unchanged | `a` |
| `/path/to/repo` | unchanged (local path) | folder name |

**Done when:** Table tests cover every row and invalid input.
