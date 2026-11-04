# Spike: session-start hooks per assistant

Always-on skills reach the model through the MCP server's `instructions`, and only some assistants put those into the prompt (see [spike-mcp-clients.md](spike-mcp-clients.md)). A session-start hook runs a command when a session starts and adds its output to the model's context, so the skill text is there without the model deciding to load it. This spike checks, per assistant, whether such a hook exists and how it behaves.

**Status:** Claude Code and Codex done on 2026-11-02; VS Code Copilot, Cursor, Antigravity and the global rules files on 2026-11-03. Every assistant measured has a hook whose text reaches the model.

**Delivery levels.** Each assistant gets one:

- **hook:** the always-on skills' text is added at session start.
- **instructions:** the model gets Quiver's `instructions` up front and loads the always-on skills itself.
- **rules file:** a pointer in a file the assistant always loads makes the model load Quiver.
- **on request:** skills load only when the prompt asks for them.

## Summary

| Assistant | Hook event | User and managed config file | Input on stdin | Output format | Output reaches the model | Size limit | After compaction | Sub-agents | Trust or approval prompt | Command form on Windows | Global rules file | Delivery level |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Claude Code 2.1.308 | `SessionStart`, matcher on `source` | `~/.claude/settings.json` or `$CLAUDE_CONFIG_DIR/settings.json`; `managed-settings.json` in a system folder | JSON with `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `source`; interactive sessions add `model` and `scratchpad_dir` | Plain stdout or `additionalContext` | Yes, 10 of 10 | 10,000 characters per hook command; above that, a file path and a 2 KB preview | Fires again with `source` `compact` | `SessionStart` does not reach them; `SubagentStart` does | The folder's trust dialog, in interactive sessions only; no approval per hook | Documented: Git Bash, or PowerShell without Git Bash; `shell` chooses | Documented: `~/.claude/CLAUDE.md` | **hook** |
| Codex 0.166.0 | `SessionStart`, matcher on `source`; runs when the first prompt is sent | `$CODEX_HOME/hooks.json` (`~/.codex/hooks.json`) or `[hooks]` in `config.toml`; managed: `[hooks]` in `requirements.toml`, or `/etc/codex/config.toml` and `/etc/codex/hooks.json` | JSON with `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `model`, `permission_mode`, `source` | Plain stdout or `additionalContext` | Yes, 10 of 10 | `additionalContextLimit` per hook command, default 2,500 tokens counted as 10,000 bytes; above that, the start and end and a file path; `0` turns it off | Fires again with `source` `compact`; compaction keeps no hook text | `SubagentStart` does; a sub-agent that starts with its parent's history, the default, has the text already | Every new or changed hook: "Hooks need review" in the TUI; `codex exec` skips it without a message | From source: `cmd.exe /C`, and `commandWindows` replaces `command` | `$CODEX_HOME/AGENTS.md`, or `AGENTS.override.md` there (from source) | **hook** |
| VS Code 1.143.1, Copilot Chat 0.71.0 | `SessionStart`, when the first prompt starts a session; `source` is always `new` | `~/.copilot/hooks/*.json`, read by both harnesses; managed: `chat.hooks.allowManagedOnly`, and for the Copilot harness `/etc/github-copilot/policy.d/*.json` (documented) | JSON with `timestamp`, `hook_event_name`, `session_id`, `transcript_path`, `source`, `model`, `cwd` | JSON only: `hookSpecificOutput.additionalContext` in the Local harness, `additionalContext` at the top level in the Copilot harness; plain stdout is dropped | Yes, 10 of 10 | None up to 200,000 characters in the Local harness; 10 MiB documented for the Copilot harness | Not measured | `SubagentStart` does, `SessionStart` does not | None seen, with workspace trust off | Documented: a `windows` field (Local), `powershell` (Copilot harness) | `*.instructions.md` with `applyTo: "**"` in the profile's `prompts` folder, or `~/.copilot/instructions` | **hook** |
| Cursor CLI 2026.10.22 | `sessionStart`, when a session starts | `~/.cursor/hooks.json`; managed: `/Library/Application Support/Cursor/hooks.json`, `/etc/cursor/hooks.json`, `C:\ProgramData\Cursor\hooks.json` (documented) | JSON with `session_id`, `conversation_id`, `workspace_roots`, `model`, `user_email`, `cursor_version` and others | `{"additional_context": "..."}` | Yes, 10 of 10 | Not measured | Not measured | Documented: `subagentStart` only allows or denies | Documented: project hooks need a trusted workspace | Documented: PowerShell in the app | Not checked | **hook** in the CLI; the app was not measured |
| Antigravity CLI `agy` 1.4.12 | `SessionStart`, once per conversation | `~/.gemini/config/hooks.json`, shared by the Antigravity products; workspace `.agents/hooks.json` | JSON with `conversationId`, `workspacePaths`, `transcriptPath`, `artifactDirectoryPath`, `modelName`, `lastUserInput` | `injectSteps` with a `systemMessage`, `userMessage` or `ephemeralMessage`; plain stdout is dropped | Yes, 10 of 10 | About 48,900 bytes; the rest is dropped with a note | Not measured | Each sub-agent conversation runs `SessionStart` | None seen | Documented: `cmd /c` | `~/.gemini/config/GEMINI.md` or `AGENTS.md`, and `~/.gemini/GEMINI.md` | **hook** |

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

