# TASK21 — Behaviour check against the real server

**Phase:** Phase 2 — MCP Server (first usable version)

**Status:** ✅ Done

**Depends on:** TASK18, TASK19

The B01 spike (`spike/b01`, results in `docs/spike-b01-results.md`) answered the design questions with a throwaway server. Repeat its checks against the real `quiver mcp` before calling the first milestone done.

1. Point `spike/b01/run.sh` and `run2.sh` at `quiver mcp` with fixture skills, or port them to a `make spike` target.
2. Expect the spike numbers: fitting prompts load the skill, non-fitting prompts load nothing, `get_skill_file` is read when `SKILL.md` refers to a file, and the always-on skill holds across five turns.
3. Check what the spike did not cover: context compaction, and a real caveman session with the caveman plugin disabled.
4. Add the results to `docs/spike-b01-results.md`.

**Notes:**
- `make spike` runs `spike/b01/real-all.sh`: the ten prompts, a five-turn session and a compacted session, each in a fresh `QUIVER_HOME` with the fixture skills in `local/`. `run-real.sh` runs one session. Every run is a real `claude -p` session and costs tokens.
- Results on 2026-10-27 match the spike: 6 of 6, 0 of 4, `get_skill_file` 2 of 2, always-on 10 of 10 and 5 of 5 turns. A second full pass gave the same numbers.
- Always-on survives `/compact` (4 of 4) because the compact summary restates the skill's rules. `get_skill` is not called again. A summary that drops the rules would lose the skill; the server cannot detect that.
- Caveman from the real source (`caveman/caveman`, always-on, user plugins off) loaded in turn one and held after compaction. Replies were about 15% shorter than the same session with caveman assigned but not always-on.
- `get_skill` ends with the skill's absolute folder, so the model may try `Read` before `get_skill_file`. In the headless runs `Read` was not allowed and it fell back to `get_skill_file`.

**Done when:** The real server matches the spike results, and compaction behaviour is recorded.
