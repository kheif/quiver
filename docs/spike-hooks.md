# Spike: session-start hooks per assistant

Always-on skills reach the model through the MCP server's `instructions`, and only some assistants put those into the prompt (see [spike-mcp-clients.md](spike-mcp-clients.md)). A session-start hook runs a command when a session starts and adds its output to the model's context, so the skill text is there without the model deciding to load it. This spike checks, per assistant, whether such a hook exists and how it behaves.

**Status:** Claude Code done on 2026-11-02. Codex, VS Code Copilot, Cursor and Antigravity are not measured yet.

**Delivery levels.** Each assistant gets one:

- **hook:** the always-on skills' text is added at session start.
- **instructions:** the model gets Quiver's `instructions` up front and loads the always-on skills itself.
- **rules file:** a pointer in a file the assistant always loads makes the model load Quiver.
- **on request:** skills load only when the prompt asks for them.

## Summary

| Assistant | Hook event | User and managed config file | Input on stdin | Output format | Output reaches the model | Size limit | After compaction | Sub-agents | Trust or approval prompt | Command form on Windows | Global rules file | Delivery level |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Claude Code 2.1.308 | `SessionStart`, matcher on `source` | `~/.claude/settings.json` or `$CLAUDE_CONFIG_DIR/settings.json`; `managed-settings.json` in a system folder | JSON with `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `source`; interactive sessions add `model` and `scratchpad_dir` | Plain stdout or `additionalContext` | Yes, 10 of 10 | 10,000 characters per hook command; above that, a file path and a 2 KB preview | Fires again with `source` `compact` | `SessionStart` does not reach them; `SubagentStart` does | The folder's trust dialog, in interactive sessions only; no approval per hook | Documented: Git Bash, or PowerShell without Git Bash; `shell` chooses | Documented: `~/.claude/CLAUDE.md` | **hook** |

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
