# Spike: session-start hooks per assistant

Always-on skills reach the model through the MCP server's `instructions`, and only some assistants put those into the prompt (see [spike-mcp-clients.md](spike-mcp-clients.md)). A session-start hook runs a command when a session starts and adds its output to the model's context, so the skill text is there without the model deciding to load it. This spike checks, per assistant, whether such a hook exists and how it behaves.

**Status:** Claude Code and Codex done on 2026-11-02. VS Code Copilot, Cursor and Antigravity are not measured yet.

**Delivery levels.** Each assistant gets one:

- **hook:** the always-on skills' text is added at session start.
- **instructions:** the model gets Quiver's `instructions` up front and loads the always-on skills itself.
- **rules file:** a pointer in a file the assistant always loads makes the model load Quiver.
- **on request:** skills load only when the prompt asks for them.

## Summary

| Assistant | Hook event | User and managed config file | Input on stdin | Output format | Output reaches the model | Size limit | After compaction | Sub-agents | Trust or approval prompt | Command form on Windows | Global rules file | Delivery level |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Claude Code 2.1.308 | `SessionStart`, matcher on `source` | `~/.claude/settings.json` or `$CLAUDE_CONFIG_DIR/settings.json`; `managed-settings.json` in a system folder | JSON with `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `source`; interactive sessions add `model` and `scratchpad_dir` | Plain stdout or `additionalContext` | Yes, 10 of 10 | 10,000 characters per hook command; above that, a file path and a 2 KB preview | Fires again with `source` `compact` | `SessionStart` does not reach them; `SubagentStart` does | The folder's trust dialog, in interactive sessions only; no approval per hook | Documented: Git Bash, or PowerShell without Git Bash; `shell` chooses | Documented: `~/.claude/CLAUDE.md` | **hook** |
| Codex 0.166.0 | `SessionStart`, matcher on `source`; runs when the first prompt is sent | `$CODEX_HOME/hooks.json` (`~/.codex/hooks.json`) or `[hooks]` in `config.toml`; managed: `[hooks]` in `requirements.toml`, or `/etc/codex/config.toml` and `/etc/codex/hooks.json` | JSON with `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `model`, `permission_mode`, `source` | Plain stdout or `additionalContext` | Yes, 10 of 10 | `additionalContextLimit` per hook command, default 2,500 tokens counted as 10,000 bytes; above that, the start and end and a file path; `0` turns it off | Fires again with `source` `compact`; compaction keeps no hook text | `SubagentStart` does; a sub-agent that starts with its parent's history, the default, has the text already | Every new or changed hook: "Hooks need review" in the TUI; `codex exec` skips it without a message | From source: `cmd.exe /C`, and `commandWindows` replaces `command` | From source: `AGENTS.override.md` or `AGENTS.md` in `$CODEX_HOME` | **hook** |

## Claude Code

Measured on 2026-11-02 with Claude Code 2.1.308 on macOS. The run script is `spike/b01/claude-hooks-run.sh`, and `spike/b01/claude-hooks-analyze.py` summarizes its output:

```sh
QUIVER=$PWD/bin/quiver spike/b01/claude-hooks-run.sh /tmp/quiver-hooks-spike
python3 spike/b01/claude-hooks-analyze.py /tmp/quiver-hooks-spike
```

Each run has its own throwaway folder with the hook in that folder's `.claude/settings.json`, and reads no other settings (`--setting-sources project`). The hook was not put into `~/.claude/settings.json`, so the user file itself was not exercised. The documentation says it is read and merged the same way. The prompts, sources and sub-agents ran on the default model (`claude-opus-5-5`), the size and failure checks on Haiku. Every check ran in two full passes; the counts below add them up, except the prompt counts, which were the same in each pass.

### The 10 prompts

The hook prints the fixture skill `reply-style` in full, and every prompt runs in a new session against the real `quiver mcp`. Without the pointer, `reply-style` is not assigned in Quiver, so the hook is its only way in.

| Hook | Always-on | Fitting skills | Other skills loaded | `get_skill` for `reply-style` |
|---|---|---|---|---|
| Plain stdout | 10 of 10 | 6 of 6 | 0 of 4 | 0 of 10 |
| `additionalContext` | 10 of 10 | 6 of 6 | 0 of 4 | 0 of 10 |
| Plain stdout, and `reply-style` also assigned as always-on | 10 of 10 | 6 of 6 | 0 of 4 | 6 of 10 |

