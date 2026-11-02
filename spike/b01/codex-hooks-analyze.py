#!/usr/bin/env python3
"""usage: CODEX_HOME=<home> codex-hooks-analyze.py <dir>

Summarizes codex-hooks-run.sh. Reads each session's rollout under
$CODEX_HOME/sessions for the text the hooks added. For the prompt runs:
always-on, fitting and other skills, and get_skill calls for reply-style. For
single runs: seconds, hook firings, which tokens the added text kept, whether
it was cut and saved to a file, and the start of the answer. For the chains,
sub-agents, TUI sessions and project folders: the sources the hook fired on and
the marker in each answer.
"""
import json
import os
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
sessions = pathlib.Path(os.environ["CODEX_HOME"]) / "sessions"


def events(path):
    for line in path.read_text(errors="replace").splitlines():
        try:
            yield json.loads(line)
        except ValueError:
            continue


def metas():
    """The session_meta of every rollout: id, parent, cwd, path."""
    out = []
    for f in sessions.rglob("rollout-*.jsonl"):
        for e in events(f):
            if e.get("type") == "session_meta":
                p = e["payload"]
                if p.get("id") in f.name:
                    out.append(dict(p, path=f))
                    break
    return out


META = metas()


def rollout(thread):
    return next((m["path"] for m in META if m["id"] == thread), None)


def messages(path):
    """(role, text, from a hook) of each message in a rollout."""
    out = []
    for e in events(path):
        p = e.get("payload") or {}
        if e.get("type") == "response_item" and p.get("type") == "message":
            text = "".join(c.get("text", "") for c in p["content"])
            kinds = (p.get("internal_chat_message_metadata_passthrough") or {}).get("content_item_kinds") or []
            out.append((p["role"], text, "hooks.additional_context" in kinds))
    return out


def hook_texts(path):
    return [t for role, t, hook in messages(path) if hook]


def exec_run(path):
    """Thread id, Quiver tool calls, shell commands and answer of a codex exec --json file."""
    thread, tools, commands, msgs = None, [], [], []
    for e in events(path):
        item = e.get("item") or {}
        if e.get("type") == "thread.started":
            thread = e["thread_id"]
        if e.get("type") != "item.completed":
            continue
        if item.get("type") == "mcp_tool_call" and item.get("server") == "quiver":
            arg = (item.get("arguments") or {}).get("id") or ""
            tools.append(f"{item['tool']}({arg})")
        elif item.get("type") == "command_execution":
            commands.append(item.get("command", ""))
        elif item.get("type") == "agent_message":
            msgs.append(item["text"])
    return thread, tools, commands, "\n".join(msgs)


def hook_log(d):
    log = d / "hook.log"
    return [e for e in events(log)] if log.exists() else []


def tokens(text):
    return sorted(set(re.findall(r"(?:START|MID|END)-TOKEN-\w+", text)))


def prompts(d):
    always = fitting = other = pointer = 0
    for f in sorted((d / "out").glob("*.jsonl")):
        _, tools, _, answer = exec_run(f)
        always += "[always-on]" in answer
        pointer += "get_skill(reply-style)" in tools
        if f.stem.startswith("fit"):
            fitting += "[used:" in answer
        else:
            other += any(t.startswith("get_skill(") and "reply-style" not in t for t in tools)
    print(f"{d.name:24} always-on {always}/10  fitting {fitting}/6  other {other}/4  get_skill(reply-style) {pointer}/10")


def single(d, out="out.jsonl"):
    thread, _, commands, answer = exec_run(d / out)
    secs = (d / "seconds.txt").read_text().strip() if (d / "seconds.txt").exists() else "?"
    path = rollout(thread)
    added = hook_texts(path) if path else []
    kept = tokens("\n".join(added))
    cut = any("Full hook output saved to" in t for t in added)
    size = sum(len(t.encode()) for t in added)
    read = any("hook_outputs" in c for c in commands)
    errs = [l for l in (d / "err.txt").read_text().splitlines() if "hook" in l.lower()] if (d / "err.txt").exists() else []
    print(f"{d.name:24} {secs:>3}s firings={len(hook_log(d))} context={size}B cut={cut} kept={kept} "
          f"read-file={read} marker={'[always-on]' in answer} answer={tokens(answer) or answer[:60]!r} {errs[:2]}")


