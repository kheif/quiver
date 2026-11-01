# TASK54: Spike: session-start hooks per assistant

**Phase:** Phase 6: Session-start delivery

**Status:** Todo

**Depends on:** TASK50

Always-on skills reach the model through the MCP server's `instructions` (TASK19). Only some assistants put those into the prompt: Claude Code and Cursor do; Codex loads them only after the model searches for Quiver's tools, Antigravity writes them to a file the model rarely reads, and Claude Desktop drops them (docs/spike-mcp-clients.md). In those three, always-on skills applied in 0 of 10 prompts.

A session-start hook solves this where an assistant has one: the assistant runs a command when a session starts and adds its output to the model's context, so the skill text is there without the model deciding to load it. This spike finds out, per assistant, whether such a hook exists and how it behaves. TASK55 and TASK57 are built on its results.

**Delivery levels.** Each assistant gets one:
- **hook:** the always-on skills' text is added at session start (TASK55).
- **instructions:** the model gets Quiver's `instructions` up front and loads the always-on skills itself (Cursor, 10 of 10 in TASK37).
- **rules file:** a pointer in a file the assistant always loads makes the model load Quiver (TASK59).
- **on request:** skills load only when the prompt asks for them (Claude Desktop).

**How to run.** Use the B01 fixture skills and prompts: `spike/b01/gui-setup.sh` builds a throwaway Quiver home whose always-on skill `reply-style` ends every reply with `[always-on]`, and the fitting skills end their answer with `[used:<id>]`; `spike/b01/prompts.txt` has 6 fitting and 4 other prompts. Put hooks into a throwaway workspace's project config where the assistant reads one, or into a throwaway config folder; never change the user's own config. The existing runners (`codex-run.sh`, `cursor-run.sh`, `antigravity-run.sh`, `run-real.sh`) show how each assistant is run headless.

**Output.** `docs/spike-hooks.md`, in the style of docs/spike-mcp-clients.md: one section per assistant with the version, the setup, the counts (always-on in n of 10, fitting skills in n of 6, other skills loaded in n of 4), and one summary table with a row per assistant and these columns: hook event, user and managed config file, input on stdin, output format, output reaches the model, size limit, after compaction, sub-agents, trust or approval prompt, command form on Windows, global rules file, delivery level.

**Done when:** every subtask is done and `docs/spike-hooks.md` gives each assistant a delivery level.

**Subtasks:**
- [TASK54.1](TASK54.1-claude-code-hooks.md): Claude Code hooks
- [TASK54.2](TASK54.2-codex-hooks.md): Codex hooks
- [TASK54.3](TASK54.3-vs-code-copilot.md): VS Code Copilot
- [TASK54.4](TASK54.4-cursor-antigravity-and-global-rules-files.md): Cursor, Antigravity and global rules files
