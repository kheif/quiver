# TASK52.2: Codex hooks

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK52](TASK52-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** ✅ Done

**Depends on:** TASK50

Facts seen on 2026-11-01 with Codex 0.166:
- Hooks are read from `~/.codex/hooks.json` (or `$CODEX_HOME/hooks.json`), in the same shape as Claude Code's (`{"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "...", "timeout": 10}]}]}}`).
- The binary has the events `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `PreCompact`, `PostCompact`, `SubagentStart` and `SubagentStop`, the output field `additionalContext`, a setting `additionalContextLimit`, and a managed hook folder (`managedDir`).
- Hooks need trust: `config.toml` keeps a `[hooks.state."<file>:<event>:<i>:<j>"]` table with `enabled` and a `trusted_hash` for each hook. `codex exec` has `--dangerously-bypass-hook-trust`.

Steps:
1. Use a throwaway `CODEX_HOME` with its own login (`codex login`); never copy or change `~/.codex`.
2. Run the checks of TASK52.1 with `codex exec --json` and once in the interactive CLI.
3. Trust: what does the user see for a new hook, in the CLI and in the Codex app? Can `quiver install` make a hook trusted without editing `config.toml` by hand, for example through a `codex` command? What changes when the hook's command changes (a moved binary)?
4. Record the default and maximum of `additionalContextLimit`, and the path and rules of the managed hook folder.
5. Find out whether a project can have its own `hooks.json`.

**Done when:** the results are in `docs/spike-hooks.md`.

**Notes:**
- Results in docs/spike-hooks.md (Codex 0.166.0, `spike/b01/codex-hooks-run.sh`, source tag `rust-v0.166.0`). Delivery level: hook. Always-on 10 of 10 with plain stdout and with `additionalContext` alike, fitting skills 0 of 6 and other skills 0 of 4 as without the hook, since the model never searches for Quiver's tools. With the server's pointer also active, no `get_skill` for `reply-style`.
- The hook runs when the first prompt is sent. A failing hook never stops a session; `codex exec` reports nothing, the TUI prints `Hook failed`. Codex does not check `hookEventName`. The command runs as `$SHELL -lc` (`cmd.exe /C` on Windows, where `commandWindows` replaces `command`).
- Limit: `additionalContextLimit` per hook, default 2,500 tokens counted as 10,000 bytes, no maximum, `0` turns it off. Above it the model gets the start and the end and a file path; a rule in the middle was followed in 0 of 6.
- Sources as in Claude Code, plus `fork`. Compaction keeps no hook text, so `compact` needs the hook. The matcher `startup|clear|compact` gives every context the text once.
- Sub-agents fire `SubagentStart` (plain stdout or `additionalContext`, 6 of 6 each). `spawn_agent` starts them with the parent's history by default, and then they have the text already; with `fork_turns` `none` only `SubagentStart` brings it.
- Trust: every new or changed hook needs it; the TUI shows "Hooks need review", `codex exec` skips the hook silently. It is a hash per hook in the user's `config.toml`, so a moved binary needs it again. No `codex` command sets it; `codex app-server` (`hooks/list`, `config/batchWrite`) did, and so did a hash computed outside Codex.
- A project can have `.codex/hooks.json`; it runs once the user trusts the folder and the hook. Managed hooks (`[hooks]` in `requirements.toml`, or `/etc/codex/config.toml` and `/etc/codex/hooks.json`) need no trust; `managed_dir` only names their source.
- Not measured: the Codex app (not installed), managed hooks, Windows, automatic compaction.