The last row shows what happens when the hook and the server's always-on pointer are both active: the model still loads the skill it already has in 6 of 10 sessions.

The two output forms behave the same. In the transcript, plain stdout is a `hook_success` record and `additionalContext` a `hook_additional_context` record. The model sees a label in front of the first line (for plain stdout, `SessionStart:startup hook success:`), so the text's first line does not begin a line.

### Size

The limit is 10,000 characters, as documented, and it is the same for both forms. Up to the limit the whole text arrives. Above it, the model gets a path to a saved file and a preview of the first 2 KB, and the end of the text does not reach it.

| Characters | Plain stdout | `additionalContext` |
|---|---|---|
| 5,000 | Whole | Whole |
| 9,900 | Whole | Whole |
| 10,100 | Preview | Preview |
| 20,000 | Preview | Preview |
| 50,000 | Preview | Preview |

- The limit counts characters, not bytes: 7,000 characters of `ş` (13,882 bytes) arrived whole.
- The limit is per hook command: three hooks of 9,000 characters each, 27,000 in all, arrived whole.

Over the limit, whether a rule applies depends on where it is. These runs had a 20 KB text and Claude Code's normal tools:

| Where the rule is | Rule followed | Model opened the saved file |
|---|---|---|
| In the preview | 6 of 6 | 6 of 6 |
| After the preview | 0 of 6 | 1 of 6, and it still did not follow the rule |

### Sources

The hook runs with `source` set to how the session started. A `matcher` on the hook selects the sources it runs for; without one it runs for all of them. `fork` is documented too and was not measured.

| Event | `source` | Result |
|---|---|---|
| New session | `startup` | Text is there. |
| `/clear` (interactive) | `clear` | With `clear` in the matcher, the text is there in 4 of 4 sessions. With the matcher `startup`, the hook does not run and the rule is gone in 2 of 2. |
| `/compact` | `compact` | The hook runs again and adds the text to the new context. The compaction summary also carried the rule in 16 of 16 sessions that had it before `/compact`, and with the matcher `startup` the rule held through the summary alone in 6 of 6. The model writes that summary, so it is not a guarantee. |
| `--resume` | `resume` | The earlier text is still in the session: the rule held in 6 of 6 with the matcher `startup`. A hook that also runs on `resume` adds a second copy; in 6 of 6 the model quoted the text of both firings. |

Automatic compaction was not triggered; only `/compact` was run.

The matcher `startup|clear|compact` gives every context the text once. It held in all three steps of the interactive session (new session, `/clear`, `/compact`), 2 of 2.

### Sub-agents

| Hooks | Main session followed the rule | Sub-agent followed the rule |
|---|---|---|
| `SessionStart` only | 6 of 6 | 0 of 6 |
| `SessionStart` and `SubagentStart` with `additionalContext` | 6 of 6 | 6 of 6 |

`SubagentStart` gets the common input plus `agent_id` and `agent_type`. It can add context but cannot block the sub-agent.

### Time and failures

The default timeout of a command hook is 600 seconds, and the hook's `timeout` field sets it in seconds. A slow hook delays the first reply by its whole run time:

| Hook | Seconds until the session ended (two passes) |
|---|---|
| None | 4, 5 |
| `sleep 5` | 9, 10 |
| `sleep 20` | 23, 26 |

Those runs are headless. The documentation says an interactive session lets you type right away, while Claude's first reply still waits for the hooks.

A failing hook never stopped a session. In every case below the session went on and the model did not get the text, even when the hook had printed it before failing.

| Hook | Outcome in the stream |
|---|---|
| Prints the text, exits 1 | `error`, exit code 1 |
| Prints the text, exits 2 | `error`, exit code 2 |
| Prints the text, runs past `timeout` | `cancelled` |
| Binary does not exist | `error`, exit code 127 |
| `additionalContext` with the wrong `hookEventName` | `error`, exit code 1 |
| Binary path with a space, not quoted | `error`, exit code 127 |

A path with a space works when quoted in the command, and in exec form (`command` plus `args`, which uses no shell). What the terminal shows for a failed hook was not checked.

### Trust

In a new folder, the hook had not run 6 seconds after an interactive start, while the trust dialog was still open, in 9 of 9 starts; once the dialog was accepted, it ran. In a folder accepted earlier, the hook ran within 6 seconds with no dialog, in 3 of 3. The documentation says this holds for every settings file, including the user's own, and that `-p` and SDK sessions never show the dialog and treat the folder as trusted. There is no separate approval per hook.

