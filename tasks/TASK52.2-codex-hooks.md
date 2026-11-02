# TASK52.2: Codex hooks

**Phase:** Phase 6: Session-start delivery (v1.4)

**Part of:** [TASK52](TASK52-spike-session-start-hooks-per-assistant.md): Spike: session-start hooks per assistant

**Status:** Todo

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