## VS Code Copilot

Measured on 2026-11-02 and 2026-11-03 with VS Code 1.143.1 on macOS, its built-in GitHub Copilot Chat 0.71.0, agent mode and the model `Auto`, which picked `gpt-5.6-luna`. The run script is `spike/b01/vscode-hooks-run.sh`, and `spike/b01/vscode-hooks-analyze.py` summarizes its output:

```sh
QUIVER=$PWD/bin/quiver spike/b01/vscode-hooks-run.sh /tmp/quiver-vscode-hooks
python3 spike/b01/vscode-hooks-analyze.py /tmp/quiver-vscode-hooks
```

VS Code has no headless mode. The script runs a second VS Code with its own profile in `/tmp/qvsc`, signed in to Copilot once by hand, so the user's own profile is not changed. The profile's path must be short, since VS Code puts a socket there and a longer path keeps it from starting. That VS Code loads `spike/b01/vscode-driver`, a small extension that sends each prompt in a new chat (`workbench.action.chat.open`), because `code chat` ignores `--user-data-dir` and opens the prompt in the user's own VS Code. The answers come from the chat session files in the profile's workspace storage, which also keep the context the model got with each prompt. Each check has its own workspace with the hook in `.github/hooks/quiver.json`.

The profile changes four things. It approves every tool call, since the agent otherwise waits for a click on terminal commands and 3 of 10 prompts never finished. It turns off workspace trust, so no dialog stops a run. It turns off personal skills and hooks: VS Code reads `~/.claude/skills` and `~/.copilot/hooks` whatever the profile, and in a first run the model acted on the user's own skills. And it sets `COPILOT_HOME` to a throwaway folder.

VS Code has two agent harnesses. The **Local** one is the default (`chat.defaultToCopilotHarness` is `false`, and an experiment can change it). The **Copilot** one runs the Copilot runtime (1.0.100-1 here), which the Copilot CLI shares. Unless named, results are for the Local harness. Each count is one pass, except where two are named.

### The 10 prompts

| Setup | Always-on | Fitting skills | Other skills loaded | `get_skill` for `reply-style` |
|---|---|---|---|---|
| Server only; `reply-style` assigned as always-on | 4 of 10 | 6 of 6 | 0 of 4 | 4 of 10 |
| Hook; `reply-style` not assigned | 10 of 10 | 6 of 6 | 0 of 4 | 0 of 10 |
| Hook, and `reply-style` also assigned as always-on | 10 of 10 | 5 of 6 | 0 of 4 | 6 of 10 |
| Server only, with the pointer in a user instructions file (below) | 8 of 10 | 6 of 6 | 0 of 4 | 8 of 10 |