### Managed settings

From the documentation; not measured, since writing there needs administrator rights and changes the whole machine.

- **File:** `managed-settings.json`, plus an optional `managed-settings.d/` folder, in `/Library/Application Support/ClaudeCode/` on macOS, `/etc/claude-code/` on Linux and WSL, and `C:\Program Files\ClaudeCode\` on Windows. MDM can deliver the same content through the macOS `com.anthropic.claudecode` domain or the Windows `HKLM\SOFTWARE\Policies\ClaudeCode` key, and the claude.ai console through server-managed settings.
- **Next to the user's hooks:** hook entries merge across levels, so "user, project, and local settings add their own hooks without removing managed ones".
- **Turning them off:** `disableAllHooks` in user, project or local settings does not disable managed hooks; only the same key at the managed level does. An administrator on the machine can edit the managed source itself.
- **The other way round:** with `allowManagedHooksOnly` in managed settings, user, project, local and plugin hooks are blocked, so a hook in the user's settings file does not run.

### Windows

From the documentation; not measured. A hook in shell form (`command` only) runs in Git Bash, or in PowerShell when Git Bash is not installed; the `shell` field chooses. Exec form needs `command` to be a real executable such as a `.exe`, so the `.cmd` shim npm installs cannot run that way.

## Codex

Measured on 2026-11-02 with `codex-cli` 0.166.0 on macOS and the default model (`gpt-6.1-sol`); what the runs could not show was read from Codex's source at the tag `rust-v0.166.0`. The run script is `spike/b01/codex-hooks-run.sh`, and `spike/b01/codex-hooks-analyze.py` summarizes its output:

```sh
CODEX_HOME=/tmp/codex-spike-home codex login
CODEX_HOME=/tmp/codex-spike-home PYTHON=<a python with pyte> QUIVER=$PWD/bin/quiver spike/b01/codex-hooks-run.sh /tmp/quiver-codex-hooks
CODEX_HOME=/tmp/codex-spike-home python3 spike/b01/codex-hooks-analyze.py /tmp/quiver-codex-hooks
```

The runs use a throwaway `CODEX_HOME` with its own login, so `~/.codex` is never read or changed. The script writes one set of hooks to that home's `hooks.json`. Each command is `slot.sh <slot>`, which runs a script from the session's own folder, so the set is trusted once and serves every session. `spike/b01/codex-hooks-tui.py` drives the interactive CLI in a pseudo-terminal and needs `pyte`. The size and failure checks ran with `model_reasoning_effort="low"`. Every check ran in two full passes; the counts below add them up, except the prompt counts, which were the same in each pass.

Asked to quote its context, the model often refused ("I can’t quote hidden developer instructions"). The size, source and failure results therefore come from the session's rollout file under `$CODEX_HOME/sessions`, which records what reached the model.

### The 10 prompts

The hook prints the fixture skill `reply-style` in full, and every prompt runs in a new `codex exec` session against the real `quiver mcp`.

| Hook | Always-on | Fitting skills | Other skills loaded | `get_skill` for `reply-style` |
|---|---|---|---|---|
| Plain stdout | 10 of 10 | 0 of 6 | 0 of 4 | 0 of 10 |
| `additionalContext` | 10 of 10 | 0 of 6 | 0 of 4 | 0 of 10 |
| Plain stdout, and `reply-style` also assigned as always-on | 10 of 10 | 0 of 6 | 0 of 4 | 0 of 10 |

Without the hook, always-on applied in 0 of 10 ([spike-mcp-clients.md](spike-mcp-clients.md)). Fitting skills stay at 0 of 6, as there: Codex defers MCP tools, and the model never searched for Quiver's. With the server's pointer also active, the model did not load `reply-style` a second time.

Both forms reach the model as a developer message that holds the text as it is, with no label in front.

### Size

A hook entry's `additionalContextLimit` sets the limit in tokens, and Codex counts 4 bytes as a token, so the default of 2,500 tokens is 10,000 bytes. `0` turns the limit off; the source has no maximum. Up to the limit the whole text arrives. Above it, the model gets a header (`Warning: truncated output (original token count: 5000)`), about the first and the last 5,000 bytes with `…2542 tokens truncated…` between them, and `Full hook output saved to:` a file under `$TMPDIR/hook_outputs/`.

| Bytes | Plain stdout | `additionalContext` |
|---|---|---|
| 5,000 | Whole | Whole |
| 9,900 | Whole | Whole |
| 10,100 | Start and end | Start and end |
| 20,000 | Start and end | Start and end |
| 50,000 | Start and end | Start and end |

- The limit counts bytes: 4,900 characters of `ş` (9,686 bytes) arrived whole, and 7,000 (13,860 bytes) were cut. Claude Code counts characters.
- With `additionalContextLimit` 0, 50,000 bytes arrived whole. With 5,000, 20,000 bytes arrived whole and 50,000 were cut to about 20,000.
- The limit is per hook command: three hooks of 9,000 bytes each, 27,000 in all, arrived whole.

Over the limit, whether a rule applies depends on where it is. These runs had a 20 KB text:

| Where the rule is | Rule followed | Model opened the saved file |
|---|---|---|
| At the start | 6 of 6 | 0 of 6 |
| In the middle | 0 of 6 | 0 of 6 |
| At the end | 6 of 6 | 0 of 6 |

### Sources

The hook runs when the first prompt is sent, not when the session opens: in 16 of 16 TUI sessions it had not run 5 seconds after the prompt box appeared. A `matcher` on the hook selects the sources it runs for, as in Claude Code.

| Event | `source` | Result |
|---|---|---|
| New session | `startup` | Text is there. |
| `/clear` (TUI) | `clear` | With `clear` in the matcher, the text is there in 4 of 4 sessions. With the matcher `startup`, the hook does not run and the rule is gone in 2 of 2. |
| `/compact` (TUI) | `compact` | The hook runs again and adds the text; the rule held in 6 of 6. The history that replaces the old one kept no hook text in 10 of 10 compactions, and the summary is encrypted. With the matcher `startup`, the rule held through the summary alone in 1 of 2. |
| `codex exec resume` | `resume` | The earlier text is still in the session: the rule held in 6 of 6 with the matcher `startup`. A hook that also runs on `resume` adds a second copy, 6 of 6. |
| `codex exec fork` | `fork` | The new session has its parent's text: the rule held in 6 of 6 with the matcher `startup`. A hook that also runs on `fork` adds a second copy, 6 of 6. |

The matcher `startup|clear|compact` gives every context the text once. It held in all three steps of the TUI session (new session, `/clear`, `/compact`), 2 of 2.

### Sub-agents

Codex's `spawn_agent` tool takes `fork_turns`. `all`, the default, starts the sub-agent with its parent's history; `none` starts it with only its message. A sub-agent fires `SubagentStart`, never `SessionStart`.

| Hooks | `fork_turns` | Sub-agent followed the rule |
|---|---|---|
| `SessionStart` only | Default | 6 of 6 |
| `SessionStart` only | `none` | 0 of 6 |
| `SessionStart` and `SubagentStart` with `additionalContext` | `none` | 6 of 6 |
| `SessionStart` and `SubagentStart` with plain stdout | `none` | 6 of 6 |

The main session followed the rule in 24 of 24. In a trial run with both hooks and the default `fork_turns`, the sub-agent had the text twice: from its parent's history and from `SubagentStart`. `SubagentStart` gets `session_id` (the parent's), `turn_id`, `transcript_path` (the sub-agent's), `cwd`, `hook_event_name`, `model`, `permission_mode`, `agent_id` and `agent_type`, and its matcher is on `agent_type`.

### Time and failures

The command runs as `$SHELL -lc '<command>'`, a login shell, in the session's folder. The default timeout is 600 seconds, and the hook's `timeout` field sets it in seconds. A slow hook delays the first reply by its whole run time:

| Hook | Seconds until the session ended (two passes) |
|---|---|
| None | 7, 8 |
| `sleep 5` | 13, 12 |
| `sleep 20` | 27, 27 |

A failing hook never stopped a session. In every case below but one the session went on without the text:

| Hook | Outcome |
|---|---|
| Prints the text, exits 1 | Text not added |
| Prints the text, exits 2 | Text not added |
| Prints the text, runs past a `timeout` of 2 seconds | Text not added; the sessions ended after 12 and 10 seconds |
| Output starts with `{` but is not JSON | Text not added |
| `additionalContext` with the wrong `hookEventName` | Text added: Codex does not check the name |
| Binary does not exist | Text not added |
| Binary path with a space, not quoted | Text not added |

`codex exec` said nothing about a failed hook, neither in its JSON nor on stderr. The TUI printed `Hook failed` with `hook exited with code 127` under it, once for each of the last two. A path with a space works when quoted. There is no exec form: `command` is always a shell line.

### Trust

A hook from a user or project file runs only once it is trusted. Trust is a `trusted_hash` per hook in the user's `config.toml`, under `[hooks.state."<file>:<event>:<group>:<hook>"]`, for example `"/Users/me/.codex/hooks.json:session_start:0:0"`. The hash is the SHA-256 of the hook's entry: event, matcher, command, timeout, `async`, `statusMessage` and `additionalContextLimit`. Changing any of these, by moving the binary for example, makes the hook "changed", and it does not run until it is trusted again.

- **TUI.** With a new or changed hook, a new session opens with this dialog:

  ```
  Hooks need review
  9 hooks are new or changed.
  Hooks can run outside the sandbox after you trust them.
  › 1. Review hooks
    2. Trust all and continue
    3. Continue without trusting (hooks won't run)
  ```

  "Review hooks" lists them by event, each with its source file, command, timeout and `New hook - review required`. The list names a hook by its `statusMessage`, or `Unnamed hook` without one. "Trust all and continue" wrote the hashes to `config.toml`, and the hooks ran.
- **`codex exec`** never asks: it skips a new or changed hook without a message. `--dangerously-bypass-hook-trust` runs them for that one invocation.
- **A moved binary.** After the command changed, `codex exec` skipped the hook in 2 of 2, and the TUI showed the dialog with `1 hook is new or changed.` in 2 of 2; "Continue without trusting" left the rule out in 2 of 2.
- **Trust from outside Codex.** No `codex` command trusts a hook. Two ways worked, in 2 of 2 each. One writes the hash that `spike/b01/codex-hook-trust.py` computes into `config.toml`; it matched what the TUI wrote for 18 of 18 hooks. The other asks `codex app-server`, which Codex marks experimental: `hooks/list` gives each hook's key, trust status and current hash, and `config/batchWrite` stores the hash. These are the two requests the TUI itself sends.
- **Codex app.** Not installed on this machine, so not measured. It reads the same files; what it shows for a new hook was not checked.

### Project hooks

A project can have its own `.codex/hooks.json`, or `[hooks]` in `.codex/config.toml`, in the session's folder or a folder above it up to the project root, by default the nearest folder with `.git`. Such a hook runs only when both the folder and the hook are trusted, and both answers are kept in the user's `config.toml`: `trust_level = "trusted"` under `[projects."<folder>"]`, and the hook's hash under a key that names the project's file. A project's files cannot trust their own hooks, since trust is read only from the user's `config.toml` and `-c` options.

- In a folder never trusted, `codex exec` did not run the project's hook, 2 of 2.
- In the TUI, a folder with a `.codex` folder first asked "Trust this folder? Codex can read, edit, and run files here, …", then showed "Hooks need review" with 1 hook. After both, the hook ran in the TUI and in a later `codex exec` there, 2 of 2.

### Managed hooks

From the source; not measured, since writing there needs administrator rights and changes the whole machine.

- **Files:** a `[hooks]` table in `requirements.toml`: `/etc/codex/requirements.toml` on macOS and Linux, `%ProgramData%\OpenAI\Codex\requirements.toml` on Windows, the macOS managed preference `requirements_toml_base64` of `com.openai.codex`, or the cloud settings of an enterprise account. Hooks of the system config count as managed too: `[hooks]` in `/etc/codex/config.toml`, and `/etc/codex/hooks.json` (`%ProgramData%\OpenAI\Codex\` on Windows).
- **The managed folder:** `managed_dir`, and `windows_managed_dir` on Windows, in the `[hooks]` table of `requirements.toml` name a folder for the managed hooks; Codex uses it as their source path, which it shows and builds their keys from. Commands are not resolved against it. Hooks from several requirement sources add up, but two different folders for the running platform are an error.
- **Trust:** managed hooks need none, and the user cannot turn them off. A hook from `requirements.toml` that cannot be loaded keeps sessions from starting, with `failed to load required managed hooks`.
- **The other way round:** `allow_managed_hooks_only = true` in `requirements.toml` ignores user, project and session hooks.

### Windows

From the source; not measured. A hook runs as `%COMSPEC% /C "<command>"`, `cmd.exe` by default, and a hook's `commandWindows` replaces `command` there.
