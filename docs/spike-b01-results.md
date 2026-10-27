# Spike B01: Assistant Behaviour Over MCP

Measured on 2026-10-15 with Claude Code 2.1.300 and a throwaway Go server (`spike/b01`, MCP Go SDK v1.8.0). The tests ran headless with `claude -p`, in empty project folders, with no user plugins (`--setting-sources project --strict-mcp-config`).

## Setup

- Three fixture skills: `sql-migration-review` (refers to `references/checklist.md`), `release-notes` and `k8s-debug`. Each one asks the model to end its answer with a marker such as `[used:k8s-debug]`, so use is visible in the output.
- Prompts: 6 where one skill clearly fits, 4 where none fits (`spike/b01/prompts.txt`).
- The server logs every tool call (`QV_LOG`).
- Scripts: `run.sh` and `run2.sh` run the sessions; `analyze.py` summarizes them.

## Results

### 1. MCP tool descriptions are not seen

Claude Code defers MCP tools behind its `ToolSearch` tool. The model sees only tool names until it loads a tool, so a skill list inside the `list_skills` description is invisible.

| Where the skill list is | Fitting prompts that loaded the skill | Non-fitting prompts that loaded a skill |
|---|---|---|
| `list_skills` tool description | 0 / 6 | 0 / 4 |
| Server `instructions` | **6 / 6** | **0 / 4** |
| Tool description, tool search turned off (`ENABLE_TOOL_SEARCH=false`) | 6 / 6 | 0 / 4 |

Quiver cannot turn tool search off for the user, so **the skill catalog goes into `instructions`**.

### 2. `get_skill_file` is used

When `SKILL.md` said "read `references/checklist.md`", the model called `get_skill_file` in 4 of 4 runs.

### 3. `instructions` are cut at about 4096 characters

Longer `instructions` are cut and end with `… [truncated]`. Everything after the cut is lost. Measured: 4077 characters arrived intact; at 4600 the cut fell right after character ~4096.

caveman's `SKILL.md` is 4678 bytes, so **the full text of an always-on skill does not fit**.

### 4. Always-on works as a pointer

`instructions` contained only: "Always-on skills: reply-style. Before your first reply in this conversation, call the get_skill tool of this server for each always-on skill and follow it for the whole conversation."

- Single-turn: the model loaded the skill and followed it in **10 / 10** runs, including the four non-fitting prompts.
- Five-turn session (`--resume`): followed in **5 / 5** turns, with one `get_skill` call in total.
- Not tested: context compaction.

### 5. Subagents are unreliable

With always-on text in `instructions`, one of two `general-purpose` subagents followed it. Treat always-on skills as main-session only.

### 6. Project resolution: use the working directory

- The server's working directory was the project folder in every run.
- `roots/list` cannot be used. Under protocol version 2026-07-28 the SDK refuses it while serving a request: `"roots/list" cannot be sent while serving a request on protocol version 2026-07-28: return an InputRequests map instead (multi round-trip requests, SEP-2322)`.
- The SDK's `InitializedHandler` never ran, so roots cannot be requested at startup either.

## Design changes

- **Skill catalog:** moved from the `list_skills` description to `instructions` (TASK18). The catalog is fixed for the session.
- **Budget:** `instructions` stay under 3800 characters. Descriptions are shortened, then dropped, then the list ends with `…and N more: call list_skills` (TASK18).
- **Always-on:** a pointer in `instructions` plus `get_skill`, instead of the full skill text (TASK19).
- **Project resolution:** `QUIVER_PROJECT`, then the working directory, walking up to `.quiver.json`. No MCP roots (TASK17).
- **Live updates:** only `prompts/list_changed`; new skills reach the catalog in the next session (TASK26).

## Recheck against the real server (TASK21)

Measured on 2026-10-27 with Claude Code 2.1.308 and the real `quiver mcp`. Same fixtures and prompts, each session in a fresh `QUIVER_HOME` with the fixtures in `local/` and `reply-style` assigned with `--always`. Run it with `make spike` (`spike/b01/real-all.sh`, one session: `run-real.sh`).

| Check | Spike | Real server |
|---|---|---|
| Fitting prompts that loaded the skill | 6 / 6 | **6 / 6** |
| Non-fitting prompts that loaded a skill | 0 / 4 | **0 / 4** |
| `get_skill_file` when `SKILL.md` refers to a file | 4 / 4 | **2 / 2** |
| Always-on followed, single turn | 10 / 10 | **10 / 10** |
| Always-on followed over five turns | 5 / 5 | **5 / 5**, one `get_skill` call |

`get_skill` ends with the skill's absolute folder. In 3 of 4 `sql-migration-review` runs (two passes) the model tried `Read` on that path first, was refused (headless runs allow only Quiver tools), then called `get_skill_file`. In a normal session `Read` would work too.

### Compaction

Two turns, then `/compact`, then two more turns. Always-on held after compaction in **4 / 4** sessions, with no second `get_skill` call. The compact summary restated the rules ("every reply … must end with the line `[always-on]`"). The skill survives because the summary model chose to keep it, not because Quiver re-sends it. The pointer in `instructions` is still there after compaction, so a model that lost the rules could load the skill again.

### Caveman

`caveman/caveman` from `JuliusBrussee/caveman`, assigned with `--always`, user plugins off (`--setting-sources project`). Five turns with `/compact` after turn two, against the same session with caveman assigned but not always-on:

| Turn | Always-on (words) | Not always-on (words) |
|---|---|---|
| 1 | 373 | 455 |
| 2 | 396 | 495 |
| 3 (after compact) | 340 | 423 |
| 4 | 479 | 501 |

The skill loaded before the first reply and the summary kept it ("the caveman skill (always-on, from the quiver MCP server) stays active"). Replies were about 15% shorter in total (4–20% per turn), with the answer first.

## Still open

- Other assistants (Cursor, Claude Desktop) may not defer tools or may handle `instructions` differently (TASK39).