VS Code defers MCP tools: the model sees their names and loads them with `tool_search`. It knows the skill IDs, so the server's instructions reach it, but it followed the always-on skill in only 4 of 10. With the hook and the pointer both active, it loaded the skill it already had in 6 of 10, as in Claude Code.

The hook's text reaches the model in the first user message, inside its `<context>` block, after `Additional instructions from hooks:`.

### Hook files and output

The hook fires when the first prompt starts a session. Its input is `timestamp`, `hook_event_name`, `session_id`, `transcript_path`, `source` (always `new`), `model` and `cwd`, and it runs in the workspace folder.

| Hook | Text reached the model |
|---|---|
| `.github/hooks/*.json`, flat form, `hookSpecificOutput.additionalContext` | 10 of 10 (the prompts above) |
| The same with the event key `sessionStart` | 2 of 2 |
| Claude Code's nested groups, with the matcher `startup` | 2 of 2; the matcher is ignored |
| `additionalContext` at the top level | 0 of 2 |
| Plain stdout | 0 of 2; the log says `Hook command returned non-JSON output` |
| Both outputs at once | 1 of 1, one copy |
| `.claude/settings.json` in the workspace | 0 of 2; 2 of 2 with `chat.useClaudeHooks` |
| A folder under `~/` named in `chat.hookFilesLocations` | 1 of 1 |
| An absolute path in `chat.hookFilesLocations` | 0 of 2; the setting accepts only relative and `~/` paths |
| `$COPILOT_HOME/hooks` | 0 of 1: the Local harness reads `~/.copilot/hooks` from the home folder |

The default locations are `.github/hooks`, `.claude/settings.local.json`, `.claude/settings.json`, `~/.copilot/hooks` and `~/.claude/settings.json`. The Claude files count only with `chat.useClaudeHooks`, which is off by default. With it on, by that list, a hook Quiver puts into `~/.claude/settings.json` for Claude Code would run in VS Code too; this was not measured.

### Size

20,000, 50,000 and 200,000 characters arrived whole, start, middle and end, in 2 of 2 each. The Copilot runtime documents a limit of 10 MiB for the combined `additionalContext`.

### Sub-agents

| Hooks | Sub-agent (`runSubagent`) followed the rule |
|---|---|
| `SessionStart` only | 0 of 2 |
| `SessionStart` and `SubagentStart` with `hookSpecificOutput.additionalContext` | 2 of 2 |

The main session followed the rule in 4 of 4. `SubagentStart` gets `agent_id` and `agent_type` (`default` here).

### Time and failures

The command runs with `/bin/sh` in the workspace folder. The documented default timeout is 30 seconds. A hook's errors go to the output channel "GitHub Copilot Chat Hooks". A `sleep 5` hook made the first request take 7.5 seconds, against 2.0 without a hook.

| Hook | Outcome |
|---|---|
| Prints the text, exits 1 | `NonBlockingError`; text not added |
| Prints the text, exits 2 | `Error`; text not added; the session went on |
| Prints the text, runs past a `timeout` of 2 seconds | Logged as timed out after 2 seconds, but VS Code waited for the command to end, 8 seconds; text not added |
| Binary does not exist | `/bin/sh: /nonexistent/quiver: No such file or directory`; the session went on |

### The Copilot harness

With `chat.defaultToCopilotHarness` on, sessions are stored under `$COPILOT_HOME/session-state` (default `~/.copilot`), and the hook output must be at the top level.

| Hook | Rule followed |
|---|---|
| `.github/hooks`, flat form, `hookSpecificOutput.additionalContext` only | 0 of 3 |
| The same with both outputs | 2 of 2 |
| Copilot's own form: `"version": 1`, the key `sessionStart`, `additionalContext` at the top level, `timeoutSec` | 2 of 2 |
| `$COPILOT_HOME/hooks`, both outputs | 2 of 2 |

