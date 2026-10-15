# TASK21 — Behaviour check against the real server

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** Todo

**Depends on:** TASK18, TASK19

The B01 spike (`spike/b01`, results in `docs/spike-b01-results.md`) answered the design questions with a throwaway server. Repeat its checks against the real `quiver mcp` before calling the first milestone done.

1. Point `spike/b01/run.sh` and `run2.sh` at `quiver mcp` with fixture skills, or port them to a `make spike` target.
2. Expect the spike numbers: fitting prompts load the skill, non-fitting prompts load nothing, `get_skill_file` is read when `SKILL.md` refers to a file, and the always-on skill holds across five turns.
3. Check what the spike did not cover: context compaction, and a real caveman session with the caveman plugin disabled.
4. Add the results to `docs/spike-b01-results.md`.

**Done when:** The real server matches the spike results, and compaction behaviour is recorded.