def chain(d):
    """A forked rollout holds only its new items and names its parent, so the
    hook texts in context are the parent's plus its own."""
    sources = [e.get("source") for e in hook_log(d)]
    thread, _, _, answer = exec_run(d / "2.jsonl")
    meta = next(m for m in META if m["id"] == thread)
    texts = hook_texts(meta["path"])
    if meta.get("forked_from_id"):
        texts = hook_texts(rollout(meta["forked_from_id"])) + texts
    in_context = re.findall(r"CONTEXT-TOKEN-\w+", "\n".join(texts))
    quoted = re.findall(r"CONTEXT-TOKEN-\w+", answer)
    print(f"{d.name:24} sources={sources} hook-texts={len(texts)} in-context={in_context} "
          f"quoted={quoted} marker={'[always-on]' in answer}")


def subagent(d):
    thread, _, _, answer = exec_run(d / "out.jsonl")
    events_ = [e["hook_event_name"] for e in hook_log(d)]
    fork = []
    for e in events(rollout(thread)):
        p = e.get("payload") or {}
        if p.get("type") == "function_call" and p.get("name") == "spawn_agent":
            fork.append(json.loads(p.get("arguments") or "{}").get("fork_turns", "default"))
    kids = [m for m in META if m.get("parent_thread_id") == thread]
    print(f"{d.name:24} hooks={events_} fork_turns={fork} parent-marker={'[always-on]' in answer}")
    for k in kids:
        msgs = messages(k["path"])
        last = next((t for role, t, _ in reversed(msgs) if role == "assistant"), "")
        copies = sum(1 for role, t, hook in msgs if hook)
        print(f"{'':24} sub-agent hook-texts={copies} marker={'[always-on]' in last}")


def interactive(d):
    print(f"{d.name:24} sources={[e.get('source') for e in hook_log(d)]} {(d / 'first-prompt.txt').read_text().strip()}")
    for m in sorted((m for m in META if m.get("cwd") == str(d)), key=lambda m: m["timestamp"]):
        # The summary itself is encrypted; the replacement history shows
        # which earlier messages compaction kept.
        compacted = [e for e in events(m["path"]) if e.get("type") == "compacted"]
        for c in compacted:
            kept = "House reply style" in json.dumps(c["payload"].get("replacement_history"))
            print(f"{'':24} {m['id'][-8:]} compaction kept the hook text: {kept}")
        prompt = None
        for role, text, hook in messages(m["path"]):
            if role == "user" and text.startswith("Suggest"):
                prompt = text
            elif role == "assistant" and prompt:
                print(f"{'':24} {m['id'][-8:]} hook-texts={len(hook_texts(m['path']))} {prompt[:32]!r} marker={'[always-on]' in text}")
                prompt = None


def tui_once(d):
    """A TUI run with one prompt, then maybe a headless run in the same folder."""
    log = hook_log(d)
    screens = sorted(p.stem for p in d.glob("screen-*.txt"))
    print(f"{d.name:24} firings={len(log)} {(d / 'first-prompt.txt').read_text().strip()} screens={screens}")
    for m in sorted((m for m in META if m.get("cwd") == str(d)), key=lambda m: m["timestamp"]):
        msgs = messages(m["path"])
        last = next((t for role, t, _ in reversed(msgs) if role == "assistant"), "")
        print(f"{'':24} {m['id'][-8:]} {m.get('originator')} hook-texts={len(hook_texts(m['path']))} marker={'[always-on]' in last}")


for d in sorted(p for p in root.iterdir() if p.is_dir()):
    n = d.name
    if n.startswith("prompts-"):
        prompts(d)
    elif n.startswith("chain-"):
        chain(d)
    elif n.startswith("sub-"):
        subagent(d)
    elif n.startswith("tui-"):
        interactive(d)
    elif n == "trust":
        print(f"{n:24} hashes: {(d / 'hashes.txt').read_text().split().count('same')} same, "
              f"{(d / 'hashes.txt').read_text().split().count('differs')} differ; "
              f"app server trusted: {[l.rsplit(':', 3)[1:] for l in (d / 'app-server.txt').read_text().split()[1::2]]}")
        tui_once(d)
    elif (d / "first-prompt.txt").exists():
        tui_once(d)
    elif (d / "out.jsonl").exists():
        single(d)