With the key `SessionStart`, the input is `hook_event_name`, `session_id`, `timestamp`, `cwd`, `source` and `initial_prompt`; with `sessionStart`, the same in camelCase. Both harnesses take a file in the flat form with the key `SessionStart` whose command prints both outputs, and both read `~/.copilot/hooks` by default.

### MCP server

- With the server in the workspace's `.vscode/mcp.json`, it starts in the workspace folder.
- With the server in the user's `mcp.json`, it started in the home folder, 2 of 2, so Quiver serves no project there. With `"cwd": "${workspaceFolder}"` added, the server did not start at all, 2 of 2.
- VS Code sends the workspace folder as an MCP root when the server is initialized (`roots` with `listChanged`), seen with `spike/b01/quiver-spike`. Quiver does not read roots today.

The user's `mcp.json` is in the user folder: `~/Library/Application Support/Code/User/` on macOS, `%APPDATA%\Code\User\` on Windows, `~/.config/Code/User/` on Linux, with `Code - Insiders` in place of `Code` for VS Code Insiders. This follows VS Code's documented user folder and was not measured on Windows, Linux or Insiders.

### User instructions file

A file `quiver.instructions.md` in the profile's `prompts` folder, with `applyTo: "**"` and the line "Before your first reply, load the tools of the quiver MCP server and follow its instructions.", is added to every chat. The session lists it among the request's references. It raised always-on from 4 to 8 of 10. VS Code also reads `~/.copilot/instructions`.

### Trust and managed hooks

No hook asked for approval, but workspace trust was off, so trust was not measured. The documentation says agent-scoped hooks need a trusted workspace. From the settings: `chat.useHooks` (on by default, policy `ChatHooks`) turns the Local harness's hooks off, and `chat.hooks.allowManagedOnly` (policy `ChatAllowManagedHooksOnly`) allows only managed ones. For the Copilot harness, GitHub documents policy hooks in `/etc/github-copilot/policy.d/*.json` on macOS and Linux, `C:\ProgramData\GitHub\Copilot\policy.d\*.json` and `HKLM\Software\Policies\GitHub\Copilot` on Windows; they need no trust and cannot be turned off by the user. None of this was measured.

## Cursor

Measured on 2026-11-03 with the Cursor CLI (`cursor-agent` 2026.10.22-27dcf5f, model Auto), headless. The run script is `spike/b01/cursor-hooks-run.sh`, and `spike/b01/cursor-hooks-analyze.py` summarizes its output:

```sh
QUIVER=$PWD/bin/quiver spike/b01/cursor-hooks-run.sh /tmp/quiver-cursor-hooks
python3 spike/b01/cursor-hooks-analyze.py /tmp/quiver-cursor-hooks
```

Each check has its own workspace with the server in `.cursor/mcp.json` and the hook in `.cursor/hooks.json`, so `~/.cursor` is not changed. The Cursor app was not measured. Cursor already passes the server's `instructions` to the model, and always-on applied in 10 of 10 without a hook ([spike-mcp-clients.md](spike-mcp-clients.md)).

### The 10 prompts

With a `sessionStart` hook that prints `{"additional_context": "<text>"}`, and `reply-style` not assigned in Quiver:

| Always-on | Fitting skills | Other skills loaded | `get_skill` for `reply-style` |
|---|---|---|---|
| 10 of 10 | 6 of 6 | 0 of 4 | 0 of 10 |

### Hook files and events

| Hook | Ran | Rule followed |
|---|---|---|
| `.cursor/hooks.json`, `sessionStart` with `additional_context` | 2 of 2 | 2 of 2 |
| `.cursor/hooks.json`, `beforeSubmitPrompt` with `additional_context` | 0 of 2 | 0 of 2 |
| `.claude/settings.json`, Claude Code's `SessionStart` with `hookSpecificOutput.additionalContext` | 0 of 2 | 0 of 2 |
| None | – | 0 of 2 |

