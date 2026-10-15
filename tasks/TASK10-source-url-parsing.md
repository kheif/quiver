# TASK10 — Source URL parsing

**Phase:** Phase 1 — Remote Sources

**Status:** ✅ Done

**Depends on:** —

`source.Parse(input)` returns a clone URL and a suggested name:

| Input | URL | Default name |
|---|---|---|
| `anthropics/skills` | `https://github.com/anthropics/skills.git` | `anthropics` |
| `github.com/a/b` | `https://github.com/a/b.git` | `a` |
| `https://gitlab.com/a/b` | `https://gitlab.com/a/b.git` | `a` |
| `git@github.com:a/b.git` | unchanged | `a` |
| `/path/to/repo` | unchanged (local path) | folder name |

- Local paths start with `/`, `./`, `../` or `~/` and are made absolute.
- URL schemes: `https`, `http`, `ssh`, `git`, `file`. HTTP(S) URLs get `.git` appended.
- The default name is lowercased and cleaned to match `config.ValidateSourceName`.
- Input that starts with `-` or uses a git transport helper (`ext::`, `fd::`) is rejected. Both can run commands through git (see TASK09).

**Notes:**
- `Parse` keeps credentials in a URL such as `https://user:token@host/a/b`. `source add` must refuse or strip them, or the token ends up in `config.json` as plain text.

**Done when:** Table tests cover every row and invalid input.