The file is `{"version": 1, "hooks": {"sessionStart": [{"command": "..."}]}}`. The hook's input is `conversation_id`, `generation_id`, `session_id`, `model`, `is_background_agent`, `hook_event_name`, `cursor_version`, `workspace_roots`, `user_email` and `transcript_path` (null in the CLI). The CLI ran `--trust`, so the documented rule that project hooks need a trusted workspace was not exercised.

From the documentation, not measured: user hooks live in `~/.cursor/hooks.json`; managed ones in `/Library/Application Support/Cursor/hooks.json` on macOS, `/etc/cursor/hooks.json` on Linux and WSL and `C:\ProgramData\Cursor\hooks.json` on Windows, and they take precedence over project and user hooks; the desktop app runs hook commands through PowerShell on Windows; `subagentStart` can only allow or deny a sub-agent; exit code 2 blocks and other failures let the action proceed. The documentation calls `sessionStart` fire-and-forget and does not say whether its context reaches the first prompt in the app.

## Antigravity

Measured on 2026-11-03 with the Antigravity CLI `agy` 1.4.12, default model, headless. The hook contract comes from the documentation built into `agy`. The run script is `spike/b01/antigravity-hooks-run.sh`, and `spike/b01/antigravity-hooks-analyze.py` summarizes its output:

```sh
QUIVER=$PWD/bin/quiver spike/b01/antigravity-hooks-run.sh /tmp/quiver-agy-hooks prompts forms size subagents
HOME=/private/tmp/agyhome agy    # sign in once, then /exit
AGY_HOME=/private/tmp/agyhome QUIVER=$PWD/bin/quiver spike/b01/antigravity-hooks-run.sh /tmp/quiver-agy-hooks global
python3 spike/b01/antigravity-hooks-analyze.py /tmp/quiver-agy-hooks
```

`agy` reads hooks, MCP servers and rules from customization roots: the global one, `~/.gemini/config/`, and the workspace's `.agents/` folder. Each check has its own workspace with the server in `.agents/mcp_config.json` and the hook in `.agents/hooks.json`, so `~/.gemini` is not changed. (TASK50 put the server into the global `mcp_config.json`, since it found no other place; the workspace's file works too.) The global check runs `agy` with a throwaway home folder, signed in once, and writes that home's files. The home must be given by its resolved path (`/private/tmp/...` on macOS); with `/tmp/...`, `agy` refused to read its own folder in it. Headless `agy` ends a session without an answer when a tool needs a permission it cannot ask for, so the runs allow Quiver's tools, `ls` and `pwd`.

### The 10 prompts

With a `SessionStart` hook that injects the skill as a system message, and `reply-style` not assigned in Quiver:

| Always-on | Fitting skills | Other skills loaded | `get_skill` for `reply-style` |
|---|---|---|---|
| 10 of 10 | 0 of 6 | 0 of 4 | 0 of 10 |

Without the hook, always-on applied in 0 of 10 and fitting skills in 1 of 6 ([spike-mcp-clients.md](spike-mcp-clients.md)); the model rarely reads the server's instructions, which `agy` writes to a file.

### Hook file and output

`hooks.json` maps a hook name to its events: `{"quiver": {"SessionStart": [{"type": "command", "command": "..."}]}}`. `SessionStart` runs once when a conversation starts; the events with a tool `matcher` are `PreToolUse` and `PostToolUse`, and the others are `PreInvocation`, `PostInvocation` and `Stop`. The command runs with `sh -c` (`cmd /c` on Windows) in the folder that holds `hooks.json`, and the default timeout is 30 seconds. The input is `conversationId`, `workspacePaths`, `transcriptPath`, `artifactDirectoryPath`, `modelName` and `lastUserInput`, in camelCase.

| Hook output | Rule followed |
|---|---|
| `{"injectSteps": [{"systemMessage": {"systemMessage": "<text>"}}]}` | 10 of 10 (the prompts above) |
| `{"injectSteps": [{"userMessage": "<text>"}]}` | 2 of 2 |
| `{"injectSteps": [{"ephemeralMessage": "<text>"}]}` | 2 of 2 |
| The system message from `PreInvocation`, which runs before every model call | 2 of 2 |
| Plain stdout | 0 of 2 |
| The system message, then exit 1 | 0 of 2 |
| No hook | 0 of 2 |

The model gets the text as a system message that begins "The following is a <SYSTEM_MESSAGE> not actually sent by the user."

### Size

The text is cut after about 48,900 bytes: the start stays, the rest is replaced with `<truncated N bytes>` and "NOTE: The output was truncated because it was too long." 20,000 bytes arrived whole; of 50,000 the last 1,092 bytes were dropped, and of 200,000 everything after the start. One session each.

### Sub-agents

A sub-agent started with `invoke_subagent` is a conversation of its own, and `SessionStart` runs for it too, with an empty `lastUserInput`. The sub-agent followed the rule in 2 of 2.

### Global hook and the app

A hook in the global `~/.gemini/config/hooks.json` ran and the rule was followed in 6 of 6 sessions. From the built-in documentation, not measured: the global `mcp_config.json` "applies to all sessions", and the customization root is the same for the CLI, Antigravity 2.0 and the IDE, which keep only their sessions in separate folders (`~/.gemini/antigravity-cli/`, `antigravity/`, `antigravity-ide/`). The Antigravity app was not installed.

## Global rules files

A pointer in a rules file that every session loads: "Before your first reply, load the tools of the quiver MCP server and follow its instructions." TASK50 put it into the workspace; here it is in the global file only, with `reply-style` assigned as always-on in Quiver and nothing in the workspace.

| Assistant and file | Always-on | Fitting skills | Other skills loaded | `get_skill` for `reply-style` |
|---|---|---|---|---|
| Codex 0.166.0, `$CODEX_HOME/AGENTS.md` | 10 of 10 | 6 of 6 | 0 of 4 | 10 of 10 |
| Antigravity `agy` 1.4.12, `~/.gemini/config/GEMINI.md` | 7 of 10 | 3 of 6 | 0 of 4 | 10 of 10 |

In Antigravity the model loaded `reply-style` in 10 of 10, and 3 sessions ended without an answer when the model went on to read a skill file outside the workspace or to run `find`, which headless `agy` cannot allow. The same pointer in `~/.gemini/config/AGENTS.md` and in `~/.gemini/GEMINI.md` worked too, 2 of 2 each. Codex reads `AGENTS.override.md` instead of `AGENTS.md` in `$CODEX_HOME` when it exists (from source).

A project's own file adds to the global one: with "Start every reply with the line [global-rule]." in the global file and "End every reply with the line [project-rule]." in the workspace's `AGENTS.md`, both were followed in 2 of 2 in Codex and 6 of 6 in Antigravity. Codex joins the two with a `--- project-doc ---` line (from source).

## One text in several hook commands

An assistant passes only so much text from one hook command (Claude Code 10,000 characters, Antigravity about 48,900 bytes), and always-on skills must arrive whole whatever their size. Quiver therefore splits its text (the always-on skills in full, then the ID and description of every other skill) over several hook commands of one event, each printing one part (`quiver hook session-start <assistant> [part]`). Measured on 2026-11-04 with throwaway workspaces, and then with the real `quiver install`:

| Assistant | One command | Several commands | Order of the parts |
|---|---|---|---|
| Claude Code 2.1.308 | 10,000 characters | 5 commands of 9,900 and 12 of 9,900 (118,800 characters) arrived whole | The order the commands finish. Parts that sleep for different times arrived in finishing order, never in file order. |
| Cursor CLI 2026.10.22 | 200,000 characters arrived whole from one command | 5 commands of 9,900 arrived whole | File order, also when the first commands are slower: they run one after another. |
| Antigravity `agy` 1.4.12 | 40,000 bytes arrived whole, 60,000 lost its end | 3 commands of 30,000 and 6 of 30,000 (180,000 bytes) arrived whole | File order, also when the first commands are slower. |

- The limit is per command everywhere, so the text of any size reaches the model; only the context window bounds it.
- Because Claude Code runs the commands at the same time, a part waits 50 ms times its number minus one before it prints (later: 50 ms for each of the first 20 parts, 10 ms after that, 3 s at most, so that hundreds of parts still arrive inside the 10 s hook limit). Without the wait, 2 of 3 runs added the parts in the order 5, 6, 3, 4, 1, 2; with it, 8 of 8 runs added them in order, while 8 sessions ran at once. Every part names itself (`part="2/4"`), and a section that continues from the previous part says so, so a model can put the parts in order when they arrive out of it.
- With the real binary and a new `quiver install`, in Claude Code: 6 always-on skills of about 4,100 characters each (4 parts) and 15 other skills were all in the context in 3 of 3 sessions, and so was a project's own always-on skill. A sub-agent got the same text from its `SubagentStart` commands in 3 of 3.
- Cursor (2 parts) and Antigravity (4 parts) with 12 always-on skills of about 12,000 characters each: all 12 skills arrived, in order, in 1 of 1 session each.
- A hook command takes about 10 ms plus the 50 ms steps; four commands finish within 0.2 s.
- Codex: `additionalContextLimit` 0 turns its limit off and 50,000 bytes arrived whole from one command (TASK52.2); Quiver splits at 40,000 bytes there. The Codex and VS Code hooks with several commands were not run in a session here; Codex needs every command trusted, which `quiver install` does through `codex app-server`.

### Many parts at once

Measured on 2026-11-04 with the Quiver binary and a library of 2,000 skills (47 MB; 100 always-on, 2.1 MB of text, 231 parts of 10,000 characters), starting all 231 commands at once and killing each after 10 s, as Claude Code's hook timeout does:

| | Delivered | Killed | Wall time |
|---|---|---|---|
| Each command reads the library and builds the text | 48 of 231 | 183 | 30 s |
| Part 1 builds and leaves the parts in `~/.quiver/cache/hook`; the others read them | 231 of 231 (2,055,674 characters) | 0 | 3.5 s |

The same library with 500 projects: `quiver add` took 21 s while it rebuilt every project's text to count its parts, and 1 s when each skill's text, size and catalog line is built once and parts are only counted.

### Real sessions after the delivery changes

Measured on 2026-11-04 with the final binary, `quiver install`'s hooks given to each assistant through its own per-run or per-project hook file, a throwaway Quiver home, and unique random tokens at the start and end of each skill:

| Assistant | Text | Result |
|---|---|---|
| Claude Code 2.1.308 | 6 always-on skills (4 parts), a project always-on skill, 15 other skills | 7 of 7 tokens and all 15 catalog entries in 3 of 3 sessions; parts 2 to 4 read the cache part 1 wrote |
| Claude Code, sub-agent | the same | 7 of 7 tokens in the sub-agent's context, from 4 `SubagentStart` commands |
| Claude Code | 25 always-on skills of 9,000 characters and a project skill (26 parts) | 26 of 26 tokens; part 1 took 137 ms |
| Claude Code | one always-on skill with an unreadable folder | the other 4 skills arrived, and the model quoted the notice naming the skill and the error |
| Cursor CLI | the 26 skills above (3 commands) | 26 of 26 |
| Antigravity `agy` | the 26 skills above (7 commands) | 26 of 26 |
| Codex 0.166.0 (`gpt-6.1-sol`) | the 26 skills above, 7 commands for each of `SessionStart` and `SubagentStart` | 26 of 26 distinct tokens, every code correct; `quiver install codex` trusted all 14 hooks through `codex app-server` |

VS Code with several commands was not run in a session.
